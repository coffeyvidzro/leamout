package session

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestServiceListByUserID(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repository := &fakeRepository{
		sessions: []Session{{
			ID:        uuid.New(),
			UserID:    userID,
			ExpiresAt: time.Now().Add(time.Hour),
		}},
	}
	service := NewService(repository)

	sessions, err := service.ListByUserID(context.Background(), userID)
	if err != nil {
		t.Fatalf("ListByUserID() error = %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("sessions = %d, want 1", len(sessions))
	}
	if repository.listUserID != userID {
		t.Fatalf("listUserID = %s, want %s", repository.listUserID, userID)
	}
}

func TestServiceRevokeSpecificSession(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	sessionID := uuid.New()
	repository := &fakeRepository{}
	service := NewService(repository)

	if err := service.RevokeSpecificSession(context.Background(), userID, sessionID); err != nil {
		t.Fatalf("RevokeSpecificSession() error = %v", err)
	}
	if repository.revokedUserID != userID || repository.revokedSessionID != sessionID {
		t.Fatalf("revoked (%s, %s), want (%s, %s)", repository.revokedUserID, repository.revokedSessionID, userID, sessionID)
	}
}

func TestServiceRevokeAllUserSessions(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repository := &fakeRepository{}
	service := NewService(repository)

	if err := service.RevokeAllUserSessions(context.Background(), userID); err != nil {
		t.Fatalf("RevokeAllUserSessions() error = %v", err)
	}
	if repository.revokedAllUserID != userID {
		t.Fatalf("revokedAllUserID = %s, want %s", repository.revokedAllUserID, userID)
	}
}

type fakeRepository struct {
	sessions         []Session
	listUserID       uuid.UUID
	revokedUserID    uuid.UUID
	revokedSessionID uuid.UUID
	revokedAllUserID uuid.UUID
}

func (r *fakeRepository) Create(ctx context.Context, params CreateParams) (*Session, error) {
	session := &Session{
		ID:        uuid.New(),
		UserID:    params.UserID,
		TokenHash: params.TokenHash,
		ExpiresAt: params.ExpiresAt,
	}
	return session, nil
}

func (r *fakeRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]Session, error) {
	r.listUserID = userID
	return r.sessions, nil
}

func (r *fakeRepository) GetByID(ctx context.Context, id uuid.UUID) (*Session, error) {
	for _, session := range r.sessions {
		if session.ID == id {
			return &session, nil
		}
	}
	return nil, ErrNotFound
}

func (r *fakeRepository) RevokeByID(ctx context.Context, userID, id uuid.UUID) error {
	r.revokedUserID = userID
	r.revokedSessionID = id
	return nil
}

func (r *fakeRepository) RevokeAllByUserID(ctx context.Context, userID uuid.UUID) error {
	r.revokedAllUserID = userID
	return nil
}

func (r *fakeRepository) RevokeByTokenHash(ctx context.Context, tokenHash string) error {
	return nil
}
