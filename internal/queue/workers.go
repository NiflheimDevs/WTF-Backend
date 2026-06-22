package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"gitlab.chabokan.net/niflheim/wtf-backend/internal/domain"
	"gitlab.chabokan.net/niflheim/wtf-backend/internal/repository"
)

// NotifyDispatcherWorker handles dispatcher notification jobs
type NotifyDispatcherWorker struct {
	river.WorkerDefaults[NotifyDispatcherJobArgs]
	auditRepo repository.AuditLogRepository
	logger    *slog.Logger
}

// NewNotifyDispatcherWorker creates a new NotifyDispatcherWorker
func NewNotifyDispatcherWorker(auditRepo repository.AuditLogRepository, logger *slog.Logger) *NotifyDispatcherWorker {
	return &NotifyDispatcherWorker{
		auditRepo: auditRepo,
		logger:    logger,
	}
}

// Work processes the dispatcher notification job
func (w *NotifyDispatcherWorker) Work(ctx context.Context, job *river.Job[NotifyDispatcherJobArgs]) error {
	args := job.Args

	w.logger.Info("processing notify dispatcher job",
		"request_id", args.RequestID,
		"region_id", args.RegionID,
		"attempt", job.Attempt,
	)

	// Simulate SMS notification (in production, this would call an SMS gateway)
	w.logger.Info("simulated SMS notification sent",
		"request_id", args.RequestID,
		"region_id", args.RegionID,
	)

	// Create audit log entry
	payload, err := json.Marshal(map[string]interface{}{
		"request_id": args.RequestID,
		"region_id":  args.RegionID,
		"simulated":  true,
		"timestamp":  time.Now(),
	})
	if err != nil {
		return fmt.Errorf("failed to marshal audit payload: %w", err)
	}

	auditLog := &domain.AuditLog{
		EventType: domain.EventDispatcherNotified,
		RequestID: &args.RequestID,
		Payload:   payload,
		CreatedAt: time.Now(),
	}

	if err := w.auditRepo.Insert(ctx, auditLog); err != nil {
		return fmt.Errorf("failed to insert audit log: %w", err)
	}

	w.logger.Info("notify dispatcher job completed",
		"request_id", args.RequestID,
		"audit_log_id", auditLog.ID,
	)

	return nil
}

// SendRequesterSMSWorker handles requester SMS notification jobs
type SendRequesterSMSWorker struct {
	river.WorkerDefaults[SendRequesterSMSJobArgs]
	auditRepo repository.AuditLogRepository
	logger    *slog.Logger
}

// NewSendRequesterSMSWorker creates a new SendRequesterSMSWorker
func NewSendRequesterSMSWorker(auditRepo repository.AuditLogRepository, logger *slog.Logger) *SendRequesterSMSWorker {
	return &SendRequesterSMSWorker{
		auditRepo: auditRepo,
		logger:    logger,
	}
}

// Work processes the requester SMS notification job
func (w *SendRequesterSMSWorker) Work(ctx context.Context, job *river.Job[SendRequesterSMSJobArgs]) error {
	args := job.Args

	w.logger.Info("processing requester SMS job",
		"request_id", args.RequestID,
		"phone", args.Phone,
		"status", args.Status,
		"attempt", job.Attempt,
	)

	w.logger.Info("simulated requester SMS sent",
		"request_id", args.RequestID,
		"phone", args.Phone,
		"message", fmt.Sprintf("Your water request status is now %s.", args.Status),
	)

	payload, err := json.Marshal(map[string]interface{}{
		"request_id": args.RequestID,
		"phone":      args.Phone,
		"status":     args.Status,
		"simulated":  true,
		"timestamp":  time.Now(),
	})
	if err != nil {
		return fmt.Errorf("failed to marshal audit payload: %w", err)
	}

	auditLog := &domain.AuditLog{
		EventType: domain.EventRequesterSMSSent,
		RequestID: &args.RequestID,
		Payload:   payload,
		CreatedAt: time.Now(),
	}

	if err := w.auditRepo.Insert(ctx, auditLog); err != nil {
		return fmt.Errorf("failed to insert audit log: %w", err)
	}

	w.logger.Info("requester SMS job completed",
		"request_id", args.RequestID,
		"audit_log_id", auditLog.ID,
	)

	return nil
}

// RefreshMetricsWorker handles metrics refresh jobs
type RefreshMetricsWorker struct {
	river.WorkerDefaults[RefreshMetricsJobArgs]
	requestRepo repository.RequestRepository
	metricsRepo repository.MetricsRepository
	logger      *slog.Logger
}

// NewRefreshMetricsWorker creates a new RefreshMetricsWorker
func NewRefreshMetricsWorker(
	requestRepo repository.RequestRepository,
	metricsRepo repository.MetricsRepository,
	logger *slog.Logger,
) *RefreshMetricsWorker {
	return &RefreshMetricsWorker{
		requestRepo: requestRepo,
		metricsRepo: metricsRepo,
		logger:      logger,
	}
}

// Work processes the metrics refresh job
func (w *RefreshMetricsWorker) Work(ctx context.Context, job *river.Job[RefreshMetricsJobArgs]) error {
	args := job.Args

	w.logger.Info("processing refresh metrics job",
		"date", args.Date.Format("2006-01-02"),
		"region_id", args.RegionID,
		"attempt", job.Attempt,
	)

	// Calculate metrics for each need type
	needTypes := []domain.NeedType{domain.NeedTypeBottledWater, domain.NeedTypeTanker}

	for _, needType := range needTypes {
		if err := w.refreshMetricsForNeedType(ctx, args.Date, args.RegionID, needType); err != nil {
			return fmt.Errorf("failed to refresh metrics for need_type %s: %w", needType, err)
		}
	}

	w.logger.Info("refresh metrics job completed",
		"date", args.Date.Format("2006-01-02"),
		"region_id", args.RegionID,
	)

	return nil
}

// refreshMetricsForNeedType calculates and upserts metrics for a specific need type
func (w *RefreshMetricsWorker) refreshMetricsForNeedType(
	ctx context.Context,
	date time.Time,
	regionID uuid.UUID,
	needType domain.NeedType,
) error {
	// Normalize date to start of day
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)

	// Count total requests for this need type
	totalFilters := repository.RequestFilters{
		RegionID: &regionID,
		FromDate: &startOfDay,
		ToDate:   &endOfDay,
	}

	// Get all requests to filter by need type and calculate metrics
	requests, err := w.requestRepo.List(ctx, totalFilters)
	if err != nil {
		return fmt.Errorf("failed to list requests: %w", err)
	}

	// Filter and calculate metrics for this specific need type
	totalCount := 0
	totalQuantity := 0
	pendingCount := 0
	dispatchedCount := 0

	for _, req := range requests {
		if req.NeedType == needType {
			totalCount++
			totalQuantity += req.Quantity

			switch req.Status {
			case domain.StatusPending:
				pendingCount++
			case domain.StatusDispatched:
				dispatchedCount++
			}
		}
	}

	// Upsert metrics
	metrics := &domain.MetricsDaily{
		MetricDate:      startOfDay,
		RegionID:        regionID,
		NeedType:        needType,
		RequestCount:    totalCount,
		TotalQuantity:   totalQuantity,
		PendingCount:    pendingCount,
		DispatchedCount: dispatchedCount,
		UpdatedAt:       time.Now(),
	}

	if err := w.metricsRepo.UpsertDaily(ctx, metrics); err != nil {
		return fmt.Errorf("failed to upsert metrics: %w", err)
	}

	return nil
}
