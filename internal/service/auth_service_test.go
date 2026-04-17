package service

import (
	"context"
	"testing"
	"time"

	"go-seckill/internal/config"
	"go-seckill/internal/model"
	"go-seckill/internal/security/jwt"
)

type fakeUserRepository struct {
	usersByID       map[uint64]*model.User
	usersByUsername map[string]*model.User
	nextID          uint64
}

func newFakeUserRepository() *fakeUserRepository {
	return &fakeUserRepository{
		usersByID:       make(map[uint64]*model.User),
		usersByUsername: make(map[string]*model.User),
		nextID:          1,
	}
}

func (r *fakeUserRepository) Create(_ context.Context, user *model.User) error {
	user.ID = r.nextID
	r.nextID++
	cloned := *user
	r.usersByID[user.ID] = &cloned
	r.usersByUsername[user.Username] = &cloned
	return nil
}

func (r *fakeUserRepository) GetByUsername(_ context.Context, username string) (*model.User, error) {
	user, ok := r.usersByUsername[username]
	if !ok {
		return nil, nil
	}

	cloned := *user
	return &cloned, nil
}

func (r *fakeUserRepository) GetByID(_ context.Context, id uint64) (*model.User, error) {
	user, ok := r.usersByID[id]
	if !ok {
		return nil, nil
	}

	cloned := *user
	return &cloned, nil
}

func TestAuthServiceRegisterAndLogin(t *testing.T) {
	repo := newFakeUserRepository()
	manager := jwt.NewManager(config.JWTConfig{
		Secret:    "test-secret",
		Issuer:    "go-seckill-test",
		AccessTTL: time.Hour,
	})
	service := NewAuthService(repo, manager)

	registerResult, err := service.Register(context.Background(), "alice", "password123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if registerResult.User.ID == 0 {
		t.Fatal("expected generated user id")
	}

	if registerResult.Token == "" {
		t.Fatal("expected register token")
	}

	_, err = service.Register(context.Background(), "alice", "password123")
	if err != ErrUserAlreadyExists {
		t.Fatalf("expected ErrUserAlreadyExists, got %v", err)
	}

	loginResult, err := service.Login(context.Background(), "alice", "password123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	if loginResult.Token == "" {
		t.Fatal("expected login token")
	}

	_, err = service.Login(context.Background(), "alice", "wrong-password")
	if err != ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}
