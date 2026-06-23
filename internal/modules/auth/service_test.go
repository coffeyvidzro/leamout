package auth

import (
	"context"
	"testing"
	"time"

	"github.com/cuffeyvidzro/leamout/internal/modules/auth/oauth"
	"github.com/cuffeyvidzro/leamout/internal/modules/session"
	"github.com/google/uuid"
)

func TestCompleteOAuthLoginUpsertsUserAndCreatesSession(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	authRepo := &fakeRepository{user: &User{
		ID:            userID,
		Name:          "Ada Lovelace",
		Email:         "ada@example.com",
		EmailVerified: true,
		Status:        UserStatusActive,
	}}
	sessionRepo := &fakeSessionRepository{}
	provider := &fakeProvider{profile: &oauth.Profile{
		Provider:       oauth.ProviderGoogle,
		ProviderUserID: "provider-user-id",
		Email:          "ada@example.com",
		EmailVerified:  true,
		Name:           "Ada Lovelace",
		AvatarURL:      "https://example.com/avatar.png",
	}}
	service := NewService(authRepo, oauth.NewRegistry(provider), session.NewService(sessionRepo))

	response, err := service.CompleteOAuthLogin(context.Background(), OAuthLoginRequest{
		Provider:  oauth.ProviderGoogle,
		Code:      "code",
		State:     "state",
		UserAgent: "go-test",
		IPAddress: "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("CompleteOAuthLogin() error = %v", err)
	}
	if response.User.Email != "ada@example.com" {
		t.Fatalf("response.User.Email = %q, want ada@example.com", response.User.Email)
	}
	if response.Session.Token == "" {
		t.Fatal("expected session token")
	}
	if !authRepo.upserted {
		t.Fatal("expected oauth user upsert")
	}
	if sessionRepo.created == nil || sessionRepo.created.UserID != userID {
		t.Fatal("expected session to be created for oauth user")
	}
	if sessionRepo.created.TokenHash != session.HashToken(response.Session.Token) {
		t.Fatal("expected session token hash to be persisted")
	}
}

func TestCompleteOAuthLoginRejectsUnverifiedEmail(t *testing.T) {
	t.Parallel()

	service := NewService(
		&fakeRepository{},
		oauth.NewRegistry(&fakeProvider{profile: &oauth.Profile{Provider: oauth.ProviderGoogle, EmailVerified: false}}),
		session.NewService(&fakeSessionRepository{}),
	)

	_, err := service.CompleteOAuthLogin(context.Background(), OAuthLoginRequest{
		Provider: oauth.ProviderGoogle,
		Code:     "code",
	})
	if err != ErrUnverifiedEmail {
		t.Fatalf("CompleteOAuthLogin() error = %v, want %v", err, ErrUnverifiedEmail)
	}
}

func TestNewOAuthStateAndOAuthURL(t *testing.T) {
	t.Parallel()

	provider := &fakeProvider{profile: &oauth.Profile{Provider: oauth.ProviderGoogle}}
	service := NewService(&fakeRepository{}, oauth.NewRegistry(provider), session.NewService(&fakeSessionRepository{}))

	state, err := service.NewOAuthState()
	if err != nil {
		t.Fatalf("NewOAuthState() error = %v", err)
	}
	if state == "" {
		t.Fatal("expected oauth state")
	}

	url, err := service.OAuthURL(oauth.ProviderGoogle, state)
	if err != nil {
		t.Fatalf("OAuthURL() error = %v", err)
	}
	if url == "https://example.com?state=" {
		t.Fatal("expected state in oauth URL")
	}
}

type fakeProvider struct {
	profile *oauth.Profile
}

func (p *fakeProvider) Name() string { return p.profile.Provider }

func (p *fakeProvider) AuthURL(state string) string { return "https://example.com?state=" + state }

func (p *fakeProvider) Exchange(ctx context.Context, code string) (*oauth.Profile, error) {
	return p.profile, nil
}

type fakeRepository struct {
	user     *User
	upserted bool
}

func (r *fakeRepository) FindUserByEmail(ctx context.Context, email string) (*User, error) {
	return r.user, nil
}

func (r *fakeRepository) FindUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	return r.user, nil
}

func (r *fakeRepository) CreateUser(ctx context.Context, profile *oauth.Profile) (*User, error) {
	return r.user, nil
}

func (r *fakeRepository) UpsertOAuthUser(ctx context.Context, profile *oauth.Profile) (*User, error) {
	r.upserted = true
	return r.user, nil
}

func (r *fakeRepository) FindAccount(ctx context.Context, provider, providerUserID string) (*Account, error) {
	return nil, nil
}

func (r *fakeRepository) CreateAccount(ctx context.Context, userID uuid.UUID, profile *oauth.Profile) (*Account, error) {
	return nil, nil
}

func (r *fakeRepository) CreateSession(ctx context.Context, params CreateSessionParams) (*Session, error) {
	return nil, nil
}

func (r *fakeRepository) FindSessionByTokenHash(ctx context.Context, tokenHash string) (*Session, error) {
	return nil, nil
}

func (r *fakeRepository) TouchSession(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (r *fakeRepository) RevokeSessionByTokenHash(ctx context.Context, tokenHash string) error {
	return nil
}

type fakeSessionRepository struct {
	created   *session.CreateParams
	revokedBy string
}

func (r *fakeSessionRepository) Create(ctx context.Context, params session.CreateParams) (*session.Session, error) {
	r.created = &params
	return &session.Session{
		ID:        uuid.New(),
		UserID:    params.UserID,
		TokenHash: params.TokenHash,
		ExpiresAt: params.ExpiresAt,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (r *fakeSessionRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]session.Session, error) {
	return nil, nil
}

func (r *fakeSessionRepository) GetByID(ctx context.Context, id uuid.UUID) (*session.Session, error) {
	return nil, session.ErrNotFound
}

func (r *fakeSessionRepository) RevokeByID(ctx context.Context, userID, id uuid.UUID) error {
	return nil
}

func (r *fakeSessionRepository) RevokeAllByUserID(ctx context.Context, userID uuid.UUID) error {
	return nil
}

func (r *fakeSessionRepository) RevokeByTokenHash(ctx context.Context, tokenHash string) error {
	r.revokedBy = tokenHash
	return nil
}
