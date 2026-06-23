package auth

import (
	"context"
	"testing"
	"time"

	"github.com/cuffeyvidzro/leamout/internal/modules/auth/oauth"
	"github.com/google/uuid"
)

func TestCompleteOAuthLoginCreatesUserAccountAndSession(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	store := &fakeStateStore{states: map[string]bool{"google:state": true}}
	provider := &fakeProvider{profile: &oauth.Profile{
		Provider:       oauth.ProviderGoogle,
		ProviderUserID: "provider-user-id",
		Email:          "ada@example.com",
		EmailVerified:  true,
		Name:           "Ada Lovelace",
		AvatarURL:      "https://example.com/avatar.png",
	}}
	service := NewService(oauth.NewRegistry(provider), repo, store)

	response, rawToken, err := service.CompleteOAuthLogin(context.Background(), OAuthLoginRequest{
		Provider:  oauth.ProviderGoogle,
		Code:      "code",
		State:     "state",
		UserAgent: "go-test",
		IPAddress: "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("CompleteOAuthLogin() error = %v", err)
	}
	if rawToken == "" {
		t.Fatal("expected raw session token")
	}
	if response.User.Email != "ada@example.com" {
		t.Fatalf("response.User.Email = %q, want ada@example.com", response.User.Email)
	}
	if len(repo.accounts) != 1 {
		t.Fatalf("created accounts = %d, want 1", len(repo.accounts))
	}
	if len(repo.sessions) != 1 {
		t.Fatalf("created sessions = %d, want 1", len(repo.sessions))
	}
	if repo.sessions[0].TokenHash != HashSessionToken(rawToken) {
		t.Fatal("session was not persisted with hashed raw token")
	}
}

func TestCompleteOAuthLoginRejectsInvalidState(t *testing.T) {
	t.Parallel()

	service := NewService(
		oauth.NewRegistry(&fakeProvider{profile: &oauth.Profile{Provider: oauth.ProviderGoogle}}),
		newFakeRepository(),
		&fakeStateStore{states: map[string]bool{}},
	)

	_, _, err := service.CompleteOAuthLogin(context.Background(), OAuthLoginRequest{
		Provider: oauth.ProviderGoogle,
		Code:     "code",
		State:    "missing",
	})
	if err != ErrInvalidOAuthState {
		t.Fatalf("CompleteOAuthLogin() error = %v, want %v", err, ErrInvalidOAuthState)
	}
}

func TestHashSessionTokenIsDeterministicAndDoesNotExposeRawToken(t *testing.T) {
	t.Parallel()

	first := HashSessionToken("session-token")
	second := HashSessionToken("session-token")
	if first != second {
		t.Fatal("expected deterministic hash")
	}
	if first == "session-token" {
		t.Fatal("expected hashed token to differ from raw token")
	}
	if len(first) != 64 {
		t.Fatalf("hash length = %d, want 64", len(first))
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

type fakeStateStore struct {
	states map[string]bool
}

func (s *fakeStateStore) SaveOAuthState(ctx context.Context, provider, state string, ttl time.Duration) error {
	s.states[provider+":"+state] = true
	return nil
}

func (s *fakeStateStore) ConsumeOAuthState(ctx context.Context, provider, state string) error {
	key := provider + ":" + state
	if !s.states[key] {
		return ErrInvalidOAuthState
	}
	delete(s.states, key)
	return nil
}

type fakeRepository struct {
	users    map[uuid.UUID]*User
	emails   map[string]uuid.UUID
	accounts map[string]*Account
	sessions []*Session
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		users:    map[uuid.UUID]*User{},
		emails:   map[string]uuid.UUID{},
		accounts: map[string]*Account{},
	}
}

func (r *fakeRepository) FindUserByEmail(ctx context.Context, email string) (*User, error) {
	id, ok := r.emails[email]
	if !ok {
		return nil, nil
	}
	return r.users[id], nil
}

func (r *fakeRepository) FindUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	return r.users[id], nil
}

func (r *fakeRepository) CreateUser(ctx context.Context, profile *oauth.Profile) (*User, error) {
	id := uuid.New()
	avatarURL := profile.AvatarURL
	user := &User{
		ID:            id,
		Name:          profile.Name,
		Email:         profile.Email,
		EmailVerified: profile.EmailVerified,
		AvatarURL:     &avatarURL,
		Status:        UserStatusActive,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	r.users[id] = user
	r.emails[user.Email] = id
	return user, nil
}

func (r *fakeRepository) FindAccount(ctx context.Context, provider, providerUserID string) (*Account, error) {
	return r.accounts[provider+":"+providerUserID], nil
}

func (r *fakeRepository) CreateAccount(ctx context.Context, userID uuid.UUID, profile *oauth.Profile) (*Account, error) {
	account := &Account{
		ID:             uuid.New(),
		UserID:         userID,
		Provider:       profile.Provider,
		ProviderUserID: profile.ProviderUserID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	r.accounts[profile.Provider+":"+profile.ProviderUserID] = account
	return account, nil
}

func (r *fakeRepository) CreateSession(ctx context.Context, params CreateSessionParams) (*Session, error) {
	session := &Session{
		ID:        uuid.New(),
		UserID:    params.UserID,
		TokenHash: params.TokenHash,
		ExpiresAt: params.ExpiresAt,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	r.sessions = append(r.sessions, session)
	return session, nil
}

func (r *fakeRepository) RevokeSessionByTokenHash(ctx context.Context, tokenHash string) error {
	return nil
}
