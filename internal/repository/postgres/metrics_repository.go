package postgres

import (
	"context"
	"fmt"

	"gitlab.chabokan.net/niflheim/wtf-backend/internal/domain"
	"gitlab.chabokan.net/niflheim/wtf-backend/internal/repository"
)

// MetricsRepository implements repository.MetricsRepository using PostgreSQL
type MetricsRepository struct {
	db *DB
}

// NewMetricsRepository creates a new MetricsRepository
func NewMetricsRepository(db *DB) repository.MetricsRepository {
	return &MetricsRepository{db: db}
}

// GetSummary returns overall system metrics
func (r *MetricsRepository) GetSummary(ctx context.Context) (*domain.MetricsSummary, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'pending') as pending,
			COUNT(*) FILTER (WHERE status = 'dispatched') as dispatched,
			COUNT(*) FILTER (WHERE status = 'fulfilled') as fulfilled
		FROM requests
	`

	var summary domain.MetricsSummary
	err := r.db.Pool.QueryRow(ctx, query).Scan(
		&summary.TotalRequests,
		&summary.PendingRequests,
		&summary.DispatchedRequests,
		&summary.FulfilledRequests,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get metrics summary: %w", err)
	}

	return &summary, nil
}

// GetByRegion returns top regions by request count
func (r *MetricsRepository) GetByRegion(ctx context.Context, limit int) ([]*domain.RegionMetrics, error) {
	query := `
		SELECT 
			r.id,
			r.name_fa,
			r.name_en,
			COUNT(req.id) as request_count
		FROM regions r
		LEFT JOIN requests req ON req.region_id = r.id
		WHERE r.is_active = true
		GROUP BY r.id, r.name_fa, r.name_en
		ORDER BY request_count DESC
		LIMIT $1
	`

	rows, err := r.db.Pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics by region: %w", err)
	}
	defer rows.Close()

	var metrics []*domain.RegionMetrics
	for rows.Next() {
		var m domain.RegionMetrics
		err := rows.Scan(
			&m.RegionID,
			&m.RegionNameFa,
			&m.RegionNameEn,
			&m.RequestCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan region metrics: %w", err)
		}
		metrics = append(metrics, &m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating region metrics: %w", err)
	}

	return metrics, nil
}

// GetByNeedType returns metrics grouped by need type
func (r *MetricsRepository) GetByNeedType(ctx context.Context) ([]*domain.NeedTypeMetrics, error) {
	query := `
		SELECT 
			need_type,
			COUNT(*) as request_count,
			COALESCE(SUM(quantity), 0) as total_quantity
		FROM requests
		GROUP BY need_type
		ORDER BY request_count DESC
	`

	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics by need type: %w", err)
	}
	defer rows.Close()

	var metrics []*domain.NeedTypeMetrics
	for rows.Next() {
		var m domain.NeedTypeMetrics
		err := rows.Scan(
			&m.NeedType,
			&m.RequestCount,
			&m.TotalQuantity,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan need type metrics: %w", err)
		}
		metrics = append(metrics, &m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating need type metrics: %w", err)
	}

	return metrics, nil
}

// UpsertDaily inserts or updates daily metrics
func (r *MetricsRepository) UpsertDaily(ctx context.Context, metrics *domain.MetricsDaily) error {
	query := `
		INSERT INTO metrics_daily (
			metric_date, region_id, need_type, request_count, 
			total_quantity, pending_count, dispatched_count, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (metric_date, region_id, need_type)
		DO UPDATE SET
			request_count = EXCLUDED.request_count,
			total_quantity = EXCLUDED.total_quantity,
			pending_count = EXCLUDED.pending_count,
			dispatched_count = EXCLUDED.dispatched_count,
			updated_at = EXCLUDED.updated_at
	`

	_, err := r.db.Pool.Exec(ctx, query,
		metrics.MetricDate,
		metrics.RegionID,
		metrics.NeedType,
		metrics.RequestCount,
		metrics.TotalQuantity,
		metrics.PendingCount,
		metrics.DispatchedCount,
		metrics.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to upsert daily metrics: %w", err)
	}

	return nil
}
