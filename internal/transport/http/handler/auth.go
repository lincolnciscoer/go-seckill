package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"go-seckill/internal/errs"
	"go-seckill/internal/model"
	"go-seckill/internal/service"
	httpmiddleware "go-seckill/internal/transport/http/middleware"
	httpresponse "go-seckill/internal/transport/http/response"
)

type AuthHandler struct {
	authService *service.AuthService
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Password string `json:"password" binding:"required,min=6,max=72"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Password string `json:"password" binding:"required,min=6,max=72"`
}

type UserProfile struct {
	ID        uint64    `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

type AuthResponse struct {
	TokenType string      `json:"token_type"`
	Token     string      `json:"token"`
	ExpiresAt int64       `json:"expires_at"`
	User      UserProfile `json:"user"`
}

type MeResponse struct {
	User UserProfile `json:"user"`
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Register godoc
// @Summary 用户注册
// @Tags auth
// @Accept json
// @Produce json
// @Param request body handler.RegisterRequest true "注册请求"
// @Success 200 {object} response.Envelope
// @Router /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var request RegisterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httpresponse.Error(c, http.StatusBadRequest, errs.CodeBadRequest, err.Error())
		return
	}

	result, err := h.authService.Register(c.Request.Context(), request.Username, request.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserAlreadyExists):
			httpresponse.Error(c, http.StatusBadRequest, errs.CodeUserAlreadyExists, "")
		default:
			httpresponse.Error(c, http.StatusInternalServerError, errs.CodeInternalError, "")
		}
		return
	}

	httpresponse.Success(c, buildAuthResponse(result))
}

// Login godoc
// @Summary 用户登录
// @Tags auth
// @Accept json
// @Produce json
// @Param request body handler.LoginRequest true "登录请求"
// @Success 200 {object} response.Envelope
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var request LoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httpresponse.Error(c, http.StatusBadRequest, errs.CodeBadRequest, err.Error())
		return
	}

	result, err := h.authService.Login(c.Request.Context(), request.Username, request.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			httpresponse.Error(c, http.StatusUnauthorized, errs.CodeInvalidCredentials, "")
		default:
			httpresponse.Error(c, http.StatusInternalServerError, errs.CodeInternalError, "")
		}
		return
	}

	httpresponse.Success(c, buildAuthResponse(result))
}

// Me godoc
// @Summary 当前登录用户信息
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Envelope
// @Router /api/v1/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	currentUser, ok := httpmiddleware.GetCurrentUser(c)
	if !ok {
		httpresponse.Error(c, http.StatusUnauthorized, errs.CodeUnauthorized, "")
		return
	}

	user, err := h.authService.GetUserByID(c.Request.Context(), currentUser.UserID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			httpresponse.Error(c, http.StatusUnauthorized, errs.CodeUnauthorized, "user not found")
		default:
			httpresponse.Error(c, http.StatusInternalServerError, errs.CodeInternalError, "")
		}
		return
	}

	httpresponse.Success(c, MeResponse{
		User: buildUserProfile(user),
	})
}

func buildAuthResponse(result *service.AuthResult) AuthResponse {
	return AuthResponse{
		TokenType: result.TokenType,
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt,
		User:      buildUserProfile(result.User),
	}
}

func buildUserProfile(user *model.User) UserProfile {
	return UserProfile{
		ID:        user.ID,
		Username:  user.Username,
		CreatedAt: user.CreatedAt,
	}
}
