package urlHandlers

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/J0es1ick/shortli/internal/models"
)

type cacheEntry struct {
	url       models.URL
	expiresAt time.Time
	createdAt time.Time
}

type cacheFlight struct {
	generation uint64
	readers    int
}

type redirectCache struct {
	mu         sync.RWMutex
	items      map[string]cacheEntry
	ttl        time.Duration
	maxEntries int
	inflight   map[string]*cacheFlight
}

var errRedirectCacheBusy = errors.New("redirect state changed repeatedly")

func newRedirectCache(ttl time.Duration, maxEntries int) *redirectCache {
	return &redirectCache{items: make(map[string]cacheEntry), inflight: make(map[string]*cacheFlight), ttl: ttl, maxEntries: maxEntries}
}

func (c *redirectCache) Get(code string) (*models.URL, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.items[code]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	value := cloneURL(entry.url)
	return &value, true
}

func (c *redirectCache) Resolve(ctx context.Context, code string, lookup func(context.Context, string) (*models.URL, error)) (*models.URL, bool, error) {
	for attempt := 0; attempt < 16; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, false, err
		}
		if value, ok := c.Get(code); ok {
			return value, true, nil
		}
		c.mu.Lock()
		entry, ok := c.items[code]
		if ok && time.Now().Before(entry.expiresAt) {
			value := cloneURL(entry.url)
			c.mu.Unlock()
			return &value, true, nil
		}
		flight := c.inflight[code]
		if flight == nil {
			if len(c.inflight) >= c.maxEntries {
				c.mu.Unlock()
				return nil, false, errRedirectCacheBusy
			}
			flight = &cacheFlight{}
			c.inflight[code] = flight
		}
		generation := flight.generation
		flight.readers++
		c.mu.Unlock()
		value, err := lookup(ctx, code)
		c.mu.Lock()
		flight.readers--
		if flight.readers == 0 {
			delete(c.inflight, code)
		}
		if generation != flight.generation {
			c.mu.Unlock()
			continue
		}
		if err == nil {
			c.storeLocked(value)
		}
		c.mu.Unlock()
		return value, false, err
	}
	return nil, false, errRedirectCacheBusy
}

func (c *redirectCache) storeLocked(url *models.URL) {
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
	c.items[url.ShortCode] = cacheEntry{url: cloneURL(*url), expiresAt: now.Add(c.ttl), createdAt: now}
}

func (c *redirectCache) Delete(code string) {
	c.mu.Lock()
	if flight := c.inflight[code]; flight != nil {
		flight.generation++
	}
	delete(c.items, code)
	c.mu.Unlock()
}

func cloneURL(url models.URL) models.URL {
	if url.UserID != nil {
		value := *url.UserID
		url.UserID = &value
	}
	if url.ExpiresAt != nil {
		value := *url.ExpiresAt
		url.ExpiresAt = &value
	}
	return url
}
