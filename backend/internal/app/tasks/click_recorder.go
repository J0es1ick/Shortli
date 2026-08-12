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
	Pending  int64 `json:"pending"`
	Queued   int64 `json:"queued"`
	Recorded int64 `json:"recorded"`
	Retried  int64 `json:"retried"`
}

type ClickRecorder struct {
	repo      clickStore
	spoolPath string
	workers   int
	wake      chan struct{}
	stop      chan struct{}
	force     chan struct{}
	closeOnce sync.Once
	forceOnce sync.Once
	wg        sync.WaitGroup
	accepting atomic.Bool
	queued    atomic.Int64
	recorded  atomic.Int64
	retried   atomic.Int64
}

type clickStore interface {
	RecordClickContext(context.Context, *models.ClickEvent) error
}

func NewClickRecorder(repo clickStore, spoolPath string, workers int) (*ClickRecorder, error) {
	if workers < 1 {
		workers = 1
	}
	if err := os.MkdirAll(spoolPath, 0o750); err != nil {
		return nil, fmt.Errorf("create click spool: %w", err)
	}
	if err := recoverProcessingFiles(spoolPath); err != nil {
		return nil, err
	}

	recorder := &ClickRecorder{
		repo: repo, spoolPath: spoolPath, workers: workers,
		wake: make(chan struct{}, 1), stop: make(chan struct{}), force: make(chan struct{}),
	}
	recorder.accepting.Store(true)
	for i := 0; i < workers; i++ {
		recorder.wg.Add(1)
		go recorder.worker()
	}
	recorder.signal()
	return recorder, nil
}

func (r *ClickRecorder) Submit(event *models.ClickEvent) error {
	if !r.accepting.Load() {
		return errors.New("click recorder is shutting down")
	}
	eventKey, err := utils.GenerateRandomString(24)
	if err != nil {
		return fmt.Errorf("generate click event key: %w", err)
	}
	event.EventKey = eventKey

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode click event: %w", err)
	}
	pendingPath := filepath.Join(r.spoolPath, eventKey+".pending.json")
	tempPath := pendingPath + ".tmp"
	if err := writeAndSync(tempPath, body); err != nil {
		return fmt.Errorf("persist click event: %w", err)
	}
	if err := os.Rename(tempPath, pendingPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("commit click event: %w", err)
	}
	r.queued.Add(1)
	r.signal()
	return nil
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
	pending, _ := r.pendingCount()
	return ClickRecorderStats{
		Pending: pending, Queued: r.queued.Load(),
		Recorded: r.recorded.Load(), Retried: r.retried.Load(),
	}
}

func (r *ClickRecorder) Close(ctx context.Context) error {
	r.closeOnce.Do(func() {
		r.accepting.Store(false)
		close(r.stop)
		r.signal()
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
		return fmt.Errorf("click recorder stopped with durable events pending: %w", ctx.Err())
	}
}

func (r *ClickRecorder) worker() {
	defer r.wg.Done()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		processed, err := r.processOne()
		if err != nil {
			log.Printf("click recorder: %v", err)
			r.retried.Add(1)
			select {
			case <-time.After(time.Second):
			case <-r.force:
				return
			}
		}
		if processed {
			continue
		}

		select {
		case <-r.wake:
		case <-ticker.C:
		case <-r.stop:
			pending, _ := r.pendingCount()
			if pending == 0 {
				return
			}
		case <-r.force:
			return
		}

		if !r.accepting.Load() {
			pending, _ := r.pendingCount()
			if pending == 0 {
				return
			}
		}
	}
}

func (r *ClickRecorder) processOne() (bool, error) {
	entries, err := os.ReadDir(r.spoolPath)
	if err != nil {
		return false, fmt.Errorf("scan spool: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".pending.json") {
			continue
		}
		pendingPath := filepath.Join(r.spoolPath, entry.Name())
		processingPath := strings.TrimSuffix(pendingPath, ".pending.json") + ".processing.json"
		if err := os.Rename(pendingPath, processingPath); err != nil {
			continue
		}

		body, err := os.ReadFile(processingPath)
		if err != nil {
			_ = os.Rename(processingPath, pendingPath)
			return true, fmt.Errorf("read queued event: %w", err)
		}
		var event models.ClickEvent
		if err := json.Unmarshal(body, &event); err != nil {
			quarantinePath := strings.TrimSuffix(processingPath, ".processing.json") + ".invalid.json"
			_ = os.Rename(processingPath, quarantinePath)
			return true, fmt.Errorf("quarantined invalid queued event %s: %w", entry.Name(), err)
		}

		recordContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = r.repo.RecordClickContext(recordContext, &event)
		cancel()
		if err != nil {
			_ = os.Rename(processingPath, pendingPath)
			return true, err
		}
		if err := os.Remove(processingPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return true, fmt.Errorf("remove recorded event: %w", err)
		}
		r.recorded.Add(1)
		return true, nil
	}
	return false, nil
}

func (r *ClickRecorder) pendingCount() (int64, error) {
	entries, err := os.ReadDir(r.spoolPath)
	if err != nil {
		return 0, err
	}
	var count int64
	for _, entry := range entries {
		if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".pending.json") || strings.HasSuffix(entry.Name(), ".processing.json")) {
			count++
		}
	}
	return count, nil
}

func (r *ClickRecorder) signal() {
	select {
	case r.wake <- struct{}{}:
	default:
	}
}

func recoverProcessingFiles(spoolPath string) error {
	entries, err := os.ReadDir(spoolPath)
	if err != nil {
		return fmt.Errorf("scan click spool during recovery: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".processing.json") {
			continue
		}
		oldPath := filepath.Join(spoolPath, entry.Name())
		newPath := strings.TrimSuffix(oldPath, ".processing.json") + ".pending.json"
		if err := os.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf("recover click event %s: %w", entry.Name(), err)
		}
	}
	return nil
}
