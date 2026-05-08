package app

import (
	"context"
	"fmt"

	"gitlab.chabokan.net/niflheim/wtf-backend/internal/config"
	"gitlab.chabokan.net/niflheim/wtf-backend/internal/repository/postgres"
	"gitlab.chabokan.net/niflheim/wtf-backend/internal/service"
)

type Container struct {
	DB       *postgres.DB
	Auth     *service.AuthService
	Regions  *service.RegionService
	Requests *service.RequestService
	Metrics  *service.MetricsService
}

func NewContainer(ctx context.Context, cfg config.Config) (*Container, error) {
	db, err := postgres.NewDB(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	users := postgres.NewUserRepository(db)
	regions := postgres.NewRegionRepository(db)
	requests := postgres.NewRequestRepository(db)
	audit := postgres.NewAuditLogRepository(db)
	metrics := postgres.NewMetricsRepository(db)

	return &Container{
		DB:       db,
		Auth:     service.NewAuthService(users, cfg.JWTSecret, cfg.JWTTTL),
		Regions:  service.NewRegionService(regions),
		Requests: service.NewRequestService(requests, regions, audit),
		Metrics:  service.NewMetricsService(metrics),
	}, nil
}

func (c *Container) Close() {
	if c.DB != nil {
		c.DB.Close()
	}
}
