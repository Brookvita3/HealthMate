package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimitMiddleware enforces rate limits using Redis fixed window algorithm
func RateLimitMiddleware(rdb *redis.Client, limitStr, windowStr string) gin.HandlerFunc {
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 100 // default
	}
	window, err := strconv.Atoi(windowStr)
	if err != nil {
		window = 60 // default
	}

	return func(c *gin.Context) {
		// 1. Identify client (User ID > API Key > IP)
		var key string
		if sub, exists := c.Get("sub"); exists {
			key = "ratelimit:user:" + sub.(string)
		} else if apiKey := c.GetHeader("X-API-Key"); apiKey != "" {
			key = "ratelimit:apikey:" + apiKey
		} else {
			key = "ratelimit:ip:" + c.ClientIP()
		}

		ctx := c.Request.Context()

		// 2. Fixed Window algorithm
		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			// Fail open: if Redis is down, allow request but log could be added here
			c.Next()
			return
		}

		if count == 1 {
			rdb.Expire(ctx, key, time.Duration(window)*time.Second)
		}

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		remaining := limit - int(count)
		if remaining < 0 {
			remaining = 0
		}
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))

		if count > int64(limit) {
			// Get TTL to calculate Retry-After
			ttl, _ := rdb.TTL(ctx, key).Result()
			retryAfter := int(ttl.Seconds())
			if retryAfter < 0 {
				retryAfter = window // fallback
			}

			c.Header("Retry-After", strconv.Itoa(retryAfter))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "too many requests",
				"retry_after": retryAfter,
			})
			return
		}

		c.Next()
	}
}
