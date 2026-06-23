package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/cuffeyvidzro/leamout/internal/modules/auth"
	"github.com/cuffeyvidzro/leamout/internal/modules/session"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	ContextAuthUser    = "auth_user"
	ContextAuthSession = "auth_session"
	ContextUserID      = "userID"
)

type AuthRepository interface {
	FindSessionByTokenHash(ctx context.Context, tokenHash string) (*auth.Session, error)
	FindUserByID(ctx context.Context, id uuid.UUID) (*auth.User, error)
	TouchSession(ctx context.Context, id uuid.UUID) error
}

func RequireAuth(repository AuthRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawToken, err := c.Cookie(auth.SessionCookieName)
		if err != nil || rawToken == "" {
			abortUnauthorized(c)
			return
		}

		session, err := repository.FindSessionByTokenHash(c.Request.Context(), session.HashToken(rawToken))
		if err != nil {
			abortUnauthorized(c)
			return
		}
		if session == nil || session.RevokedAt != nil || !session.ExpiresAt.After(time.Now()) {
			abortUnauthorized(c)
			return
		}

		user, err := repository.FindUserByID(c.Request.Context(), session.UserID)
		if err != nil {
			abortUnauthorized(c)
			return
		}
		if user == nil || user.Status != auth.UserStatusActive {
			abortUnauthorized(c)
			return
		}

		if err := repository.TouchSession(c.Request.Context(), session.ID); err != nil {
			abortUnauthorized(c)
			return
		}

		c.Set(ContextAuthUser, user)
		c.Set(ContextAuthSession, session)
		c.Set(ContextUserID, user.ID)
		c.Next()
	}
}

func CurrentUser(c *gin.Context) (*auth.User, bool) {
	value, ok := c.Get(ContextAuthUser)
	if !ok {
		return nil, false
	}

	user, ok := value.(*auth.User)
	return user, ok
}

func CurrentSession(c *gin.Context) (*auth.Session, bool) {
	value, ok := c.Get(ContextAuthSession)
	if !ok {
		return nil, false
	}

	session, ok := value.(*auth.Session)
	return session, ok
}

func abortUnauthorized(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error": "unauthorized",
	})
}
