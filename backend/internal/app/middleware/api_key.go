package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	response "github.com/J0es1ick/shortli/internal/app/httputils"
	"github.com/J0es1ick/shortli/internal/repository"
)

type apiKeyRate struct {
	windowStart time.Time
	count       int
}

type APIKeyAuth struct {
	keyRepo  *repository.APIKeyRepository
	userRepo *repository.UserRepository
	mu       sync.Mutex
	usage    map[int64]apiKeyRate
	limit    int
	window   time.Duration
}

func NewAPIKeyAuth(keyRepo *repository.APIKeyRepository, userRepo *repository.UserRepository, limit int, window time.Duration) *APIKeyAuth {
	return &APIKeyAuth{keyRepo: keyRepo, userRepo: userRepo, usage: make(map[int64]apiKeyRate), limit: limit, window: window}
}

func (a *APIKeyAuth) Require(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw := strings.TrimSpace(r.Header.Get("X-API-Key"))
		if !strings.HasPrefix(raw, "sk_live_") || len(raw) < 40 {
			response.Error(w, http.StatusUnauthorized, "Invalid or missing API key")
			return
		}
		key, err := a.keyRepo.Authenticate(raw)
		if err != nil {
			response.Error(w, http.StatusUnauthorized, "Invalid or missing API key")
			return
		}
		remaining, allowed := a.allow(key.ID)
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(a.limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		if !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(int(a.window.Seconds())))
			response.Error(w, http.StatusTooManyRequests, "API key rate limit exceeded")
			return
		}
		user, err := a.userRepo.FindUserByID(key.UserID)
		if err != nil {
			response.Error(w, http.StatusUnauthorized, "API key owner not found")
			return
		}
		go a.keyRepo.Touch(key.ID)
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next(w, r.WithContext(ctx))
	}
}

func (a *APIKeyAuth) allow(id int64) (int, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now()
	entry := a.usage[id]
	if entry.windowStart.IsZero() || now.Sub(entry.windowStart) >= a.window {
		entry = apiKeyRate{windowStart: now}
	}
	if entry.count >= a.limit {
		return 0, false
	}
	entry.count++
	a.usage[id] = entry
	return a.limit - entry.count, true
}
