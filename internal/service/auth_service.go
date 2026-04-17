package service

import (
	"context"
	"errors"
	"strings"

	"go-seckill/internal/model"
	"go-seckill/internal/repository"
	"go-seckill/internal/security/jwt"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
)

type AuthService struct {
	users      repository.UserRepository
	jwtManager *jwt.Manager
}

type AuthResult struct {
	User       *model.User
	Token      string
	ExpiresAt  int64
	TokenType  string
}

func NewAuthService(users repository.UserRepository, jwtManager *jwt.Manager) *AuthService {
	return &AuthService{
		users:      users,
		jwtManager: jwtManager,
	}
}

func (s *AuthService) Register(ctx context.Context, username string, password string) (*AuthResult, error) {
	username = strings.TrimSpace(username)

	existingUser, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, ErrUserAlreadyExists
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Username:     username,
		PasswordHash: string(passwordHash),
	}

	if err := s.users.Create(ctx, user); err != nil {
		var mysqlErr *mysqlDriver.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return nil, ErrUserAlreadyExists
		}

		return nil, err
	}

	return s.buildAuthResult(user)
}

func (s *AuthService) Login(ctx context.Context, username string, password string) (*AuthResult, error) {
	username = strings.TrimSpace(username)

	user, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return s.buildAuthResult(user)
}

func (s *AuthService) GetUserByID(ctx context.Context, userID uint64) (*model.User, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	return user, nil
}

func (s *AuthService) buildAuthResult(user *model.User) (*AuthResult, error) {
	token, expiresAt, err := s.jwtManager.GenerateToken(user)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		User:      user,
		Token:     token,
		ExpiresAt: expiresAt.Unix(),
		TokenType: "Bearer",
	}, nil
}
