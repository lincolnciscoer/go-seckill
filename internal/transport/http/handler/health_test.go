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
	engine.GET("/healthz", NewHealthHandler("go-seckill"))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recorder := httptest.NewRecorder()

	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, recorder.Code)
	}

	var response HealthSuccessResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Code != "OK" {
		t.Fatalf("expected code to be OK, got %q", response.Code)
	}

	if response.Message != "success" {
		t.Fatalf("expected message to be success, got %q", response.Message)
	}

	if response.Data.Status != "ok" {
		t.Fatalf("expected status to be ok, got %q", response.Data.Status)
	}

	if response.Data.Service != "go-seckill" {
		t.Fatalf("expected service to be go-seckill, got %q", response.Data.Service)
	}

	if response.Data.Time.IsZero() {
		t.Fatal("expected time to be populated")
	}
}
