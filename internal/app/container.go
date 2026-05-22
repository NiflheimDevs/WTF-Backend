package app

import (
	"context"
	"fmt"
	"log/slog"

	"gitlab.chabokan.net/niflheim/wtf-backend/internal/config"
	"gitlab.chabokan.net/niflheim/wtf-backend/internal/queue"
	"gitlab.chabokan.net/niflheim/wtf-backend/internal/repository/postgres"
	"gitlab.chabokan.net/niflheim/wtf-backend/internal/service"
	"github.com/riverqueue/river"
)

type Container struct {
	DB            *postgres.DB
	Queue         *queue.Client
	Auth          *service.AuthService
	Regions       *service.RegionService
	Requests      *service.RequestService
	Metrics       *service.MetricsService
	workersEnabled bool
}

func NewContainer(ctx context.Context, cfg config.Config) (*Container, error) {
	db, err := postgres.NewDB(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	// Initialize repositories
	users := postgres.NewUserRepository(db)
	regions := postgres.NewRegionRepository(db)
	requests := postgres.NewRequestRepository(db)
	audit := postgres.NewAuditLogRepository(db)
	metrics := postgres.NewMetricsRepository(db)

	// Setup logger
	logger := slog.Default()

	// Create workers for queue
	notifyWorker := queue.NewNotifyDispatcherWorker(audit, logger)
	metricsWorker := queue.NewRefreshMetricsWorker(requests, metrics, logger)

	// Register workers with River
	workers := river.NewWorkers()
	river.AddWorker(workers, notifyWorker)
	river.AddWorker(workers, metricsWorker)

	// Create queue client
	queueClient, err := queue.NewClient(db.Pool, workers)
	if err != nil {
		return nil, fmt.Errorf("create queue client: %w", err)
	}

	// Conditionally start queue processing based on config
	if cfg.EnableWorkers {
		if err := queueClient.Start(ctx); err != nil {
			return nil, fmt.Errorf("start queue client: %w", err)
		}
		logger.Info("background workers enabled and started")
	} else {
		logger.Info("background workers disabled (ENABLE_WORKERS=false)")
	}

	return &Container{
		DB:             db,
		Queue:          queueClient,
		Auth:           service.NewAuthService(users, cfg.JWTSecret, cfg.JWTTTL),
		Regions:        service.NewRegionService(regions),
		Requests:       service.NewRequestService(requests, regions, audit, queueClient),
		Metrics:        service.NewMetricsService(metrics),
		workersEnabled: cfg.EnableWorkers,
	}, nil
}

func (c *Container) Close() {
	if c.Queue != nil && c.workersEnabled {
		ctx := context.Background()
		_ = c.Queue.Stop(ctx)
	}
	if c.DB != nil {
		c.DB.Close()
	}
}
