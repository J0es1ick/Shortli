package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/J0es1ick/shortli/internal/models"
	"github.com/J0es1ick/shortli/internal/utils"
)

type ClickRecorderStats struct {
	Pending        int64 `json:"pending"`
	PendingBytes   int64 `json:"pending_bytes"`
	MaxBytes       int64 `json:"max_bytes"`
	Buffered       int64 `json:"buffered"`
	BufferCapacity int64 `json:"buffer_capacity"`
	Queued         int64 `json:"queued"`
	Recorded       int64 `json:"recorded"`
	Retried        int64 `json:"retried"`
	Dropped        int64 `json:"dropped"`
}

var (
	ErrClickSpoolFull      = errors.New("click spool capacity exceeded")
	ErrClickQueueFull      = errors.New("click writer queue capacity exceeded")
	ErrClickRecorderClosed = errors.New("click recorder is shutting down")
)

type queuedClick struct {
	eventKey string
	body     []byte
}

type ClickRecorder struct {
	repo           clickStore
	spoolPath      string
	maxBytes       int64
	bufferCapacity int64
	writeQueue     chan queuedClick
	processQueue   chan string
	stop           chan struct{}
	force          chan struct{}
	writerDone     chan struct{}
	closeOnce      sync.Once
	forceOnce      sync.Once
	submitMu       sync.RWMutex
	wg             sync.WaitGroup
	accepting      atomic.Bool
	pending        atomic.Int64
	pendingBytes   atomic.Int64
	allocatedBytes atomic.Int64
	queued         atomic.Int64
	recorded       atomic.Int64
	retried        atomic.Int64
	dropped        atomic.Int64
}

type clickStore interface {
	RecordClickContext(context.Context, *models.ClickEvent) error
}

func NewClickRecorder(repo clickStore, spoolPath string, workers int, maxBytes int64, bufferCapacity int) (*ClickRecorder, error) {
	if workers < 1 {
		workers = 1
	}
	if maxBytes < 1 {
		return nil, errors.New("click spool capacity must be positive")
	}
	if bufferCapacity < 1 {
		return nil, errors.New("click writer queue capacity must be positive")
	}
	if err := os.MkdirAll(spoolPath, 0o750); err != nil {
		return nil, fmt.Errorf("create click spool: %w", err)
	}

	pendingPaths, pendingBytes, err := recoverSpool(spoolPath)
	if err != nil {
		return nil, err
	}
	processCapacity := len(pendingPaths) + bufferCapacity
	recorder := &ClickRecorder{
		repo:           repo,
		spoolPath:      spoolPath,
		maxBytes:       maxBytes,
		bufferCapacity: int64(bufferCapacity),
		writeQueue:     make(chan queuedClick, bufferCapacity),
		processQueue:   make(chan string, processCapacity),
		stop:           make(chan struct{}),
		force:          make(chan struct{}),
		writerDone:     make(chan struct{}),
	}
	recorder.pending.Store(int64(len(pendingPaths)))
	recorder.pendingBytes.Store(pendingBytes)
	recorder.allocatedBytes.Store(pendingBytes)
	for _, path := range pendingPaths {
		recorder.processQueue <- path
	}

	recorder.accepting.Store(true)
	recorder.wg.Add(1)
	go recorder.writer()
	for i := 0; i < workers; i++ {
		recorder.wg.Add(1)
		go recorder.worker()
	}
	return recorder, nil
}

func (r *ClickRecorder) Submit(event *models.ClickEvent) error {
	eventKey, err := utils.GenerateRandomString(24)
	if err != nil {
		return fmt.Errorf("generate click event key: %w", err)
	}
	event.EventKey = eventKey
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode click event: %w", err)
	}

	r.submitMu.RLock()
	defer r.submitMu.RUnlock()
	if !r.accepting.Load() {
		return ErrClickRecorderClosed
	}
	if !r.reserveBytes(int64(len(body))) {
		r.dropped.Add(1)
		return ErrClickSpoolFull
	}

	item := queuedClick{eventKey: eventKey, body: body}
	select {
	case r.writeQueue <- item:
		r.queued.Add(1)
		return nil
	default:
		r.allocatedBytes.Add(-int64(len(body)))
		r.dropped.Add(1)
		return ErrClickQueueFull
	}
}

func (r *ClickRecorder) reserveBytes(size int64) bool {
	for {
		allocated := r.allocatedBytes.Load()
		if size > r.maxBytes-allocated {
			return false
		}
		if r.allocatedBytes.CompareAndSwap(allocated, allocated+size) {
			return true
		}
	}
}

func writeAndSync(path string, body []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(body); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func (r *ClickRecorder) Stats() ClickRecorderStats {
	return ClickRecorderStats{
		Pending: r.pending.Load(), PendingBytes: r.pendingBytes.Load(), MaxBytes: r.maxBytes,
		Buffered: int64(len(r.writeQueue)), BufferCapacity: r.bufferCapacity,
		Queued: r.queued.Load(), Recorded: r.recorded.Load(),
		Retried: r.retried.Load(), Dropped: r.dropped.Load(),
	}
}

func (r *ClickRecorder) Close(ctx context.Context) error {
	r.closeOnce.Do(func() {
		r.submitMu.Lock()
		r.accepting.Store(false)
		close(r.stop)
		r.submitMu.Unlock()
	})

	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		r.forceOnce.Do(func() { close(r.force) })
		<-done
		return fmt.Errorf("click recorder stopped before all events were persisted: %w", ctx.Err())
	}
}

