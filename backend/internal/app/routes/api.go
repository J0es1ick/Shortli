package routes

import (
	"net/http"
	"time"

	abuseHandlers "github.com/J0es1ick/shortli/internal/app/handlers/abuseHandlers"
	authHandlers "github.com/J0es1ick/shortli/internal/app/handlers/authHandlers"
	developerHandlers "github.com/J0es1ick/shortli/internal/app/handlers/developerHandlers"
	"github.com/J0es1ick/shortli/internal/app/handlers/urlHandlers"
	userHandlers "github.com/J0es1ick/shortli/internal/app/handlers/userHandlers"
	"github.com/J0es1ick/shortli/internal/app/middleware"
	"github.com/J0es1ick/shortli/internal/app/tasks"
	"github.com/J0es1ick/shortli/internal/config"
	"github.com/J0es1ick/shortli/internal/models"
	"github.com/J0es1ick/shortli/internal/repository"
)

func SetupRoutes(
	cfg *config.Config,
	urlRepository *repository.UrlRepository,
	userRepo *repository.UserRepository,
	sessionRepo *repository.SessionRepository,
	apiKeyRepo *repository.APIKeyRepository,
	abuseRepo *repository.AbuseRepository,
	clickRecorder *tasks.ClickRecorder,
	metrics *middleware.MetricsRegistry,
	clientIP *middleware.ClientIPResolver,
) http.Handler {
	mux := http.NewServeMux()

	urlHandler := urlHandlers.NewHandler(cfg, urlRepository, abuseRepo, clickRecorder, clientIP)
	authHandler := authHandlers.NewAuthHandler(userRepo, sessionRepo, cfg.CookieSecure, cfg.AdminBootstrapToken)
	userHandler := userHandlers.NewUserHandler(userRepo, urlHandler.InvalidateRedirect, cfg.CookieSecure)
	developerHandler := developerHandlers.NewHandler(apiKeyRepo)
	abuseHandler := abuseHandlers.NewHandler(
		abuseRepo, urlRepository, cfg.AnalyticsSalt,
		clientIP, urlHandler.InvalidateRedirect,
	)
	apiKeyAuth := middleware.NewAPIKeyAuth(apiKeyRepo, userRepo, 120, time.Minute)
	shortenLimiter := middleware.NewRateLimiter(60, time.Hour, clientIP)
	reportLimiter := middleware.NewRateLimiter(5, time.Hour, clientIP)
	authLimiter := middleware.NewRateLimiter(20, 15*time.Minute, clientIP)
	staffOnly := middleware.RequireRoles(models.RoleOwner, models.RoleAdmin, models.RoleSupport)
	managementOnly := middleware.RequireRoles(models.RoleOwner, models.RoleAdmin)

	mux.HandleFunc("GET /", urlHandler.Home)
	mux.HandleFunc("GET /api/health", urlHandler.Health)
	mux.HandleFunc("GET /api/health/live", urlHandler.Liveness)
	mux.HandleFunc("GET /api/health/ready", urlHandler.Health)
	mux.HandleFunc("GET /api/metrics", metrics.Handler(cfg.MetricsToken, func() middleware.ClickQueueMetrics {
		stats := clickRecorder.Stats()
		return middleware.ClickQueueMetrics{
			Pending: stats.Pending, PendingBytes: stats.PendingBytes,
			MaxBytes: stats.MaxBytes, Queued: stats.Queued,
			Recorded: stats.Recorded, Retried: stats.Retried, Dropped: stats.Dropped,
		}
	}))
	mux.Handle("POST /api/shorten", shortenLimiter.Middleware(http.HandlerFunc(urlHandler.Shorten)))
	mux.Handle("POST /api/abuse-reports", reportLimiter.Middleware(http.HandlerFunc(abuseHandler.Create)))
	mux.HandleFunc("GET /api/stats/{shortCode}", middleware.RequireAuth(urlHandler.UrlStats))
	mux.HandleFunc("GET /{shortCode}", urlHandler.Redirect)

	mux.Handle("POST /api/register", authLimiter.Middleware(http.HandlerFunc(authHandler.Register)))
	mux.Handle("POST /api/admin/bootstrap", authLimiter.Middleware(http.HandlerFunc(authHandler.Bootstrap)))
	mux.Handle("POST /api/login", authLimiter.Middleware(http.HandlerFunc(authHandler.Login)))
	mux.HandleFunc("POST /api/logout", authHandler.Logout)
	mux.HandleFunc("GET /api/me", authHandler.Me)
	mux.HandleFunc("GET /api/v1/openapi.json", developerHandler.OpenAPI)

	mux.HandleFunc("POST /api/v1/links", apiKeyAuth.Require(urlHandler.Shorten))
	mux.HandleFunc("GET /api/v1/links/{shortCode}", apiKeyAuth.Require(urlHandler.UrlStats))
	mux.HandleFunc("PATCH /api/v1/links/{shortCode}", apiKeyAuth.Require(urlHandler.Update))
	mux.HandleFunc("DELETE /api/v1/links/{shortCode}", apiKeyAuth.Require(urlHandler.Delete))
	mux.HandleFunc("GET /api/v1/links/{shortCode}/analytics", apiKeyAuth.Require(urlHandler.Analytics))

	mux.HandleFunc("GET /api/history", middleware.RequireAuth(urlHandler.UserHistory))
	mux.HandleFunc("PATCH /api/urls/{shortCode}", middleware.RequireAuth(urlHandler.Update))
	mux.HandleFunc("GET /api/urls/{shortCode}/analytics", middleware.RequireAuth(urlHandler.Analytics))
	mux.HandleFunc("DELETE /api/urls/{shortCode}", middleware.RequireAuth(urlHandler.Delete))
	mux.HandleFunc("GET /api/developer/keys", middleware.RequireAuth(developerHandler.List))
	mux.HandleFunc("POST /api/developer/keys", middleware.RequireAuth(developerHandler.Create))
	mux.HandleFunc("DELETE /api/developer/keys/{id}", middleware.RequireAuth(developerHandler.Revoke))

	mux.HandleFunc("GET /api/user/profile", middleware.RequireAuth(userHandler.GetProfile))
	mux.HandleFunc("PUT /api/user/profile", middleware.RequireAuth(userHandler.UpdateProfile))
	mux.HandleFunc("POST /api/user/change-password", middleware.RequireAuth(userHandler.ChangePassword))
	mux.HandleFunc("DELETE /api/user/account", middleware.RequireAuth(userHandler.DeleteAccount))

	mux.HandleFunc("GET /api/admin/stats", staffOnly(urlHandler.AdminStats))
	mux.HandleFunc("GET /api/admin/urls", staffOnly(urlHandler.Stats))
	mux.HandleFunc("GET /api/admin/search", staffOnly(urlHandler.SearchUrls))
	mux.HandleFunc("DELETE /api/admin/urls/{shortCode}", staffOnly(urlHandler.Delete))
	mux.HandleFunc("GET /api/admin/abuse-reports", staffOnly(abuseHandler.List))
	mux.HandleFunc("PATCH /api/admin/abuse-reports/{id}", staffOnly(abuseHandler.Resolve))
	mux.HandleFunc("GET /api/admin/blocked-domains", staffOnly(abuseHandler.BlockedDomains))
	mux.HandleFunc("DELETE /api/admin/blocked-domains/{id}", managementOnly(abuseHandler.UnblockDomain))

	mux.HandleFunc("GET /api/admin/users", managementOnly(userHandler.GetAllUsers))
	mux.HandleFunc("PUT /api/admin/users/{id}", managementOnly(userHandler.UpdateUser))
	mux.HandleFunc("DELETE /api/admin/users/{id}", managementOnly(userHandler.DeleteUser))

	handler := middleware.SecurityHeaders(
		middleware.CORSMiddleware(
			middleware.AuthMiddleware(userRepo, sessionRepo, cfg.CookieSecure)(mux),
			cfg.FrontendOrigin,
		),
	)
	handler = middleware.RequestTimeout(time.Duration(cfg.RequestTimeout)*time.Second, handler)

	return handler
}
