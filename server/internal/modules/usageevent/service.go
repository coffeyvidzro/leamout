package usageevent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidEvent = errors.New("invalid usage event")

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Ingest(ctx context.Context, userID uuid.UUID, req IngestRequest) (*IngestResponse, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("%w: missing user id", ErrInvalidEvent)
	}
	if len(req.Events) == 0 {
		return nil, fmt.Errorf("%w: events are required", ErrInvalidEvent)
	}
	if len(req.Events) > 1000 {
		return nil, fmt.Errorf("%w: events cannot exceed 1000 per request", ErrInvalidEvent)
	}

	events := make([]CreateParams, 0, len(req.Events))
	for _, item := range req.Events {
		event, err := normalizeCreateRequest(item)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return s.repo.Ingest(ctx, userID, events)
}

func (s *Service) List(ctx context.Context, params ListParams) (*ListResponse, error) {
	if params.UserID == uuid.Nil {
		return nil, fmt.Errorf("%w: missing user id", ErrInvalidEvent)
	}
	if params.Source != "" && params.Source != SourceUser && params.Source != SourceSystem {
		return nil, fmt.Errorf("%w: invalid source", ErrInvalidEvent)
	}
	if params.StartTimestamp != nil && params.EndTimestamp != nil && params.StartTimestamp.After(*params.EndTimestamp) {
		return nil, fmt.Errorf("%w: start_timestamp must be before end_timestamp", ErrInvalidEvent)
	}

	return s.repo.List(ctx, params)
}

func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (*UsageEvent, error) {
	if userID == uuid.Nil || id == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid event id", ErrInvalidEvent)
	}

	return s.repo.Get(ctx, userID, id)
}

func normalizeCreateRequest(req EventCreateRequest) (CreateParams, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return CreateParams{}, fmt.Errorf("%w: name is required", ErrInvalidEvent)
	}

	metadata := req.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}

	timestamp := nowUTC()
	if req.Timestamp != nil {
		timestamp = req.Timestamp.UTC()
	}
	if timestamp.After(time.Now().UTC().Add(5 * time.Minute)) {
		return CreateParams{}, fmt.Errorf("%w: timestamp cannot be more than 5 minutes in the future", ErrInvalidEvent)
	}

	return CreateParams{
		Timestamp:          timestamp,
		Name:               name,
		Source:             SourceUser,
		CustomerID:         req.CustomerID,
		ExternalCustomerID: optionalString(req.ExternalCustomerID),
		ExternalID:         optionalString(req.ExternalID),
		ParentID:           req.ParentID,
		Metadata:           metadata,
	}, nil
}

func optionalString(value *string) string {
	if value == nil {
		return ""
	}

	return strings.TrimSpace(*value)
}
