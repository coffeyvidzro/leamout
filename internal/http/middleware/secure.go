package middleware

import (
	"github.com/gin-contrib/secure"
	"github.com/gin-gonic/gin"
)

func Secure(development bool) gin.HandlerFunc {
	config := secure.DefaultConfig()

	// Usually false if HTTPS is handled by a reverse proxy like Caddy, Nginx,
	// Traefik, Render, Fly.io, Railway, etc.
	config.SSLRedirect = false

	config.ReferrerPolicy = "no-referrer"
	config.ContentSecurityPolicy = "default-src 'self'"
	config.IENoOpen = true

	if !development {
		config.STSSeconds = 31536000
	}

	return secure.New(config)
}
