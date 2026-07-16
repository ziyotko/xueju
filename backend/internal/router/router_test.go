package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"xueju/backend/internal/config"
)

func TestHealthWithoutDatabase(t *testing.T) {
	engine := New(config.Config{AppName: "xueju-api", AppEnv: "test", JWTSecret: "test"}, nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/health", nil)

	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	var body responseBody
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data := body.Data.(map[string]interface{})
	if data["database"] != "not_configured" {
		t.Fatalf("expected not_configured database, got %v", data["database"])
	}
}

func TestBusinessEndpointRequiresDatabase(t *testing.T) {
	engine := New(config.Config{AppName: "xueju-api", AppEnv: "test", JWTSecret: "test"}, nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/events", nil)

	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}

func TestPublicTagsDoNotRequireDatabase(t *testing.T) {
	engine := New(config.Config{AppName: "xueju-api", AppEnv: "test", JWTSecret: "test"}, nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/dict/tags", nil)

	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func TestModerationEndpointsRequireAdminAuthentication(t *testing.T) {
	engine := New(config.Config{AppName: "xueju-api", AppEnv: "test", JWTSecret: "test", UploadDir: t.TempDir()}, nil)
	paths := []string{"/api/admin/moderation/summary", "/api/admin/uploads/1/preview"}
	for _, path := range paths {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path, nil)
		engine.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("%s expected 401, got %d", path, recorder.Code)
		}
	}
}

type responseBody struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}
