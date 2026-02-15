package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	analyticsdb "go-shortener/internal/analytics/database"
	analyticshttp "go-shortener/internal/analytics/delivery/http"
	"go-shortener/internal/analytics/enrichment"
	analyticssqlite "go-shortener/internal/analytics/repository/sqlite"
	"go-shortener/internal/analytics/usecase"

	_ "modernc.org/sqlite"
	"go.uber.org/zap"
)

// fallbackGeoIPResolver always returns "Unknown" when GeoIP database is unavailable
type fallbackGeoIPResolver struct{}

func (f *fallbackGeoIPResolver) ResolveCountry(ip string) string {
	return "Unknown"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	port := getEnv("PORT", "8081")
	databasePath := getEnv("DATABASE_PATH", "data/analytics.db")
	geoipDBPath := getEnv("GEOIP_DB_PATH", "data/GeoLite2-Country.mmdb")

	// Ensure data directory exists
	if err := os.MkdirAll(filepath.Dir(databasePath), 0755); err != nil {
		logger.Fatal("failed to create data directory", zap.Error(err))
	}

	// Open database (separate from URL Service)
	db, err := analyticsdb.OpenDB(databasePath)
	if err != nil {
		logger.Fatal("failed to open database", zap.Error(err))
	}
	defer db.Close()

	// Run analytics migrations
	if err := analyticsdb.RunMigrations(db); err != nil {
		logger.Fatal("failed to run migrations", zap.Error(err))
	}

	logger.Info("analytics database initialized", zap.String("path", databasePath))

	// Initialize enrichment services
	geoIPResolver, err := enrichment.NewGeoIPResolver(geoipDBPath)
	if err != nil {
		logger.Warn("GeoIP database not available, country resolution disabled",
			zap.Error(err),
			zap.String("path", geoipDBPath),
		)
		geoIPResolver = nil
	} else {
		logger.Info("GeoIP database loaded successfully", zap.String("path", geoipDBPath))
		defer geoIPResolver.Close()
	}

	deviceDetector := enrichment.NewDeviceDetector()
	refererClassifier := enrichment.NewRefererClassifier()

	// Wire dependencies
	repo := analyticssqlite.NewClickRepository(db)

	// Use fallback GeoIP resolver if database is unavailable
	var geoIP usecase.GeoIPResolver
	if geoIPResolver != nil {
		geoIP = geoIPResolver
	} else {
		geoIP = &fallbackGeoIPResolver{}
	}

	service := usecase.NewAnalyticsService(repo, geoIP, deviceDetector, refererClassifier)
	handler := analyticshttp.NewHandler(service, logger, db)
	router := analyticshttp.NewRouter(handler, logger)

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Start server
	go func() {
		logger.Info("analytics service starting", zap.String("port", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server failed", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("analytics service shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("server forced to shutdown", zap.Error(err))
	}

	logger.Info("analytics service stopped")
}
