package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventType represents the type of audit event
type EventType string

const (
	EventRequestSubmitted     EventType = "request_submitted"
	EventRequestStatusChanged EventType = "request_status_changed"
	EventDispatcherNotified   EventType = "dispatcher_notified"
	EventRequesterSMSSent     EventType = "requester_sms_sent"
	EventMetricsRefreshed     EventType = "metrics_refreshed"
)

// AuditLog represents an immutable audit trail entry
type AuditLog struct {
	ID          int64           `json:"id"`
	EventType   EventType       `json:"event_type"`
	RequestID   *uuid.UUID      `json:"request_id,omitempty"`
	ActorUserID *uuid.UUID      `json:"actor_user_id,omitempty"`
	Payload     json.RawMessage `json:"payload"`
	CreatedAt   time.Time       `json:"created_at"`
}
