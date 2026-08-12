package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	response "github.com/J0es1ick/shortli/internal/app/httputils"
)

type RateLimiter struct {
	mux               sync.Mutex
	limit             int
	window            time.Duration
	requests          map[string][]time.Time
	trustProxyHeaders bool
}

func NewRateLimiter(limit int, window time.Duration, trustProxyHeaders bool) *RateLimiter {
	return &RateLimiter{
		requests:          make(map[string][]time.Time),
		limit:             limit,
		window:            window,
		trustProxyHeaders: trustProxyHeaders,
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := GetClientIP(r, rl.trustProxyHeaders)

		rl.mux.Lock()

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
			rl.mux.Unlock()
			response.Error(w, http.StatusTooManyRequests, "Rate limit exceeded")
			return
		}

		rl.requests[clientIP] = append(rl.requests[clientIP], now)
		rl.mux.Unlock()
		next.ServeHTTP(w, r)
	})
}

func GetClientIP(r *http.Request, trustProxyHeaders bool) string {
	if trustProxyHeaders {
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			if ip := strings.TrimSpace(strings.Split(forwarded, ",")[0]); net.ParseIP(ip) != nil {
				return ip
			}
		}
		if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); net.ParseIP(ip) != nil {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
