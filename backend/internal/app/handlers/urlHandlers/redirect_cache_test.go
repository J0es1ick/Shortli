package urlHandlers

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/J0es1ick/shortli/internal/models"
)

func TestRedirectCacheStoresCopiesAndExpires(t *testing.T) {
	cache := newRedirectCache(20*time.Millisecond, 2)
	url := &models.URL{ID: 1, ShortCode: "demo", OriginalURL: "https://example.com", IsActive: true}
	_, _, _ = cache.Resolve(context.Background(), url.ShortCode, func(context.Context, string) (*models.URL, error) { return url, nil })
	url.OriginalURL = "https://changed.example.com"

	cached, ok := cache.Get("demo")
	if !ok || cached.OriginalURL != "https://example.com" {
		t.Fatalf("cache did not preserve the stored value: %#v, %v", cached, ok)
	}
	time.Sleep(25 * time.Millisecond)
	if _, ok := cache.Get("demo"); ok {
		t.Fatal("expired cache entry was returned")
	}
}

func TestRedirectCacheInvalidationAllowsResumeAndRecreation(t *testing.T) {
	cache := newRedirectCache(time.Minute, 2)
	active := false
	exists := true
	lookup := func(context.Context, string) (*models.URL, error) {
		if !exists {
			return nil, sql.ErrNoRows
		}
		return &models.URL{ID: 1, ShortCode: "demo", IsActive: active}, nil
	}
	if value, _, err := cache.Resolve(context.Background(), "demo", lookup); err != nil || value.IsActive {
		t.Fatalf("initial paused state: %v %v", value, err)
	}
	active = true
	cache.Delete("demo")
	if value, hit, err := cache.Resolve(context.Background(), "demo", lookup); err != nil || !value.IsActive || hit {
		t.Fatalf("resume: %v %v %v", value, hit, err)
	}
	exists = false
	cache.Delete("demo")
	if _, _, err := cache.Resolve(context.Background(), "demo", lookup); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("delete: %v", err)
	}
	exists = true
	if value, hit, err := cache.Resolve(context.Background(), "demo", lookup); err != nil || !value.IsActive || hit {
		t.Fatalf("recreation: %v %v %v", value, hit, err)
	}
}

func TestRedirectCacheBoundsRetriesAndCopiesPointers(t *testing.T) {
	cache := newRedirectCache(time.Minute, 2)
	calls := 0
	_, _, err := cache.Resolve(context.Background(), "busy", func(context.Context, string) (*models.URL, error) {
		calls++
		cache.Delete("busy")
		return nil, sql.ErrNoRows
	})
	if !errors.Is(err, errRedirectCacheBusy) || calls != 16 {
		t.Fatalf("retry limit: calls=%d err=%v", calls, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := cache.Resolve(ctx, "busy", nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
	owner, expiry := 1, time.Now().Add(time.Hour)
	value := &models.URL{ShortCode: "copy", UserID: &owner, ExpiresAt: &expiry}
	_, _, _ = cache.Resolve(context.Background(), "copy", func(context.Context, string) (*models.URL, error) { return value, nil })
	owner = 2
	cached, _ := cache.Get("copy")
	if *cached.UserID != 1 {
		t.Fatal("stored owner pointer was aliased")
	}
	*cached.UserID = 3
	*cached.ExpiresAt = time.Time{}
	cached, _ = cache.Get("copy")
	if *cached.UserID != 1 || cached.ExpiresAt.IsZero() {
		t.Fatal("returned pointers were aliased")
	}
}

func TestRedirectCacheIgnoresUnrelatedInvalidations(t *testing.T) {
	cache := newRedirectCache(time.Minute, 2)
	calls := 0
	_, _, err := cache.Resolve(context.Background(), "lookup", func(context.Context, string) (*models.URL, error) {
		calls++
		cache.Delete("unrelated")
		return nil, sql.ErrNoRows
	})
	if !errors.Is(err, sql.ErrNoRows) || calls != 1 {
		t.Fatalf("unrelated mutation affected lookup: calls=%d err=%v", calls, err)
	}
	if len(cache.inflight) != 0 {
		t.Fatal("finished lookup retained generation metadata")
	}
}

func TestRedirectCacheBoundsInflightMetadata(t *testing.T) {
	cache := newRedirectCache(time.Minute, 1)
	started, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		_, _, _ = cache.Resolve(context.Background(), "blocked", func(context.Context, string) (*models.URL, error) {
			close(started)
			<-release
			return nil, sql.ErrNoRows
		})
	}()
	<-started
	_, _, err := cache.Resolve(context.Background(), "other", func(context.Context, string) (*models.URL, error) {
		t.Error("overload started another lookup")
		return nil, sql.ErrNoRows
	})
	if !errors.Is(err, errRedirectCacheBusy) {
		t.Errorf("overload: %v", err)
	}
	close(release)
	<-done
	if len(cache.inflight) != 0 {
		t.Fatal("finished lookup retained generation metadata")
	}
}
