package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimiter struct {
	requests map[string][]time.Time
	mutex    sync.RWMutex
	limit    int
	window   time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (rl *rateLimiter) allow(clientIP string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Clean old entries
	if requests, exists := rl.requests[clientIP]; exists {
		var validRequests []time.Time
		for _, reqTime := range requests {
			if reqTime.After(cutoff) {
				validRequests = append(validRequests, reqTime)
			}
		}
		rl.requests[clientIP] = validRequests
	}

	// Check if limit exceeded
	if len(rl.requests[clientIP]) >= rl.limit {
		return false
	}

	// Add current request
	rl.requests[clientIP] = append(rl.requests[clientIP], now)
	return true
}

// RateLimitMiddleware creates a rate limiting middleware
// Default: 1000 requests per minute per IP
func RateLimitMiddleware() gin.HandlerFunc {
	limiter := newRateLimiter(1000, time.Minute)

	return gin.HandlerFunc(func(c *gin.Context) {
		clientIP := c.ClientIP()
		
		if !limiter.allow(clientIP) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"message": "Rate limit exceeded. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	})
}

// StrictRateLimitMiddleware for sensitive endpoints like auth
// 10 requests per minute per IP
func StrictRateLimitMiddleware() gin.HandlerFunc {
	limiter := newRateLimiter(10, time.Minute)

	return gin.HandlerFunc(func(c *gin.Context) {
		clientIP := c.ClientIP()
		
		if !limiter.allow(clientIP) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"message": "Too many authentication attempts. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	})
}
