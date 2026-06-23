package middleware

import (
	"context"
	"net/http"

	"github.com/cuffeyvidzro/leamout/internal/modules/session"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	SessionCookieName = "session_id"
	SessionContextKey = "auth.session"
	ContextUserID     = "userID"
)

type SessionService interface {
	GetByToken(ctx context.Context, token string) (*session.Session, error)
}

func AuthMiddleware(sessionService SessionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie(SessionCookieName)
		if err != nil || cookie == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing authentication cookie",
			})
			return
		}

		sess, err := sessionService.GetByToken(c.Request.Context(), cookie)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired session",
			})
			return
		}

		c.Set(SessionContextKey, sess)
		c.Set(ContextUserID, sess.UserID)
		c.Next()
	}
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
	return MustGetSession(c).UserID
}
