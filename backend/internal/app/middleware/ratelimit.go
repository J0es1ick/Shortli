package middleware

import (
	"net/http"
	"sync"
	"time"

	response "github.com/J0es1ick/shortli/internal/app/httputils"
)

type RateLimiter struct {
	mux 	sync.Mutex
	limit 	int
	window  time.Duration
	requests map[string][]time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
    return &RateLimiter{
        requests: make(map[string][]time.Time),
        limit:    limit,
        window:   window,
    }
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := getClientIP(r) 

		rl.mux.Lock()
		defer rl.mux.Unlock()

		now := time.Now()
		if _, exists := rl.requests[clientIP]; !exists {
			rl.requests[clientIP] = []time.Time{}
		}

		validRequests := []time.Time{}
		for _, t := range rl.requests[clientIP] {
			if now.Sub(t) <= rl.window {
				validRequests = append(validRequests, t)
			}
		}
		rl.requests[clientIP] = validRequests

		if len(rl.requests[clientIP]) >= rl.limit {
			response.Error(w, http.StatusTooManyRequests, "Rate limit exceeded")
			return
		}

		rl.requests[clientIP] = append(rl.requests[clientIP], now)
		next.ServeHTTP(w, r)
	})
}

func getClientIP(r *http.Request) string {
    if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
        return ip
    }
    if ip := r.Header.Get("X-Real-IP"); ip != "" {
        return ip
    }
    return r.RemoteAddr
}