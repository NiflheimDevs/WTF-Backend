package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"gitlab.chabokan.net/niflheim/wtf-backend/internal/domain"
	"gitlab.chabokan.net/niflheim/wtf-backend/internal/repository"
)

// RequestRepository implements repository.RequestRepository using PostgreSQL
type RequestRepository struct {
	db *DB
}

// NewRequestRepository creates a new RequestRepository
func NewRequestRepository(db *DB) repository.RequestRepository {
	return &RequestRepository{db: db}
}

// Create creates a new request
func (r *RequestRepository) Create(ctx context.Context, req *domain.Request) error {
	query := `
		INSERT INTO requests (
			id, region_id, need_type, quantity, contact_phone, note,
			status, submitted_ip, submitted_user_agent, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.db.Pool.Exec(ctx, query,
		req.ID,
		req.RegionID,
		req.NeedType,
		req.Quantity,
		req.ContactPhone,
		req.Note,
		req.Status,
		req.SubmittedIP,
		req.SubmittedUserAgent,
		req.CreatedAt,
		req.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	return nil
}

// FindByID finds a request by ID
func (r *RequestRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Request, error) {
	query := `
		SELECT 
			id, region_id, need_type, quantity, contact_phone, note,
			status, submitted_ip, submitted_user_agent,
			dispatched_by, dispatched_at, created_at, updated_at
		FROM requests
		WHERE id = $1
	`

	var req domain.Request
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&req.ID,
		&req.RegionID,
		&req.NeedType,
		&req.Quantity,
		&req.ContactPhone,
		&req.Note,
		&req.Status,
		&req.SubmittedIP,
		&req.SubmittedUserAgent,
		&req.DispatchedBy,
		&req.DispatchedAt,
		&req.CreatedAt,
		&req.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("request not found: %w", err)
		}
		return nil, fmt.Errorf("failed to find request by id: %w", err)
	}

	return &req, nil
}

// List returns requests matching the given filters
func (r *RequestRepository) List(ctx context.Context, filters repository.RequestFilters) ([]*domain.Request, error) {
	query := `
		SELECT 
			id, region_id, need_type, quantity, contact_phone, note,
			status, submitted_ip, submitted_user_agent,
			dispatched_by, dispatched_at, created_at, updated_at
		FROM requests
		WHERE 1=1
	`

	args := []interface{}{}
	argPos := 1

	if filters.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, *filters.Status)
		argPos++
	}

	if filters.RegionID != nil {
		query += fmt.Sprintf(" AND region_id = $%d", argPos)
		args = append(args, *filters.RegionID)
		argPos++
	}

	if filters.FromDate != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argPos)
		args = append(args, *filters.FromDate)
		argPos++
	}

	if filters.ToDate != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argPos)
		args = append(args, *filters.ToDate)
		argPos++
	}

	query += " ORDER BY created_at DESC"

	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, filters.Limit)
		argPos++
	}

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argPos)
		args = append(args, filters.Offset)
		argPos++
	}

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list requests: %w", err)
	}
	defer rows.Close()

	var requests []*domain.Request
	for rows.Next() {
		var req domain.Request
		err := rows.Scan(
			&req.ID,
			&req.RegionID,
			&req.NeedType,
			&req.Quantity,
			&req.ContactPhone,
			&req.Note,
			&req.Status,
			&req.SubmittedIP,
			&req.SubmittedUserAgent,
			&req.DispatchedBy,
			&req.DispatchedAt,
			&req.CreatedAt,
			&req.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan request: %w", err)
		}
		requests = append(requests, &req)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating requests: %w", err)
	}

	return requests, nil
}

// UpdateStatus updates the status of a request
func (r *RequestRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.RequestStatus, dispatchedBy *uuid.UUID) error {
	query := `
		UPDATE requests
		SET status = $1, dispatched_by = $2, dispatched_at = $3, updated_at = $4
		WHERE id = $5
	`

	var dispatchedAt *time.Time
	if status == domain.StatusDispatched {
		now := time.Now()
		dispatchedAt = &now
	}

	result, err := r.db.Pool.Exec(ctx, query, status, dispatchedBy, dispatchedAt, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update request status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("request not found")
	}

	return nil
}

// Count returns the total count of requests matching the filters
func (r *RequestRepository) Count(ctx context.Context, filters repository.RequestFilters) (int, error) {
	query := "SELECT COUNT(*) FROM requests WHERE 1=1"
	args := []interface{}{}
	argPos := 1

	if filters.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, *filters.Status)
		argPos++
	}

	if filters.RegionID != nil {
		query += fmt.Sprintf(" AND region_id = $%d", argPos)
		args = append(args, *filters.RegionID)
		argPos++
	}

	if filters.FromDate != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argPos)
		args = append(args, *filters.FromDate)
		argPos++
	}

	if filters.ToDate != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argPos)
		args = append(args, *filters.ToDate)
		argPos++
	}

	var count int
	err := r.db.Pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count requests: %w", err)
	}

	return count, nil
}
