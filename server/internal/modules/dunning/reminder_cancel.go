package dunning

import (
	"context"

	"github.com/google/uuid"
)

func (s *Service) MarkAttemptCanceled(ctx context.Context, attemptID uuid.UUID, metadata map[string]any) error {
	if err := s.validateAttemptTransition(ctx, attemptID, AttemptStatusCanceled); err != nil {
		return err
	}
	return s.repository.MarkAttemptCanceled(ctx, attemptID, metadata)
}

func (r *Repository) MarkAttemptCanceled(ctx context.Context, attemptID uuid.UUID, metadata map[string]any) error {
	return r.markAttemptCanceled(ctx, attemptID, metadata)
}
