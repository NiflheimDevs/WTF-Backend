package domain

import (
	"time"

	"github.com/google/uuid"
)

// NeedType represents the type of water supply needed
type NeedType string

const (
	NeedTypeBottledWater NeedType = "bottled_water"
	NeedTypeTanker       NeedType = "tanker"
)

// RequestStatus represents the current status of a request
type RequestStatus string

const (
	StatusPending    RequestStatus = "pending"
	StatusDispatched RequestStatus = "dispatched"
	StatusFulfilled  RequestStatus = "fulfilled"
	StatusCancelled  RequestStatus = "cancelled"
)

// Request represents a water supply request from a citizen
type Request struct {
	ID                uuid.UUID      `json:"id"`
	RegionID          uuid.UUID      `json:"region_id"`
	NeedType          NeedType       `json:"need_type"`
	Quantity          int            `json:"quantity"`
	ContactPhone      *string        `json:"contact_phone,omitempty"`
	Note              *string        `json:"note,omitempty"`
	Status            RequestStatus  `json:"status"`
	SubmittedIP       *string        `json:"submitted_ip,omitempty"`
	SubmittedUserAgent *string       `json:"submitted_user_agent,omitempty"`
	DispatchedBy      *uuid.UUID     `json:"dispatched_by,omitempty"`
	DispatchedAt      *time.Time     `json:"dispatched_at,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

// IsPending checks if request is in pending status
func (r *Request) IsPending() bool {
	return r.Status == StatusPending
}

// IsDispatched checks if request has been dispatched
func (r *Request) IsDispatched() bool {
	return r.Status == StatusDispatched
}

// IsFulfilled checks if request has been fulfilled
func (r *Request) IsFulfilled() bool {
	return r.Status == StatusFulfilled
}

// CanTransitionTo checks if status transition is valid
func (r *Request) CanTransitionTo(newStatus RequestStatus) bool {
	switch r.Status {
	case StatusPending:
		return newStatus == StatusDispatched || newStatus == StatusCancelled
	case StatusDispatched:
		return newStatus == StatusFulfilled || newStatus == StatusCancelled
	case StatusFulfilled, StatusCancelled:
		return false // Terminal states
	default:
		return false
	}
}
