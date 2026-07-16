package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"xueju/backend/internal/response"
)

func BodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

type rateBucket struct {
	Count   int
	ResetAt time.Time
}

func RateLimit(limit int, window time.Duration) gin.HandlerFunc {
	return rateLimit(limit, window, func(c *gin.Context) string {
		return "ip:" + c.ClientIP()
	})
}

func RateLimitByUser(limit int, window time.Duration) gin.HandlerFunc {
	return rateLimit(limit, window, func(c *gin.Context) string {
		if value, ok := c.Get(ContextUserID); ok {
			return "user:" + fmt.Sprint(value)
		}
		return "ip:" + c.ClientIP()
	})
}

func rateLimit(limit int, window time.Duration, keyFor func(*gin.Context) string) gin.HandlerFunc {
	var mu sync.Mutex
	buckets := map[string]rateBucket{}
	return func(c *gin.Context) {
		now := time.Now()
		key := keyFor(c)
		mu.Lock()
		if len(buckets) > 1024 {
			for bucketKey, candidate := range buckets {
				if now.After(candidate.ResetAt) {
					delete(buckets, bucketKey)
				}
			}
		}
		bucket := buckets[key]
		if bucket.ResetAt.IsZero() || now.After(bucket.ResetAt) {
			bucket = rateBucket{ResetAt: now.Add(window)}
		}
		bucket.Count++
		buckets[key] = bucket
		mu.Unlock()
		if bucket.Count > limit {
			response.Error(c, http.StatusTooManyRequests, response.CodeBadRequest, "too many attempts, please try again later")
			c.Abort()
			return
		}
		c.Next()
	}
}
