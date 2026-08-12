package urlHandlers

import (
	"sync"
	"time"

	"github.com/J0es1ick/shortli/internal/models"
)

type cacheEntry struct {
	url       models.URL
	expiresAt time.Time
	createdAt time.Time
}

type redirectCache struct {
	mu         sync.RWMutex
	items      map[string]cacheEntry
	ttl        time.Duration
	maxEntries int
}

func newRedirectCache(ttl time.Duration, maxEntries int) *redirectCache {
	return &redirectCache{items: make(map[string]cacheEntry), ttl: ttl, maxEntries: maxEntries}
}

func (c *redirectCache) Get(code string) (*models.URL, bool) {
	c.mu.RLock()
	entry, ok := c.items[code]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.Delete(code)
		return nil, false
	}
	value := entry.url
	return &value, true
}

func (c *redirectCache) Set(url *models.URL) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	if len(c.items) >= c.maxEntries {
		oldestCode := ""
		oldestTime := now
		for code, entry := range c.items {
			if now.After(entry.expiresAt) {
				delete(c.items, code)
				continue
			}
			if oldestCode == "" || entry.createdAt.Before(oldestTime) {
				oldestCode, oldestTime = code, entry.createdAt
			}
		}
		if len(c.items) >= c.maxEntries && oldestCode != "" {
			delete(c.items, oldestCode)
		}
	}
	c.items[url.ShortCode] = cacheEntry{url: *url, expiresAt: now.Add(c.ttl), createdAt: now}
}

func (c *redirectCache) Delete(code string) {
	c.mu.Lock()
	delete(c.items, code)
	c.mu.Unlock()
}
