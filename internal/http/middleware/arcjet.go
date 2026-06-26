package middleware

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/arcjet/arcjet-go"
	"github.com/gin-gonic/gin"
)

// Arcjet now accepts log *slog.Logger as an argument.
func Arcjet(client *arcjet.Client, log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if client == nil {
			c.Next()
			return
		}

		if isExcludedPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		start := time.Now()

		decision, err := client.Protect(
			c.Request.Context(),
			c.Request,
			arcjet.WithRequested(1),
		)

		log.InfoContext(
			c.Request.Context(),
			"arcjet protection check",
			"latency_ms", time.Since(start).Milliseconds(),
			"path", c.FullPath(),
			"method", c.Request.Method,
		)

		if err != nil {
			log.ErrorContext(
				c.Request.Context(),
				"arcjet client error",
				"error", err,
				"path", c.FullPath(),
				"method", c.Request.Method,
			)

			// Fail open:
			// Arcjet/runtime failure should not take Leamout offline.
			c.Next()
			return
		}

		if decision.IsDenied() {
			if decision.IP.IsVPN || decision.IP.IsProxy || decision.IP.IsTor {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error": "VPN/Proxy not allowed",
				})
				return
			}

			if decision.Reason.IsRateLimit() {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"error": "Rate limit exceeded",
				})
				return
			}

			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Request denied",
			})
			return
		}

		c.Next()
	}
}

func isExcludedPath(path string) bool {
	switch {
	case path == "/health", path == "/favicon.ico":
		return true

	// Public dunning recovery links are protected by high-entropy tokens.
	// Do not let bot/security middleware block customers opening SMS links.
	case strings.HasPrefix(path, "/v1/dunning/"):
		return true

	default:
		return false
	}
}
