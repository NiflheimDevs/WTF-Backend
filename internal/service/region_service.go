package service

import (
	"context"

	"gitlab.chabokan.net/niflheim/wtf-backend/internal/domain"
	"gitlab.chabokan.net/niflheim/wtf-backend/internal/repository"
)

type RegionService struct {
	regions repository.RegionRepository
}

func NewRegionService(regions repository.RegionRepository) *RegionService {
	return &RegionService{regions: regions}
}

func (s *RegionService) ListActive(ctx context.Context) ([]*domain.Region, error) {
	return s.regions.ListActive(ctx)
}
