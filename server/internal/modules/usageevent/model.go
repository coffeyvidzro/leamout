package usageevent

import (
	"time"

	"github.com/google/uuid"
)

type Source string

const (
	SourceSystem Source = "system"
	SourceUser   Source = "user"
)

type UsageEvent struct {
	ID                 uuid.UUID      `json:"id"`
	UserID             uuid.UUID      `json:"user_id"`
	ParentID           *uuid.UUID     `json:"parent_id,omitempty"`
	Timestamp          time.Time      `json:"timestamp"`
	Name               string         `json:"name"`
	Source             Source         `json:"source"`
	CustomerID         *uuid.UUID     `json:"customer_id,omitempty"`
	ExternalCustomerID *string        `json:"external_customer_id,omitempty"`
	ExternalID         *string        `json:"external_id,omitempty"`
	Metadata           map[string]any `json:"metadata"`
	ChildCount         int            `json:"child_count"`
	CreatedAt          time.Time      `json:"created_at"`
}

type IngestRequest struct {
	Events []EventCreateRequest `json:"events" binding:"required,min=1,max=1000,dive"`
}

type EventCreateRequest struct {
	Timestamp          *time.Time     `json:"timestamp,omitempty"`
	Name               string         `json:"name" binding:"required,min=1,max=200"`
	CustomerID         *uuid.UUID     `json:"customer_id,omitempty"`
	ExternalCustomerID *string        `json:"external_customer_id,omitempty" binding:"omitempty,max=200"`
	ExternalID         *string        `json:"external_id,omitempty" binding:"omitempty,max=200"`
	ParentID           *uuid.UUID     `json:"parent_id,omitempty"`
	Metadata           map[string]any `json:"metadata,omitempty"`
}

type CreateParams struct {
	Timestamp          time.Time
	Name               string
	Source             Source
	CustomerID         *uuid.UUID
	ExternalCustomerID string
	ExternalID         string
	ParentID           *uuid.UUID
	Metadata           map[string]any
}

type IngestResponse struct {
	Inserted   int `json:"inserted"`
	Duplicates int `json:"duplicates"`
}

type ListParams struct {
	UserID             uuid.UUID
	Name               string
	Source             Source
	CustomerID         *uuid.UUID
	ExternalCustomerID string
	ParentID           *uuid.UUID
	StartTimestamp     *time.Time
	EndTimestamp       *time.Time
	Page               int
	Limit              int
	Sorting            string
}

type ListResponse struct {
	Items      []UsageEvent `json:"items"`
	Pagination Pagination   `json:"pagination"`
}

type Pagination struct {
	TotalCount int `json:"total_count"`
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	MaxPage    int `json:"max_page"`
}
