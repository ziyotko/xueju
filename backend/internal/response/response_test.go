package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestErrorHidesInternalMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Set("requestID", "test-request")

	Error(context, http.StatusInternalServerError, CodeServerError, "sql: secret connection detail")

	var body Body
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Message != "服务暂时不可用，请稍后重试" {
		t.Fatalf("unexpected public message: %q", body.Message)
	}
}

func TestErrorKeepsClientMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)

	Error(context, http.StatusBadRequest, CodeBadRequest, "参数错误")

	var body Body
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Message != "参数错误" {
		t.Fatalf("unexpected public message: %q", body.Message)
	}
}
