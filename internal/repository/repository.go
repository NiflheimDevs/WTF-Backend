package repository

import (
	"context"
	"time"

	"gitlab.chabokan.net/niflheim/wtf-backend/internal/domain"
	"github.com/google/uuid"
)

// UserRepository defines operations for user data access
type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	Create(ctx context.Context, user *domain.User) error
}

// RegionRepository defines operations for region data access
type RegionRepository interface {
	ListActive(ctx context.Context) ([]*domain.Region, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Region, error)
}

// RequestFilters defines filters for listing requests
type RequestFilters struct {
	Status   *domain.RequestStatus
	RegionID *uuid.UUID
	FromDate *time.Time
	ToDate   *time.Time
	Limit    int
	Offset   int
}

// RequestRepository defines operations for request data access
type RequestRepository interface {
	Create(ctx context.Context, req *domain.Request) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Request, error)
	List(ctx context.Context, filters RequestFilters) ([]*domain.Request, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.RequestStatus, dispatchedBy *uuid.UUID) error
	Count(ctx context.Context, filters RequestFilters) (int, error)
}

// MetricsRepository defines operations for metrics data access
type MetricsRepository interface {
	GetSummary(ctx context.Context) (*domain.MetricsSummary, error)
	GetByRegion(ctx context.Context, limit int) ([]*domain.RegionMetrics, error)
	GetByNeedType(ctx context.Context) ([]*domain.NeedTypeMetrics, error)
	UpsertDaily(ctx context.Context, metrics *domain.MetricsDaily) error
}

// AuditLogRepository defines operations for audit log data access
type AuditLogRepository interface {
	Insert(ctx context.Context, log *domain.AuditLog) error
	ListByRequestID(ctx context.Context, requestID uuid.UUID) ([]*domain.AuditLog, error)
}
