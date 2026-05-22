package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"gitlab.chabokan.net/niflheim/wtf-backend/internal/domain"
	"gitlab.chabokan.net/niflheim/wtf-backend/internal/queue"
	"gitlab.chabokan.net/niflheim/wtf-backend/internal/repository"
)

type CreateRequestInput struct {
	RegionID           uuid.UUID
	NeedType           domain.NeedType
	Quantity           int
	ContactPhone       *string
	Note               *string
	SubmittedIP        *net.IPNet
	SubmittedUserAgent *string
}

type RequestListResult struct {
	Requests []*domain.Request `json:"requests"`
	Total    int               `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}

type RequestDetail struct {
	Request  *domain.Request    `json:"request"`
	AuditLog []*domain.AuditLog `json:"audit_log"`
}

type RequestService struct {
	requests repository.RequestRepository
	regions  repository.RegionRepository
	audit    repository.AuditLogRepository
	queue    *queue.Client
}

func NewRequestService(
	requests repository.RequestRepository,
	regions repository.RegionRepository,
	audit repository.AuditLogRepository,
	queue *queue.Client,
) *RequestService {
	return &RequestService{
		requests: requests,
		regions:  regions,
		audit:    audit,
		queue:    queue,
	}
}

func (s *RequestService) Create(ctx context.Context, input CreateRequestInput) (*domain.Request, error) {
	// Validate region exists
	if _, err := s.regions.FindByID(ctx, input.RegionID); err != nil {
		return nil, fmt.Errorf("%w: region", ErrNotFound)
	}

	now := time.Now().UTC()
	req := &domain.Request{
		ID:                 uuid.New(),
		RegionID:           input.RegionID,
		NeedType:           input.NeedType,
		Quantity:           input.Quantity,
		ContactPhone:       trimStringPtr(input.ContactPhone),
		Note:               trimStringPtr(input.Note),
		Status:             domain.StatusPending,
		SubmittedIP:        input.SubmittedIP,
		SubmittedUserAgent: input.SubmittedUserAgent,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	// Create request in database
	if err := s.requests.Create(ctx, req); err != nil {
		return nil, err
	}

	// Insert audit log
	_ = s.insertAudit(ctx, domain.EventRequestSubmitted, &req.ID, nil, map[string]any{
		"region_id": req.RegionID,
		"need_type": req.NeedType,
		"quantity":  req.Quantity,
	})

	// Enqueue background jobs asynchronously
	// Note: In a production system with transactional guarantees, you would use
	// InsertTx within a database transaction. For now, we enqueue after the request is created.
	go func() {
		bgCtx := context.Background()

		// Enqueue dispatcher notification job
		if err := s.queue.EnqueueNotifyDispatcher(bgCtx, queue.NotifyDispatcherJobArgs{
			RequestID: req.ID,
			RegionID:  req.RegionID,
		}); err != nil {
			// Log error but don't fail the request creation
			// In production, you'd want proper error tracking here
			fmt.Printf("failed to enqueue notify dispatcher job: %v\n", err)
		}

		// Enqueue metrics refresh job
		if err := s.queue.EnqueueRefreshMetrics(bgCtx, queue.RefreshMetricsJobArgs{
			Date:     now,
			RegionID: req.RegionID,
		}); err != nil {
			fmt.Printf("failed to enqueue refresh metrics job: %v\n", err)
		}
	}()

	return req, nil
}

func (s *RequestService) List(ctx context.Context, filters repository.RequestFilters, page, pageSize int) (*RequestListResult, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	filters.Limit = pageSize
	filters.Offset = (page - 1) * pageSize

	total, err := s.requests.Count(ctx, filters)
	if err != nil {
		return nil, err
	}

	items, err := s.requests.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	return &RequestListResult{Requests: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *RequestService) Detail(ctx context.Context, id uuid.UUID) (*RequestDetail, error) {
	req, err := s.requests.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: request", ErrNotFound)
	}

	logs, err := s.audit.ListByRequestID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &RequestDetail{Request: req, AuditLog: logs}, nil
}

func (s *RequestService) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.RequestStatus, actorID uuid.UUID) (*domain.Request, error) {
	req, err := s.requests.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: request", ErrNotFound)
	}

	if !req.CanTransitionTo(status) {
		return nil, ErrInvalidTransition
	}

	var dispatchedBy *uuid.UUID
	if status == domain.StatusDispatched {
		dispatchedBy = &actorID
	}

	if err := s.requests.UpdateStatus(ctx, id, status, dispatchedBy); err != nil {
		return nil, err
	}

	_ = s.insertAudit(ctx, domain.EventRequestStatusChanged, &id, &actorID, map[string]any{
		"from": req.Status,
		"to":   status,
	})

	// Enqueue metrics refresh when status changes
	go func() {
		bgCtx := context.Background()
		if err := s.queue.EnqueueRefreshMetrics(bgCtx, queue.RefreshMetricsJobArgs{
			Date:     time.Now(),
			RegionID: req.RegionID,
		}); err != nil {
			fmt.Printf("failed to enqueue refresh metrics job: %v\n", err)
		}
	}()

	updated, err := s.requests.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *RequestService) insertAudit(ctx context.Context, event domain.EventType, requestID, actorID *uuid.UUID, payload map[string]any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return s.audit.Insert(ctx, &domain.AuditLog{
		EventType:   event,
		RequestID:   requestID,
		ActorUserID: actorID,
		Payload:     body,
		CreatedAt:   time.Now().UTC(),
	})
}

func trimStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
