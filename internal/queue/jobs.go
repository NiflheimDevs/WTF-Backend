package queue

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// NotifyDispatcherJobArgs represents arguments for dispatcher notification job
type NotifyDispatcherJobArgs struct {
	RequestID uuid.UUID `json:"request_id"`
	RegionID  uuid.UUID `json:"region_id"`
}

// Kind returns the job type identifier for River
func (NotifyDispatcherJobArgs) Kind() string {
	return "notify_dispatcher"
}

// RefreshMetricsJobArgs represents arguments for metrics refresh job
type RefreshMetricsJobArgs struct {
	Date     time.Time `json:"date"`
	RegionID uuid.UUID `json:"region_id"`
}

// Kind returns the job type identifier for River
func (RefreshMetricsJobArgs) Kind() string {
	return "refresh_metrics"
}

// MarshalJSON implements custom JSON marshaling for NotifyDispatcherJobArgs
func (args NotifyDispatcherJobArgs) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RequestID string `json:"request_id"`
		RegionID  string `json:"region_id"`
	}{
		RequestID: args.RequestID.String(),
		RegionID:  args.RegionID.String(),
	})
}

// UnmarshalJSON implements custom JSON unmarshaling for NotifyDispatcherJobArgs
func (args *NotifyDispatcherJobArgs) UnmarshalJSON(data []byte) error {
	var temp struct {
		RequestID string `json:"request_id"`
		RegionID  string `json:"region_id"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	requestID, err := uuid.Parse(temp.RequestID)
	if err != nil {
		return err
	}
	regionID, err := uuid.Parse(temp.RegionID)
	if err != nil {
		return err
	}

	args.RequestID = requestID
	args.RegionID = regionID
	return nil
}

// MarshalJSON implements custom JSON marshaling for RefreshMetricsJobArgs
func (args RefreshMetricsJobArgs) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Date     string `json:"date"`
		RegionID string `json:"region_id"`
	}{
		Date:     args.Date.Format(time.RFC3339),
		RegionID: args.RegionID.String(),
	})
}

// UnmarshalJSON implements custom JSON unmarshaling for RefreshMetricsJobArgs
func (args *RefreshMetricsJobArgs) UnmarshalJSON(data []byte) error {
	var temp struct {
		Date     string `json:"date"`
		RegionID string `json:"region_id"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	date, err := time.Parse(time.RFC3339, temp.Date)
	if err != nil {
		return err
	}
	regionID, err := uuid.Parse(temp.RegionID)
	if err != nil {
		return err
	}

	args.Date = date
	args.RegionID = regionID
	return nil
}
