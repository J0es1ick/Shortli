package urlHandlers

import (
	"testing"
	"time"

	"github.com/J0es1ick/shortli/internal/models"
)

func TestRedirectCacheStoresCopiesAndExpires(t *testing.T) {
	cache := newRedirectCache(20*time.Millisecond, 2)
	url := &models.URL{ID: 1, ShortCode: "demo", OriginalURL: "https://example.com", IsActive: true}
	cache.Set(url)
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
