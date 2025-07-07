package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"docker-registry-manager/internal/api"
	"docker-registry-manager/internal/config"
	"docker-registry-manager/internal/storage"

	"github.com/gorilla/handlers"
	"github.com/sirupsen/logrus"
)

func main() {
	var configFile = flag.String("config", "config.yaml", "Configuration file path")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %v", err)
	}

	// Setup logging
	setupLogging(cfg.Logging)

	logrus.Info("Starting Docker Registry Manager...")

	// Initialize storage
	storageBackend, err := storage.NewFilesystemStorage(cfg.Storage.Path)
	if err != nil {
		logrus.Fatalf("Failed to initialize storage: %v", err)
	}

	// Create API router
	router := api.NewRouter(cfg, storageBackend)

	// Setup CORS if enabled
	var handler http.Handler = router
	if cfg.CORS.Enabled {
		corsHandler := handlers.CORS(
			handlers.AllowedOrigins(cfg.CORS.AllowedOrigins),
			handlers.AllowedMethods(cfg.CORS.AllowedMethods),
			handlers.AllowedHeaders(cfg.CORS.AllowedHeaders),
		)
		handler = corsHandler(router)
	}

	// Create HTTP server
	server := &http.Server{
		Addr:         cfg.GetAddress(),
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in a goroutine
	go func() {
		logrus.Infof("Server starting on %s", cfg.GetAddress())
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("Shutting down server...")

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		logrus.Errorf("Server forced to shutdown: %v", err)
	}

	logrus.Info("Server exited")
}

func setupLogging(cfg config.LoggingConfig) {
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		logrus.Warnf("Invalid log level '%s', using 'info'", cfg.Level)
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)

	switch cfg.Format {
	case "json":
		logrus.SetFormatter(&logrus.JSONFormatter{})
	default:
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}
}
