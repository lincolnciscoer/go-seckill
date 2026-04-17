package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.GET("/healthz", Health)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recorder := httptest.NewRecorder()

	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, recorder.Code)
	}

	var response HealthResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Status != "ok" {
		t.Fatalf("expected status to be ok, got %q", response.Status)
	}

	if response.Service != "go-seckill" {
		t.Fatalf("expected service to be go-seckill, got %q", response.Service)
	}

	if response.Time.IsZero() {
		t.Fatal("expected time to be populated")
	}
}
