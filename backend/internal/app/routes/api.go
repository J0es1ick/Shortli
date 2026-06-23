package routes

import (
	"net/http"

	authHandlers "github.com/J0es1ick/shortli/internal/app/handlers/authHandlers"
	"github.com/J0es1ick/shortli/internal/app/handlers/urlHandlers"
	userHandlers "github.com/J0es1ick/shortli/internal/app/handlers/userHandlers"
	"github.com/J0es1ick/shortli/internal/app/middleware"
	"github.com/J0es1ick/shortli/internal/config"
	"github.com/J0es1ick/shortli/internal/repository"
)

func SetupRoutes(cfg *config.Config, urlRepository *repository.UrlRepository, userRepo *repository.UserRepository, sessionRepo *repository.SessionRepository) http.Handler {
	mux := http.NewServeMux()

	urlHandler := urlHandlers.NewHandler(cfg, urlRepository)
	authHandler := authHandlers.NewAuthHandler(userRepo, sessionRepo)
	userHandler := userHandlers.NewUserHandler(userRepo)

	mux.HandleFunc("GET /", urlHandler.Home)
	mux.HandleFunc("POST /api/shorten", urlHandler.Shorten)
	mux.HandleFunc("GET /api/stats/{shortCode}", urlHandler.UrlStats)
	mux.HandleFunc("GET /{shortCode}", urlHandler.Redirect)

	mux.HandleFunc("POST /api/register", authHandler.Register)
	mux.HandleFunc("POST /api/login", authHandler.Login)
	mux.HandleFunc("POST /api/logout", authHandler.Logout)
	mux.HandleFunc("GET /api/me", authHandler.Me)

	mux.HandleFunc("GET /api/history", middleware.RequireAuth(urlHandler.UserHistory))
	mux.HandleFunc("DELETE /api/urls/{shortCode}", middleware.RequireAuth(urlHandler.Delete))
	
	mux.HandleFunc("GET /api/user/profile", middleware.RequireAuth(userHandler.GetProfile))
	mux.HandleFunc("PUT /api/user/profile", middleware.RequireAuth(userHandler.UpdateProfile))
	mux.HandleFunc("POST /api/user/change-password", middleware.RequireAuth(userHandler.ChangePassword))
	mux.HandleFunc("DELETE /api/user/account", middleware.RequireAuth(userHandler.DeleteAccount))

	mux.HandleFunc("GET /api/admin/stats", middleware.RequireAdmin(urlHandler.AdminStats))
	mux.HandleFunc("GET /api/admin/urls", middleware.RequireAdmin(urlHandler.Stats))
	mux.HandleFunc("GET /api/admin/search", middleware.RequireAdmin(urlHandler.SearchUrls))
	mux.HandleFunc("DELETE /api/admin/urls/{shortCode}", middleware.RequireAdmin(urlHandler.Delete))
	
	mux.HandleFunc("GET /api/admin/users", middleware.RequireAdmin(userHandler.GetAllUsers))
	mux.HandleFunc("PUT /api/admin/users/{id}", middleware.RequireAdmin(userHandler.UpdateUser))
	mux.HandleFunc("DELETE /api/admin/users/{id}", middleware.RequireAdmin(userHandler.DeleteUser))

	handler := middleware.CORSMiddleware(
		middleware.AuthMiddleware(userRepo, sessionRepo)(mux),
	)
	
	return handler
}