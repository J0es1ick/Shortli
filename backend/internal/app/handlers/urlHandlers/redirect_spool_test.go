package urlHandlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/J0es1ick/shortli/internal/app/middleware"
	"github.com/J0es1ick/shortli/internal/app/tasks"
	"github.com/J0es1ick/shortli/internal/config"
	"github.com/J0es1ick/shortli/internal/models"
)

type unavailableClickStore struct{}

func (unavailableClickStore) RecordClickContext(context.Context, *models.ClickEvent) error {
	return errors.New("database unavailable")
}

func TestRedirectRemainsResponsiveWithLargeFailedSpool(t *testing.T) {
	spoolPath := t.TempDir()
	const recoveredEvents = 4000
	recoveredBody := []byte(`{"event_key":"recovered","url_id":1}`)
	for i := 0; i < recoveredEvents; i++ {
		name := fmt.Sprintf("%024d.pending.json", i)
		if err := os.WriteFile(filepath.Join(spoolPath, name), recoveredBody, 0o600); err != nil {
			t.Fatalf("seed spool: %v", err)
		}
	}

	recorder, err := tasks.NewClickRecorder(unavailableClickStore{}, spoolPath, 2, 1<<30, 1024)
	if err != nil {
		t.Fatalf("create click recorder: %v", err)
	}
	initialStats := recorder.Stats()
	if initialStats.Pending != recoveredEvents || initialStats.PendingBytes != int64(recoveredEvents*len(recoveredBody)) {
		t.Fatalf("startup stats = %+v", initialStats)
	}

	handler := &Handler{
		cfg:           &config.Config{AnalyticsSalt: "0123456789abcdef0123456789abcdef"},
		redirectCache: newRedirectCache(time.Minute, 10),
		clickRecorder: recorder,
		clientIP:      middleware.NewClientIPResolver(""),
	}
	handler.redirectCache.Set(&models.URL{
		ID: 1, OriginalURL: "https://example.com/destination", ShortCode: "fast",
		IsActive: true,
	})

	const parallelRedirects = 512
	start := time.Now()
	var wg sync.WaitGroup
	errorsChannel := make(chan error, parallelRedirects)
	for i := 0; i < parallelRedirects; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			request := httptest.NewRequest(http.MethodGet, "http://shortli.test/fast", nil)
			response := httptest.NewRecorder()
			handler.Redirect(response, request)
			if response.Code != http.StatusFound {
				errorsChannel <- fmt.Errorf("redirect status = %d", response.Code)
			}
		}()
	}
	wg.Wait()
	close(errorsChannel)
	for err := range errorsChannel {
		t.Error(err)
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("parallel redirects took %s with a large spool", elapsed)
	}
	if stats := recorder.Stats(); stats.Queued != parallelRedirects || stats.Dropped != 0 {
		t.Fatalf("redirect queue stats = %+v", stats)
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_ = recorder.Close(shutdownContext)
}
