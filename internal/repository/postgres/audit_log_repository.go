package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gitlab.chabokan.net/niflheim/wtf-backend/internal/domain"
	"gitlab.chabokan.net/niflheim/wtf-backend/internal/repository"
)

// AuditLogRepository implements repository.AuditLogRepository using PostgreSQL
type AuditLogRepository struct {
	db *DB
}

// NewAuditLogRepository creates a new AuditLogRepository
func NewAuditLogRepository(db *DB) repository.AuditLogRepository {
	return &AuditLogRepository{db: db}
}

// Insert creates a new audit log entry
func (r *AuditLogRepository) Insert(ctx context.Context, log *domain.AuditLog) error {
	query := `
		INSERT INTO audit_log (event_type, request_id, actor_user_id, payload, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	err := r.db.Pool.QueryRow(ctx, query,
		log.EventType,
		log.RequestID,
		log.ActorUserID,
		log.Payload,
		log.CreatedAt,
	).Scan(&log.ID)

	if err != nil {
		return fmt.Errorf("failed to insert audit log: %w", err)
	}

	return nil
}

// ListByRequestID returns all audit logs for a specific request
func (r *AuditLogRepository) ListByRequestID(ctx context.Context, requestID uuid.UUID) ([]*domain.AuditLog, error) {
	query := `
		SELECT id, event_type, request_id, actor_user_id, payload, created_at
		FROM audit_log
		WHERE request_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to list audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*domain.AuditLog
	for rows.Next() {
		var log domain.AuditLog
		err := rows.Scan(
			&log.ID,
			&log.EventType,
			&log.RequestID,
			&log.ActorUserID,
			&log.Payload,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, &log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating audit logs: %w", err)
	}

	return logs, nil
}
