package domain

import "github.com/google/uuid"

// Region represents a geographic area (district/neighborhood)
type Region struct {
	ID           uuid.UUID  `json:"id"`
	NameFa       string     `json:"name_fa"`
	NameEn       string     `json:"name_en"`
	ParentID     *uuid.UUID `json:"parent_id,omitempty"`
	IsActive     bool       `json:"is_active"`
	DisplayOrder int        `json:"display_order"`
}
