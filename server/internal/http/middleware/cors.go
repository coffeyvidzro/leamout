package middleware

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CORS(allowOrigins []string, development bool) gin.HandlerFunc {
	config := cors.Config{
		AllowOrigins:     allowOrigins,
		AllowCredentials: true,
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"Content-Length",
			"Cache-Control",
			"Origin",
			"X-CSRF-Token",
			"X-Requested-With",
			"X-Request-ID",
			"X-Correlation-ID",
		},
		ExposeHeaders: []string{
			"X-Request-ID",
			"X-Correlation-ID",
		},
		MaxAge: 12 * time.Hour,
	}

	if development && len(allowOrigins) == 0 {
		config.AllowOriginFunc = func(origin string) bool {
			return origin != ""
		}
	}

	return cors.New(config)
}
