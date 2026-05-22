package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/riverqueue/river"
	"gitlab.chabokan.net/niflheim/wtf-backend/internal/config"
	"gitlab.chabokan.net/niflheim/wtf-backend/internal/queue"
	"gitlab.chabokan.net/niflheim/wtf-backend/internal/repository/postgres"
)

func main() {
	// Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Initialize database connection
	ctx := context.Background()
	db, err := postgres.NewDB(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	logger.Info("database connection established")

	// Initialize repositories
	auditRepo := postgres.NewAuditLogRepository(db)
	requestRepo := postgres.NewRequestRepository(db)
	metricsRepo := postgres.NewMetricsRepository(db)

	// Create workers
	notifyWorker := queue.NewNotifyDispatcherWorker(auditRepo, logger)
	metricsWorker := queue.NewRefreshMetricsWorker(requestRepo, metricsRepo, logger)

	// Register workers with River
	workers := river.NewWorkers()
	river.AddWorker(workers, notifyWorker)
	river.AddWorker(workers, metricsWorker)

	// Create queue client
	queueClient, err := queue.NewClient(db.Pool, workers)
	if err != nil {
		logger.Error("failed to create queue client", "error", err)
		os.Exit(1)
	}

	// Start the worker
	if err := queueClient.Start(ctx); err != nil {
		logger.Error("failed to start queue client", "error", err)
		os.Exit(1)
	}

	logger.Info("worker started successfully", "workers", []string{"notify_dispatcher", "refresh_metrics"})

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	logger.Info("shutdown signal received, stopping worker gracefully...")

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := queueClient.Stop(shutdownCtx); err != nil {
		logger.Error("error during shutdown", "error", err)
		os.Exit(1)
	}

	logger.Info("worker stopped successfully")
}
