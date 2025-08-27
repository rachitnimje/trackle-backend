package middleware

import (
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CORSMiddleware() gin.HandlerFunc {
	var allowedOrigins []string
	corsOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if corsOrigins != "" {
		allowedOrigins = strings.Split(corsOrigins, ",")
		for i := range allowedOrigins {
			allowedOrigins[i] = strings.TrimSpace(allowedOrigins[i])
		}
	}

	config := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{
			"Origin", 
			"Authorization", 
			"Content-Type", 
			"Accept", 
			"X-Requested-With",
			"Cache-Control",
			"X-Request-ID",
			"X-Client-Version",
		},
		ExposeHeaders:    []string{"Content-Length", "Set-Cookie"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	if len(allowedOrigins) > 0 {
		config.AllowOrigins = allowedOrigins
	} else {
		panic("CORS_ALLOWED_ORIGINS environment variable is not set or empty. Refusing to start with no allowed origins.")
	}

	return cors.New(config)
}