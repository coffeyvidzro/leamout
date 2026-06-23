package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cuffeyvidzro/leamout/internal/modules/auth"
	"github.com/cuffeyvidzro/leamout/internal/modules/auth/oauth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestRequireAuthRejectsMissingCookie(t *testing.T) {
	t.Parallel()

	router := authTestRouter(&authMiddlewareRepository{})
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestRequireAuthSetsCurrentUserAndSession(t *testing.T) {
	t.Parallel()

	rawToken := "raw-session-token"
	user := &auth.User{
		ID:     uuid.New(),
		Status: auth.UserStatusActive,
	}
	session := &auth.Session{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: auth.HashSessionToken(rawToken),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	repository := &authMiddlewareRepository{
		user:    user,
		session: session,
	}
	router := authTestRouter(repository)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: rawToken})

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !repository.touched {
		t.Fatal("expected session to be touched")
	}
}

func authTestRouter(repository auth.Repository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/protected", RequireAuth(repository), func(c *gin.Context) {
		if _, ok := CurrentUser(c); !ok {
			c.Status(http.StatusInternalServerError)
			return
		}
		if _, ok := CurrentSession(c); !ok {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusOK)
	})
	return router
}

type authMiddlewareRepository struct {
	user    *auth.User
	session *auth.Session
	touched bool
}

func (r *authMiddlewareRepository) FindUserByEmail(ctx context.Context, email string) (*auth.User, error) {
	return nil, nil
}

func (r *authMiddlewareRepository) FindUserByID(ctx context.Context, id uuid.UUID) (*auth.User, error) {
	return r.user, nil
}

func (r *authMiddlewareRepository) CreateUser(ctx context.Context, profile *oauth.Profile) (*auth.User, error) {
	return nil, nil
}

func (r *authMiddlewareRepository) FindAccount(ctx context.Context, provider, providerUserID string) (*auth.Account, error) {
	return nil, nil
}

func (r *authMiddlewareRepository) CreateAccount(ctx context.Context, userID uuid.UUID, profile *oauth.Profile) (*auth.Account, error) {
	return nil, nil
}

func (r *authMiddlewareRepository) CreateSession(ctx context.Context, session auth.CreateSessionParams) (*auth.Session, error) {
	return nil, nil
}

func (r *authMiddlewareRepository) FindSessionByTokenHash(ctx context.Context, tokenHash string) (*auth.Session, error) {
	if r.session == nil || r.session.TokenHash != tokenHash {
		return nil, nil
	}
	return r.session, nil
}

func (r *authMiddlewareRepository) TouchSession(ctx context.Context, id uuid.UUID) error {
	r.touched = true
	return nil
}

func (r *authMiddlewareRepository) RevokeSessionByTokenHash(ctx context.Context, tokenHash string) error {
	return nil
}
