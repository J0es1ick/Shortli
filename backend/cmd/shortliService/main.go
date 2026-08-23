package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
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

	migrationCtx, migrationCancel := context.WithTimeout(context.Background(), 60*time.Second)
	if err := database.Migrate(migrationCtx, db.DB); err != nil {
		migrationCancel()
		log.Fatalf("Failed to apply database migrations: %v", err)
	}
	migrationCancel()
	log.Println("Database migrations are up to date")

	urlRepo := repository.NewUrlRepository(db.DB)
	userRepo := repository.NewUserRepository(db.DB)
	sessionRepo := repository.NewSessionRepository(db.DB)
	apiKeyRepo := repository.NewAPIKeyRepository(db.DB)
	abuseRepo := repository.NewAbuseRepository(db.DB)
	maintenanceRepo := repository.NewMaintenanceRepository(db.DB)
	clickRecorder, err := tasks.NewClickRecorder(urlRepo, cfg.ClickSpoolPath, 2, cfg.ClickSpoolMaxBytes)
	if err != nil {
		log.Fatalf("Failed to initialize durable click recorder: %v", err)
	}
	clientIP := middleware.NewClientIPResolver(cfg.TrustedProxyCIDRs)
	metrics := middleware.NewMetricsRegistry(clientIP, cfg.AnalyticsSalt)
	handler := routes.SetupRoutes(
		cfg, urlRepo, userRepo, sessionRepo, apiKeyRepo,
		abuseRepo, clickRecorder, metrics, clientIP,
	)

	runContext, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()
	maintenanceTask := tasks.NewMaintenanceTask(
		maintenanceRepo, 24*time.Hour,
		cfg.AnalyticsRetentionDays, cfg.ReportRetentionDays,
	)
	go maintenanceTask.Start(runContext)

	rateLimiter := middleware.NewRateLimiter(300, time.Minute, clientIP)
	handler = rateLimiter.Middleware(handler)
	handler = metrics.Middleware(handler)

	server := &http.Server{
		Addr:              ":" + cfg.ServerPort,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("Server starting on port %s", cfg.ServerPort)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case <-runContext.Done():
		log.Println("Shutdown signal received")
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			log.Printf("Server failed: %v", err)
		}
	}
	stopSignals()

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(cfg.ShutdownTimeout)*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
	if err := clickRecorder.Close(shutdownCtx); err != nil {
		log.Printf("Click recorder shutdown: %v", err)
	}

	log.Println("Server gracefully stopped")
}
