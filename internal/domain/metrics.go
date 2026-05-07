package domain

import (
	"time"

	"github.com/google/uuid"
)

// MetricsDaily represents aggregated daily metrics per region and need type
type MetricsDaily struct {
	MetricDate      time.Time `json:"metric_date"`
	RegionID        uuid.UUID `json:"region_id"`
	NeedType        NeedType  `json:"need_type"`
	RequestCount    int       `json:"request_count"`
	TotalQuantity   int       `json:"total_quantity"`
	PendingCount    int       `json:"pending_count"`
	DispatchedCount int       `json:"dispatched_count"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// MetricsSummary represents overall system metrics
type MetricsSummary struct {
	TotalRequests      int `json:"total_requests"`
	PendingRequests    int `json:"pending_requests"`
	DispatchedRequests int `json:"dispatched_requests"`
	FulfilledRequests  int `json:"fulfilled_requests"`
}

// RegionMetrics represents metrics for a specific region
type RegionMetrics struct {
	RegionID     uuid.UUID `json:"region_id"`
	RegionNameFa string    `json:"region_name_fa"`
	RegionNameEn string    `json:"region_name_en"`
	RequestCount int       `json:"request_count"`
}

// NeedTypeMetrics represents metrics by need type
type NeedTypeMetrics struct {
	NeedType     NeedType `json:"need_type"`
	RequestCount int      `json:"request_count"`
	TotalQuantity int     `json:"total_quantity"`
}
