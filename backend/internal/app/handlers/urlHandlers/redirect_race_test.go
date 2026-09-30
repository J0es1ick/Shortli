package urlHandlers

import (
	"context"
	"database/sql/driver"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/J0es1ick/shortli/internal/app/middleware"
	"github.com/J0es1ick/shortli/internal/app/tasks"
	"github.com/J0es1ick/shortli/internal/config"
	"github.com/J0es1ick/shortli/internal/models"
	"github.com/J0es1ick/shortli/internal/repository"
	"github.com/J0es1ick/shortli/internal/testutil"
)

func TestRedirectCannotResurrectChangedLink(t *testing.T) {
	for _, mutation := range []string{"pause", "delete", "external-invalidation"} {
		t.Run(mutation, func(t *testing.T) {
			var mu sync.Mutex
			active, exists, reads := true, true, 0
			started, release := make(chan struct{}), make(chan struct{})
			var unblock sync.Once
			t.Cleanup(func() { unblock.Do(func() { close(release) }) })
			stub := &testutil.SQLStub{}
			stub.Query = func(ctx context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
				if strings.Contains(query, "DELETE") {
					mu.Lock()
					exists = false
					mu.Unlock()
					return &testutil.Rows{Names: []string{"url_id"}, Values: [][]driver.Value{{int64(1)}}}, nil
				}
				mu.Lock()
				reads++
				first, capturedActive, capturedExists := reads == 1, active, exists
				mu.Unlock()
				if first {
					close(started)
					select {
					case <-release:
					case <-ctx.Done():
						return nil, ctx.Err()
					}
				}
				rows := &testutil.Rows{Names: []string{"url_id", "original_url", "short_code", "user_id", "click_count", "created_at", "expires_at", "is_active"}}
				if capturedExists {
					rows.Values = [][]driver.Value{{int64(1), "https://example.com/", "race", int64(1), int64(0), time.Now(), nil, capturedActive}}
				}
				return rows, nil
			}
			stub.Exec = func(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
				mu.Lock()
				defer mu.Unlock()
				if strings.Contains(query, "DELETE") {
					exists = false
				} else {
					active = false
				}
				return driver.RowsAffected(1), nil
			}
			recorder, err := tasks.NewClickRecorder(unavailableClickStore{}, t.TempDir(), 1, 1<<20, 10)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()
				_ = recorder.Close(ctx)
			})
			h := &Handler{urlRepository: repository.NewUrlRepository(stub.DB(t)), redirectCache: newRedirectCache(time.Minute, 10), clickRecorder: recorder,
				cfg: &config.Config{AnalyticsSalt: "0123456789abcdef0123456789abcdef"}, clientIP: middleware.NewClientIPResolver("")}
			done := make(chan *httptest.ResponseRecorder, 1)
			go func() {
				w := httptest.NewRecorder()
				h.Redirect(w, httptest.NewRequest("GET", "/race", nil))
				done <- w
			}()
			select {
			case <-started:
			case <-time.After(3 * time.Second):
				t.Fatal("lookup did not start")
			}
			if mutation == "external-invalidation" {
				mu.Lock()
				active = false
				mu.Unlock()
				h.InvalidateRedirect("race")
			} else {
				r := httptest.NewRequest("PATCH", "/api/urls/race", strings.NewReader(`{"is_active":false}`))
				r.SetPathValue("shortCode", "race")
				r = r.WithContext(context.WithValue(r.Context(), middleware.UserContextKey, &models.User{ID: 1, Role: models.RoleUser}))
				w := httptest.NewRecorder()
				if mutation == "delete" {
					r.Method = "DELETE"
					h.Delete(w, r)
				} else {
					h.Update(w, r)
				}
				if w.Code >= 300 {
					t.Fatalf("mutation: %d %s", w.Code, w.Body.String())
				}
			}
			unblock.Do(func() { close(release) })
			want := http.StatusGone
			if mutation == "delete" {
				want = http.StatusNotFound
			}
			select {
			case w := <-done:
				if w.Code != want {
					t.Errorf("in-flight redirect = %d, want %d", w.Code, want)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("redirect did not finish")
			}
			w := httptest.NewRecorder()
			h.Redirect(w, httptest.NewRequest("GET", "/race", nil))
			if w.Code != want {
				t.Fatalf("later redirect = %d, cache=%s, want %d", w.Code, w.Header().Get("X-Shortli-Cache"), want)
			}
		})
	}
}
