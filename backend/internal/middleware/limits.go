package middleware

import (
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
	var mu sync.Mutex
	buckets := map[string]rateBucket{}
	return func(c *gin.Context) {
		now := time.Now()
		key := c.ClientIP()
		mu.Lock()
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
