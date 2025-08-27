package middleware

import (
	"fmt"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/rachitnimje/trackle-web/utils"
)

// ErrorRecoveryMiddleware catches any panics in the request handling chain
// and converts them to appropriate error responses
func ErrorRecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log the stack trace
				stack := debug.Stack()
				fmt.Printf("[PANIC RECOVERED] %v\n%s\n", err, stack)

				// Create a clean error for the client
				appErr := utils.NewInternalError("An unexpected error occurred", fmt.Errorf("%v", err))
				utils.ErrorResponse(c, appErr.StatusCode, "Server Error", appErr)
				
				// Abort the request chain
				c.Abort()
			}
		}()
		
		c.Next()
	}
}
