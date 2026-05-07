package postgres

import (
	"context"
	"errors"
	"fmt"

	"gitlab.chabokan.net/niflheim/wtf-backend/internal/domain"
	"gitlab.chabokan.net/niflheim/wtf-backend/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// RegionRepository implements repository.RegionRepository using PostgreSQL
type RegionRepository struct {
	db *DB
}

// NewRegionRepository creates a new RegionRepository
func NewRegionRepository(db *DB) repository.RegionRepository {
	return &RegionRepository{db: db}
}

// ListActive returns all active regions ordered by display_order
func (r *RegionRepository) ListActive(ctx context.Context) ([]*domain.Region, error) {
	query := `
		SELECT id, name_fa, name_en, parent_id, is_active, display_order
		FROM regions
		WHERE is_active = true
		ORDER BY display_order, name_en
	`

	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list active regions: %w", err)
	}
	defer rows.Close()

	var regions []*domain.Region
	for rows.Next() {
		var region domain.Region
		err := rows.Scan(
			&region.ID,
			&region.NameFa,
			&region.NameEn,
			&region.ParentID,
			&region.IsActive,
			&region.DisplayOrder,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan region: %w", err)
		}
		regions = append(regions, &region)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating regions: %w", err)
	}

	return regions, nil
}

// FindByID finds a region by ID
func (r *RegionRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Region, error) {
	query := `
		SELECT id, name_fa, name_en, parent_id, is_active, display_order
		FROM regions
		WHERE id = $1
	`

	var region domain.Region
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&region.ID,
		&region.NameFa,
		&region.NameEn,
		&region.ParentID,
		&region.IsActive,
		&region.DisplayOrder,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("region not found: %w", err)
		}
		return nil, fmt.Errorf("failed to find region by id: %w", err)
	}

	return &region, nil
}
