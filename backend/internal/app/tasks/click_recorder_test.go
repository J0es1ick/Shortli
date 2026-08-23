package tasks

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/J0es1ick/shortli/internal/models"
)

type fakeClickStore struct {
	mu     sync.Mutex
	fail   bool
	events map[string]int
}

func (s *fakeClickStore) RecordClickContext(_ context.Context, event *models.ClickEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fail {
		return errors.New("database unavailable")
	}
	s.events[event.EventKey]++
	return nil
}

func TestClickRecorderRecoversDurableEventAfterRestart(t *testing.T) {
	spool := t.TempDir()
	failingStore := &fakeClickStore{fail: true, events: map[string]int{}}
	first, err := NewClickRecorder(failingStore, spool, 1, 1<<20)
	if err != nil {
		t.Fatalf("create first recorder: %v", err)
	}
	if err := first.Submit(&models.ClickEvent{URLID: 42, ClickedAt: time.Now()}); err != nil {
		t.Fatalf("submit event: %v", err)
	}

	closeContext, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := first.Close(closeContext); err == nil {
		t.Fatal("expected close to preserve an event while storage is unavailable")
	}
	if stats := first.Stats(); stats.Pending != 1 {
		t.Fatalf("pending after forced stop = %d, want 1", stats.Pending)
	}

	healthyStore := &fakeClickStore{events: map[string]int{}}
	second, err := NewClickRecorder(healthyStore, spool, 1, 1<<20)
	if err != nil {
		t.Fatalf("create recovered recorder: %v", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for second.Stats().Pending != 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if stats := second.Stats(); stats.Pending != 0 || stats.Recorded != 1 {
		t.Fatalf("recovery stats = %+v, want one recorded and none pending", stats)
	}
	shutdownContext, shutdownCancel := context.WithTimeout(context.Background(), time.Second)
	defer shutdownCancel()
	if err := second.Close(shutdownContext); err != nil {
		t.Fatalf("close recovered recorder: %v", err)
	}

	healthyStore.mu.Lock()
	defer healthyStore.mu.Unlock()
	if len(healthyStore.events) != 1 {
		t.Fatalf("recorded unique events = %d, want 1", len(healthyStore.events))
	}
	for _, count := range healthyStore.events {
		if count != 1 {
			t.Fatalf("event recorded %d times, want once", count)
		}
	}
}

func TestClickRecorderRejectsEventsBeyondCapacity(t *testing.T) {
	store := &fakeClickStore{fail: true, events: map[string]int{}}
	recorder, err := NewClickRecorder(store, t.TempDir(), 1, 1)
	if err != nil {
		t.Fatalf("create recorder: %v", err)
	}
	if err := recorder.Submit(&models.ClickEvent{URLID: 42, ClickedAt: time.Now()}); !errors.Is(err, ErrClickSpoolFull) {
		t.Fatalf("Submit() error = %v, want ErrClickSpoolFull", err)
	}
	stats := recorder.Stats()
	if stats.Pending != 0 || stats.Dropped != 1 || stats.MaxBytes != 1 {
		t.Fatalf("capacity stats = %+v", stats)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := recorder.Close(ctx); err != nil {
		t.Fatalf("close recorder: %v", err)
	}
}