func (r *ClickRecorder) writer() {
	defer r.wg.Done()
	defer close(r.writerDone)

	for {
		select {
		case item := <-r.writeQueue:
			if !r.persistUntilForced(item) {
				r.discardBuffered()
				return
			}
		case <-r.stop:
			for {
				select {
				case item := <-r.writeQueue:
					if !r.persistUntilForced(item) {
						r.discardBuffered()
						return
					}
				default:
					return
				}
			}
		case <-r.force:
			r.discardBuffered()
			return
		}
	}
}

func (r *ClickRecorder) persistUntilForced(item queuedClick) bool {
	for {
		select {
		case <-r.force:
			r.discard(item)
			return false
		default:
		}

		pendingPath, err := r.persist(item)
		if err == nil {
			select {
			case r.processQueue <- pendingPath:
				return true
			case <-r.force:
				return false
			}
		}

		log.Printf("click writer: %v", err)
		r.retried.Add(1)
		select {
		case <-time.After(250 * time.Millisecond):
		case <-r.force:
			r.discard(item)
			return false
		}
	}
}

func (r *ClickRecorder) persist(item queuedClick) (string, error) {
	pendingPath := filepath.Join(r.spoolPath, item.eventKey+".pending.json")
	tempPath := pendingPath + ".tmp"
	if err := writeAndSync(tempPath, item.body); err != nil {
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("persist click event: %w", err)
	}
	if err := os.Rename(tempPath, pendingPath); err != nil {
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("commit click event: %w", err)
	}
	size := int64(len(item.body))
	r.pending.Add(1)
	r.pendingBytes.Add(size)
	return pendingPath, nil
}

func (r *ClickRecorder) discard(item queuedClick) {
	r.allocatedBytes.Add(-int64(len(item.body)))
	r.dropped.Add(1)
}

func (r *ClickRecorder) discardBuffered() {
	for {
		select {
		case item := <-r.writeQueue:
			r.discard(item)
		default:
			return
		}
	}
}

func (r *ClickRecorder) worker() {
	defer r.wg.Done()
	for {
		select {
		case pendingPath := <-r.processQueue:
			if !r.processUntilForced(pendingPath) {
				return
			}
		case <-r.writerDone:
			for {
				select {
				case pendingPath := <-r.processQueue:
					if !r.processUntilForced(pendingPath) {
						return
					}
				default:
					return
				}
			}
		case <-r.force:
			return
		}
	}
}

func (r *ClickRecorder) processUntilForced(pendingPath string) bool {
	for {
		select {
		case <-r.force:
			return false
		default:
		}

		retry, err := r.processPath(pendingPath)
		if err != nil {
			log.Printf("click recorder: %v", err)
		}
		if !retry {
			return true
		}

		r.retried.Add(1)
		select {
		case <-time.After(time.Second):
		case <-r.force:
			return false
		}
	}
}

func (r *ClickRecorder) processPath(pendingPath string) (bool, error) {
	processingPath := strings.TrimSuffix(pendingPath, ".pending.json") + ".processing.json"
	if err := os.Rename(pendingPath, processingPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return true, fmt.Errorf("claim queued event: %w", err)
	}

	body, err := os.ReadFile(processingPath)
	if err != nil {
		_ = os.Rename(processingPath, pendingPath)
		return true, fmt.Errorf("read queued event: %w", err)
	}
	var event models.ClickEvent
	if err := json.Unmarshal(body, &event); err != nil {
		quarantinePath := strings.TrimSuffix(processingPath, ".processing.json") + ".invalid.json"
		if renameErr := os.Rename(processingPath, quarantinePath); renameErr != nil {
			_ = os.Rename(processingPath, pendingPath)
			return true, fmt.Errorf("quarantine invalid queued event: %w", renameErr)
		}
		r.pending.Add(-1)
		return false, fmt.Errorf("quarantined invalid queued event: %w", err)
	}

	recordContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = r.repo.RecordClickContext(recordContext, &event)
	cancel()
	if err != nil {
		_ = os.Rename(processingPath, pendingPath)
		return true, err
	}
	if err := os.Remove(processingPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		_ = os.Rename(processingPath, pendingPath)
		return true, fmt.Errorf("remove recorded event: %w", err)
	}

	size := int64(len(body))
	r.pending.Add(-1)
	r.pendingBytes.Add(-size)
	r.allocatedBytes.Add(-size)
	r.recorded.Add(1)
	return false, nil
}

func recoverSpool(spoolPath string) ([]string, int64, error) {
	entries, err := os.ReadDir(spoolPath)
	if err != nil {
		return nil, 0, fmt.Errorf("scan click spool during recovery: %w", err)
	}

	pendingPaths := make([]string, 0, len(entries))
	var pendingBytes int64
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(spoolPath, entry.Name())
		if strings.HasSuffix(entry.Name(), ".pending.json.tmp") {
			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				return nil, 0, fmt.Errorf("remove incomplete click event %s: %w", entry.Name(), err)
			}
			continue
		}

		info, err := entry.Info()
		if err != nil {
			return nil, 0, fmt.Errorf("inspect click spool entry %s: %w", entry.Name(), err)
		}
		pendingBytes += info.Size()
		if strings.HasSuffix(entry.Name(), ".processing.json") {
			pendingPath := strings.TrimSuffix(path, ".processing.json") + ".pending.json"
			if err := os.Rename(path, pendingPath); err != nil {
				return nil, 0, fmt.Errorf("recover click event %s: %w", entry.Name(), err)
			}
			path = pendingPath
		}
		if strings.HasSuffix(path, ".pending.json") {
			pendingPaths = append(pendingPaths, path)
		}
	}
	sort.Strings(pendingPaths)
	return pendingPaths, pendingBytes, nil
}
