package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/cuffeyvidzro/leamout/internal/modules/pat"
	"github.com/cuffeyvidzro/leamout/internal/modules/session"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	SessionCookieName = "lmt-session"
	SessionContextKey = "auth.session"
	SubjectContextKey = "auth.subject"
	ContextUserID     = "userID"
	AuthMethodSession = "session"
	AuthMethodPAT     = "pat"
)

type SessionService interface {
	GetByToken(ctx context.Context, token string) (*session.Session, error)
}

type PATService interface {
	Authenticate(ctx context.Context, rawToken string) (*pat.Token, error)
}

type AuthSubject struct {
	UserID  uuid.UUID
	Method  string
	TokenID *uuid.UUID
}

func AuthMiddleware(sessionService SessionService, patService PATService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if authenticateBearerPAT(c, patService) {
			c.Next()
			return
		}

		if authenticateSession(c, sessionService) {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "missing or invalid authentication",
		})
	}
}

func SessionAuthMiddleware(sessionService SessionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !authenticateSession(c, sessionService) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing or invalid session",
			})
			return
		}

		c.Next()
	}
}

func authenticateSession(c *gin.Context, sessionService SessionService) bool {
	cookie, err := c.Cookie(SessionCookieName)
	if err != nil || cookie == "" {
		return false
	}

	sess, err := sessionService.GetByToken(c.Request.Context(), cookie)
	if err != nil {
		return false
	}

	c.Set(SessionContextKey, sess)
	c.Set(ContextUserID, sess.UserID)
	c.Set(SubjectContextKey, AuthSubject{UserID: sess.UserID, Method: AuthMethodSession})
	return true
}

func authenticateBearerPAT(c *gin.Context, patService PATService) bool {
	if patService == nil {
		return false
	}

	authorization := c.GetHeader("Authorization")
	rawToken, ok := strings.CutPrefix(authorization, "Bearer ")
	if !ok || rawToken == "" {
		return false
	}

	token, err := patService.Authenticate(c.Request.Context(), rawToken)
	if err != nil {
		return false
	}

	c.Set(ContextUserID, token.UserID)
	c.Set(SubjectContextKey, AuthSubject{UserID: token.UserID, Method: AuthMethodPAT, TokenID: &token.ID})
	return true
}

func GetSession(c *gin.Context) (*session.Session, bool) {
	value, ok := c.Get(SessionContextKey)
	if !ok {
		return nil, false
	}

	sess, ok := value.(*session.Session)
	return sess, ok
}

func MustGetSession(c *gin.Context) *session.Session {
	sess, ok := GetSession(c)
	if !ok {
		panic("missing auth session in gin context")
	}

	return sess
}

func GetUserID(c *gin.Context) uuid.UUID {
	value, ok := c.Get(ContextUserID)
	if !ok {
		return MustGetSession(c).UserID
	}

	userID, ok := value.(uuid.UUID)
	if !ok {
		return uuid.Nil
	}

	return userID
}

func GetAuthSubject(c *gin.Context) (AuthSubject, bool) {
	value, ok := c.Get(SubjectContextKey)
	if !ok {
		return AuthSubject{}, false
	}

	subject, ok := value.(AuthSubject)
	return subject, ok
}
