package authHandlers

import (
	"context"
	"database/sql/driver"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/J0es1ick/shortli/internal/app/middleware"
	"github.com/J0es1ick/shortli/internal/repository"
	"github.com/J0es1ick/shortli/internal/testutil"
)

func TestLogoutOnlyConfirmsRevocation(t *testing.T) {
	for _, tc := range []struct {
		name            string
		cookie, failure bool
		want            int
	}{
		{"success", true, false, 200}, {"database outage", true, true, 503}, {"already logged out", false, false, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			stub := &testutil.SQLStub{Exec: func(_ context.Context, _ string, args []driver.NamedValue) (driver.Result, error) {
				calls++
				if args[0].Value == "raw-token" {
					t.Error("raw token sent to database")
				}
				if tc.failure {
					return nil, errors.New("database unavailable")
				}
				return driver.RowsAffected(0), nil
			}}
			db := stub.DB(t)
			users, sessions := repository.NewUserRepository(db), repository.NewSessionRepository(db)
			h := middleware.AuthMiddleware(users, sessions, true)(http.HandlerFunc(NewAuthHandler(users, sessions, true, "").Logout))
			r := httptest.NewRequest("POST", "/api/logout", nil)
			if tc.cookie {
				r.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: "raw-token"})
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.want {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			if tc.cookie && calls != 1 {
				t.Fatalf("deletions=%d", calls)
			}
			cookies := w.Result().Cookies()
			if tc.failure {
				if len(cookies) != 0 {
					t.Fatal("failed revocation cleared cookie")
				}
			} else if len(cookies) != 1 || cookies[0].MaxAge != -1 || !cookies[0].Secure || !cookies[0].HttpOnly {
				t.Fatalf("invalid logout cookie: %+v", cookies)
			}
		})
	}
}
