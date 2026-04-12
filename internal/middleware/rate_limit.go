package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimiter struct {
	requests map[string][]time.Time
	mu       sync.Mutex
	limit    int
	window   time.Duration
}

func newRateLimiter(requestsPerMinute int) *rateLimiter {
	return &rateLimiter{
		requests: make(map[string][]time.Time),
		limit:    requestsPerMinute,
		window:   time.Minute,
	}
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Clean old requests
	if times, exists := rl.requests[key]; exists {
		var valid []time.Time
		for _, t := range times {
			if t.After(cutoff) {
				valid = append(valid, t)
			}
		}
		rl.requests[key] = valid
	}

	// Check limit
	if len(rl.requests[key]) >= rl.limit {
		return false
	}

	// Add new request
	rl.requests[key] = append(rl.requests[key], now)
	return true
}

func RateLimitMiddleware(requestsPerMinute int) gin.HandlerFunc {
	limiter := newRateLimiter(requestsPerMinute)

	return func(c *gin.Context) {
		key := c.ClientIP()
		if !limiter.allow(key) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}
		c.Next()
	}
}
