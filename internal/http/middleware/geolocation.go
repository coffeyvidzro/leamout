package middleware

import (
	"log/slog"

	"github.com/cuffeyvidzro/leamout/internal/platform/geoip"
	"github.com/gin-gonic/gin"
)

const ContextGeolocation = "geolocation"

func Geolocation(locator *geoip.Geolocator, log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if locator == nil {
			c.Next()
			return
		}

		ip := c.ClientIP()

		info, err := locator.Lookup(c.Request.Context(), ip)
		if err == nil && info != nil {
			c.Set(ContextGeolocation, info)
		}

		c.Next()
	}
}

func GetGeolocation(c *gin.Context) (*geoip.GeoInfo, bool) {
	value, exists := c.Get(ContextGeolocation)
	if !exists {
		return nil, false
	}

	info, ok := value.(*geoip.GeoInfo)
	return info, ok
}
