package middleware

import (
	"context"
	"net/http"
	"time"
)

func RequestTimeout(duration time.Duration, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), duration)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
