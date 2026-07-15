package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			buffer := make([]byte, 12)
			_, _ = rand.Read(buffer)
			requestID = hex.EncodeToString(buffer)
		}
		c.Set("requestID", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
		entry, _ := json.Marshal(map[string]interface{}{
			"type": "http_request", "requestId": requestID, "method": c.Request.Method,
			"path": c.Request.URL.Path, "status": c.Writer.Status(), "latencyMs": time.Since(started).Milliseconds(),
			"clientIp": c.ClientIP(), "client": clientKind(c.Request.UserAgent()),
		})
		log.Print(string(entry))
	}
}

func clientKind(userAgent string) string {
	value := strings.ToLower(userAgent)
	switch {
	case strings.Contains(value, "wechatdevtools"):
		return "wechat-devtools"
	case strings.Contains(value, "micromessenger"):
		return "wechat"
	case strings.Contains(value, "mozilla"):
		return "browser"
	default:
		return "other"
	}
}
