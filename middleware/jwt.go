package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/rachitnimje/trackle-web/utils"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get token from cookie first
		token, err := c.Cookie("auth_token")
		if err != nil {
			// If no cookie, try Authorization header
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				appErr := utils.NewAuthenticationError("Authorization token required", nil)
				utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
				c.Abort()
				return
			}
			
			// Extract token from "Bearer <token>"
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				token = authHeader[7:]
			} else {
				appErr := utils.NewAuthenticationError("Invalid authorization header format", nil)
				utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
				c.Abort()
				return
			}
		}

		// Validate the token
		claims, err := utils.ValidateJWT(token)
		if err != nil {
			appErr := utils.NewAuthenticationError("Invalid or expired token", err)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			c.Abort()
			return
		}

		// Set user ID in context for use in handlers
		c.Set("user_id", claims.UserID)
		c.Next()
	}
}