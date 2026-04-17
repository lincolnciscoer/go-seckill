package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeChecker struct {
	name string
	err  error
}

func (f fakeChecker) Name() string {
	return f.name
}

func (f fakeChecker) Check(context.Context) error {
	return f.err
}

func TestHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.GET("/healthz", NewHealthHandler(
		"go-seckill",
		fakeChecker{name: "mysql"},
		fakeChecker{name: "redis"},
	))

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

	if len(response.Data.Dependencies) != 2 {
		t.Fatalf("expected 2 dependencies, got %d", len(response.Data.Dependencies))
	}

	if response.Data.Time.IsZero() {
		t.Fatal("expected time to be populated")
	}
}

func TestHealthReturnsServiceUnavailableWhenDependencyFails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.GET("/healthz", NewHealthHandler(
		"go-seckill",
		fakeChecker{name: "mysql"},
		fakeChecker{name: "redis", err: context.DeadlineExceeded},
	))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recorder := httptest.NewRecorder()

	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status code %d, got %d", http.StatusServiceUnavailable, recorder.Code)
	}

	var response HealthSuccessResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Code != "DEPENDENCY_UNAVAILABLE" {
		t.Fatalf("expected dependency unavailable code, got %q", response.Code)
	}

	if response.Data.Status != "degraded" {
		t.Fatalf("expected health status degraded, got %q", response.Data.Status)
	}
}
