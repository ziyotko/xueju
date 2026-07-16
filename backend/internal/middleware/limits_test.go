package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimitByUserSeparatesUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set(ContextUserID, c.GetHeader("X-Test-User"))
		c.Next()
	})
	engine.GET("/limited", RateLimitByUser(1, time.Minute), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	call := func(user string) int {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/limited", nil)
		request.Header.Set("X-Test-User", user)
		engine.ServeHTTP(recorder, request)
		return recorder.Code
	}

	if status := call("1"); status != http.StatusNoContent {
		t.Fatalf("first request status=%d", status)
	}
	if status := call("1"); status != http.StatusTooManyRequests {
		t.Fatalf("repeated request status=%d", status)
	}
	if status := call("2"); status != http.StatusNoContent {
		t.Fatalf("different user must have an independent bucket, status=%d", status)
	}
}
