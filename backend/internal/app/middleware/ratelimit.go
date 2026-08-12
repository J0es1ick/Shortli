package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	response "github.com/J0es1ick/shortli/internal/app/httputils"
)

type ClientIPResolver struct {
	trusted []*net.IPNet
}

func NewClientIPResolver(values string) *ClientIPResolver {
	resolver := &ClientIPResolver{}
	for _, value := range strings.Split(values, ",") {
		_, network, err := net.ParseCIDR(strings.TrimSpace(value))
		if err == nil {
			resolver.trusted = append(resolver.trusted, network)
		}
	}
	return resolver
}

func (r *ClientIPResolver) Resolve(request *http.Request) string {
	peer := parseRequestIP(request.RemoteAddr)
	if peer == nil {
		return request.RemoteAddr
	}
	if !r.isTrusted(peer) {
		return peer.String()
	}

	forwarded := request.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		parts := strings.Split(forwarded, ",")
		for index := len(parts) - 1; index >= 0; index-- {
			candidate := net.ParseIP(strings.TrimSpace(parts[index]))
			if candidate == nil {
				continue
			}
			if !r.isTrusted(candidate) {
				return candidate.String()
			}
		}
	}

	if candidate := net.ParseIP(strings.TrimSpace(request.Header.Get("X-Real-IP"))); candidate != nil {
		return candidate.String()
	}
	return peer.String()
}

func (r *ClientIPResolver) Trusts(request *http.Request) bool {
	return r.isTrusted(parseRequestIP(request.RemoteAddr))
}

func (r *ClientIPResolver) isTrusted(ip net.IP) bool {
	if ip == nil {
		return false
	}
	for _, network := range r.trusted {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func parseRequestIP(value string) net.IP {
	host, _, err := net.SplitHostPort(value)
	if err == nil {
		return net.ParseIP(host)
	}
	return net.ParseIP(strings.Trim(value, "[]"))
}

type rateEntry struct {
	requests []time.Time
	lastSeen time.Time
}

type RateLimiter struct {
	mux         sync.Mutex
	limit       int
	window      time.Duration
	requests    map[string]rateEntry
	resolver    *ClientIPResolver
	lastCleanup time.Time
	maxEntries  int
}

func NewRateLimiter(limit int, window time.Duration, resolver *ClientIPResolver) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string]rateEntry), limit: limit, window: window,
		resolver: resolver, lastCleanup: time.Now(), maxEntries: 50_000,
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := rl.resolver.Resolve(r)
		now := time.Now()

		rl.mux.Lock()
		rl.cleanup(now)
		entry := rl.requests[clientIP]
		validRequests := entry.requests[:0]
		for _, requestTime := range entry.requests {
			if now.Sub(requestTime) <= rl.window {
				validRequests = append(validRequests, requestTime)
			}
		}
		entry.requests = validRequests
		entry.lastSeen = now
		if len(entry.requests) >= rl.limit {
			rl.requests[clientIP] = entry
			rl.mux.Unlock()
			response.Error(w, http.StatusTooManyRequests, "Rate limit exceeded")
			return
		}

		entry.requests = append(entry.requests, now)
		rl.requests[clientIP] = entry
		rl.mux.Unlock()
		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) cleanup(now time.Time) {
	interval := rl.window
	if interval > time.Minute {
		interval = time.Minute
	}
	if now.Sub(rl.lastCleanup) < interval && len(rl.requests) < rl.maxEntries {
		return
	}
	for key, entry := range rl.requests {
		if now.Sub(entry.lastSeen) > rl.window {
			delete(rl.requests, key)
		}
	}
	for len(rl.requests) >= rl.maxEntries {
		var oldestKey string
		var oldest time.Time
		for key, entry := range rl.requests {
			if oldest.IsZero() || entry.lastSeen.Before(oldest) {
				oldestKey, oldest = key, entry.lastSeen
			}
		}
		delete(rl.requests, oldestKey)
	}
	rl.lastCleanup = now
}
