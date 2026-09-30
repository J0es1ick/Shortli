package middleware

import (
	"context"
	"database/sql/driver"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/J0es1ick/shortli/internal/repository"
	"github.com/J0es1ick/shortli/internal/testutil"
)

func TestAuthDistinguishesOutageFromMissingSession(t *testing.T) {
	for _, mode := range []string{"session outage", "user outage", "missing", "expired"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			stub := &testutil.SQLStub{Query: func(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
				calls++
				if mode == "session outage" || calls > 1 {
					return nil, errors.New("database unavailable")
				}
				rows := &testutil.Rows{Names: []string{"session_id", "user_id", "expires_at", "created_at"}}
				expires := time.Now().Add(time.Hour)
				if mode == "expired" {
					expires = time.Now().Add(-time.Hour)
				}
				if mode != "missing" {
					rows.Values = [][]driver.Value{{"hash", int64(1), expires, time.Now()}}
				}
				return rows, nil
			}}
			db := stub.DB(t)
			downstream := false
			h := AuthMiddleware(repository.NewUserRepository(db), repository.NewSessionRepository(db), true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				downstream = true
				if GetUserFromContext(r) != nil {
					t.Error("unexpected authenticated user")
				}
				w.WriteHeader(204)
			}))
			r := httptest.NewRequest("GET", "/api/me", nil)
			r.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "test-token"})
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if mode == "missing" || mode == "expired" {
				if !downstream || len(w.Result().Cookies()) != 1 {
					t.Fatal("invalid session was not cleared")
				}
			} else if downstream || w.Code != 503 || len(w.Result().Cookies()) != 0 {
				t.Fatalf("outage changed auth state: downstream=%v status=%d cookie=%q", downstream, w.Code, w.Header().Get("Set-Cookie"))
			}
		})
	}
}

func TestPublicRedirectAndHealthDoNotDependOnSessionStorage(t *testing.T) {
	for _, path := range []string{"/public-code", "/api/health", "/api/health/ready"} {
		h := AuthMiddleware(nil, nil, true)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) }))
		r := httptest.NewRequest("GET", path, nil)
		r.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "test-token"})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 204 || len(w.Result().Cookies()) != 0 {
			t.Fatalf("public route %s depended on session", path)
		}
	}
}
