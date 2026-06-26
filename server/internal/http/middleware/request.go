package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	ContextRequestID     = "request_id"
	ContextCorrelationID = "correlation_id"
)

func RequestContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
		}

		correlationID := c.GetHeader("X-Correlation-ID")
		if correlationID == "" {
			correlationID = requestID
		}

		c.Set(ContextRequestID, requestID)
		c.Set(ContextCorrelationID, correlationID)

		c.Header("X-Request-ID", requestID)
		c.Header("X-Correlation-ID", correlationID)

		c.Next()
	}
}

func RequestLogger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		path := c.Request.URL.Path

		c.Next()

		status := c.Writer.Status()
		latency := time.Since(startedAt)

		attrs := []slog.Attr{
			slog.String("request_id", c.GetString(ContextRequestID)),
			slog.String("correlation_id", c.GetString(ContextCorrelationID)),
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.Int("status", status),
			slog.Duration("latency", latency),
			slog.String("client_ip", c.ClientIP()),
			slog.String("remote_addr", c.Request.RemoteAddr),
			slog.String("user_agent", c.GetHeader("User-Agent")),
		}

		if geo, ok := GetGeolocation(c); ok {
			attrs = append(attrs,
				slog.String("geo_ip", geo.IP),
				slog.String("geo_country_code", geo.CountryCode),
				slog.String("geo_country_name", geo.CountryName),
				slog.String("geo_city", geo.City),
				slog.String("geo_timezone", geo.TimeZone),
				slog.String("geo_source", geo.Source),
			)
		}

		if len(c.Errors) > 0 {
			attrs = append(attrs, slog.String("errors", c.Errors.String()))
		}

		level := slog.LevelInfo
		switch {
		case status >= http.StatusInternalServerError:
			level = slog.LevelError
		case status >= http.StatusBadRequest:
			level = slog.LevelWarn
		}

		log.LogAttrs(c.Request.Context(), level, "http request completed", attrs...)
	}
}
