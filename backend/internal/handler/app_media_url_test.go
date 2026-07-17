package handler

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMediaURLForRequestUpgradesSameHostToForwardedHTTPS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	request := httptest.NewRequest("GET", "http://www.xueju.xyz/api/users/me", nil)
	request.Host = "www.xueju.xyz"
	request.Header.Set("X-Forwarded-Proto", "https")
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = request

	got := mediaURLForRequest(context, "http://www.xueju.xyz/uploads/avatars/avatar.jpeg")
	want := "https://www.xueju.xyz/uploads/avatars/avatar.jpeg"
	if got != want {
		t.Fatalf("mediaURLForRequest() = %q, want %q", got, want)
	}
}

func TestMediaURLForRequestPreservesExternalHost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	request := httptest.NewRequest("GET", "https://www.xueju.xyz/api/users/me", nil)
	request.Host = "www.xueju.xyz"
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = request

	want := "http://cdn.example.com/uploads/avatars/avatar.jpeg"
	if got := mediaURLForRequest(context, want); got != want {
		t.Fatalf("mediaURLForRequest() = %q, want %q", got, want)
	}
}
