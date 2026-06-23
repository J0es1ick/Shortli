package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/J0es1ick/shortli/internal/app/middleware"
	"github.com/J0es1ick/shortli/internal/app/routes"
	"github.com/J0es1ick/shortli/internal/app/tasks"
	"github.com/J0es1ick/shortli/internal/config"
	"github.com/J0es1ick/shortli/internal/database"
	"github.com/J0es1ick/shortli/internal/repository"
)

func main() {
	cfg, err := config.InitConfig()
	if err != nil {
		log.Fatalf("Config initialization error: %v", err)
	}

	fmt.Printf("Server port: %s\n", cfg.ServerPort)
	fmt.Printf("DB host: %s\n", cfg.Database.Host)

	db, err := database.DBInit(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	fmt.Println("Connection successful")
	defer db.Close()

	urlRepo := repository.NewUrlRepository(db.DB)
	userRepo := repository.NewUserRepository(db.DB)
	sessionRepo := repository.NewSessionRepository(db.DB)
	handler := routes.SetupRoutes(cfg, urlRepo, userRepo, sessionRepo)

	cleanupTask := tasks.NewCleanupTask(urlRepo, 24*time.Hour)
	go cleanupTask.Start()

	rateLimiter := middleware.NewRateLimiter(100, time.Minute) 
    handler = rateLimiter.Middleware(handler)

	server := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: handler,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Server starting on port %s", cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil {
			log.Printf("Server failed: %v", err)
			quit <- syscall.SIGTERM
		}
	}()

	<- quit
	log.Println("Shutting down server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server gracefully stopped")
}