package middleware

import (
	"context"
	"net/http"
	"time"

	response "github.com/J0es1ick/shortli/internal/app/httputils"
	"github.com/J0es1ick/shortli/internal/models"
	"github.com/J0es1ick/shortli/internal/repository"
)

type contextKey string

const (
	UserContextKey    contextKey = "user"
	SessionCookieName string     = "session_id"
)

func AuthMiddleware(userRepo *repository.UserRepository, sessionRepo *repository.SessionRepository, secureCookies bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sessionCookie, err := r.Cookie(SessionCookieName)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			session, err := sessionRepo.GetSessionByID(r.Context(), sessionCookie.Value)
			if err != nil {
				http.SetCookie(w, &http.Cookie{
					Name:     SessionCookieName,
					Value:    "",
					Expires:  time.Now().Add(-1 * time.Hour),
					Path:     "/",
					HttpOnly: true,
					Secure:   secureCookies,
					SameSite: http.SameSiteStrictMode,
				})
				next.ServeHTTP(w, r)
				return
			}

			if time.Now().After(session.ExpiresAt) {
				_ = sessionRepo.DeleteSession(r.Context(), sessionCookie.Value)
				http.SetCookie(w, &http.Cookie{
					Name:     SessionCookieName,
					Value:    "",
					Expires:  time.Now().Add(-1 * time.Hour),
					Path:     "/",
					HttpOnly: true,
					Secure:   secureCookies,
					SameSite: http.SameSiteStrictMode,
				})
				next.ServeHTTP(w, r)
				return
			}

			user, err := userRepo.FindUserByID(r.Context(), session.UserID)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := GetUserFromContext(r)
		if user == nil {
			response.Error(w, http.StatusUnauthorized, "Authentication required")
			return
		}
		next(w, r)
	}
}

func RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := GetUserFromContext(r)
		if user == nil {
			response.Error(w, http.StatusUnauthorized, "Authentication required")
			return
		}
		if !user.IsAdmin {
			response.Error(w, http.StatusForbidden, "Admin access required")
			return
		}
		next(w, r)
	}
}

func GetUserFromContext(r *http.Request) *models.User {
	user, ok := r.Context().Value(UserContextKey).(*models.User)
	if !ok {
		return nil
	}
	return user
}
