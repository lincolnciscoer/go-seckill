package jwt

import (
	"testing"
	"time"

	"go-seckill/internal/config"
	"go-seckill/internal/model"
)

func TestManagerGenerateAndParseToken(t *testing.T) {
	manager := NewManager(config.JWTConfig{
		Secret:    "test-secret",
		Issuer:    "go-seckill-test",
		AccessTTL: time.Hour,
	})

	token, expiresAt, err := manager.GenerateToken(&model.User{
		ID:       42,
		Username: "tester",
	})
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	if token == "" {
		t.Fatal("expected token to be non-empty")
	}

	if expiresAt.IsZero() {
		t.Fatal("expected expiresAt to be set")
	}

	claims, err := manager.ParseToken(token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}

	if claims.UserID != 42 {
		t.Fatalf("expected user id 42, got %d", claims.UserID)
	}

	if claims.Username != "tester" {
		t.Fatalf("expected username tester, got %q", claims.Username)
	}
}
