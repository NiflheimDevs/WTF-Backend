package service

import (
	"context"

	"gitlab.chabokan.net/niflheim/wtf-backend/internal/domain"
	"gitlab.chabokan.net/niflheim/wtf-backend/internal/repository"
)

type MetricsService struct {
	metrics repository.MetricsRepository
}

func NewMetricsService(metrics repository.MetricsRepository) *MetricsService {
	return &MetricsService{metrics: metrics}
}

func (s *MetricsService) Summary(ctx context.Context) (*domain.MetricsSummary, error) {
	return s.metrics.GetSummary(ctx)
}

func (s *MetricsService) ByRegion(ctx context.Context, limit int) ([]*domain.RegionMetrics, error) {
	if limit < 1 || limit > 50 {
		limit = 10
	}
	return s.metrics.GetByRegion(ctx, limit)
}

func (s *MetricsService) ByNeedType(ctx context.Context) ([]*domain.NeedTypeMetrics, error) {
	return s.metrics.GetByNeedType(ctx)
}
