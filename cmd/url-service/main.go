package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"go-shortener/internal/urlservice/database"
	httpdelivery "go-shortener/internal/urlservice/delivery/http"
	"go-shortener/internal/urlservice/repository/sqlite"
	"go-shortener/internal/urlservice/usecase"

	dapr "github.com/dapr/go-sdk/client"
	_ "modernc.org/sqlite"

	"go.uber.org/zap"
)

// getEnv retrieves an environment variable or returns the default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	// Initialize logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	// Read configuration from environment variables
	port := getEnv("PORT", "8080")
	databasePath := getEnv("DATABASE_PATH", "data/shortener.db")
	baseURL := getEnv("BASE_URL", "http://localhost:8080")
	rateLimitStr := getEnv("RATE_LIMIT", "100")

	rateLimit, err := strconv.Atoi(rateLimitStr)
	if err != nil {
		logger.Fatal("invalid RATE_LIMIT value", zap.String("value", rateLimitStr), zap.Error(err))
	}

	// Ensure data directory exists
	if err := os.MkdirAll(filepath.Dir(databasePath), 0755); err != nil {
		logger.Fatal("failed to create data directory", zap.Error(err))
	}

	// Open database
	db, err := database.OpenDB(databasePath)
	if err != nil {
		logger.Fatal("failed to open database", zap.Error(err))
	}
	defer db.Close()

	// Run migrations
	if err := database.RunMigrations(db); err != nil {
		logger.Fatal("failed to run migrations", zap.Error(err))
	}

	logger.Info("database initialized", zap.String("path", databasePath))

	// Create Dapr client for pub/sub publishing
	daprClient, err := dapr.NewClient()
	if err != nil {
		// Log warning but don't fail — service can run without Dapr for local dev
		logger.Warn("failed to create Dapr client, click tracking disabled", zap.Error(err))
	} else {
		defer daprClient.Close()
	}

	// Wire dependencies
	repo := sqlite.NewURLRepository(db)
	service := usecase.NewURLService(repo, daprClient, logger, baseURL)
	handler := httpdelivery.NewHandler(service, baseURL, daprClient, logger)
	rateLimiter := httpdelivery.NewRateLimiter(rateLimit)
	router := httpdelivery.NewRouter(handler, logger, rateLimiter)

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("server starting",
			zap.String("port", port),
			zap.String("base_url", baseURL),
			zap.Int("rate_limit", rateLimit),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server failed", zap.Error(err))
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("server shutting down")

	// Graceful shutdown with 10 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("server forced to shutdown", zap.Error(err))
	}

	logger.Info("server stopped")
}
