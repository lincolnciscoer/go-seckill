package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"go-seckill/internal/errs"
	jwtmanager "go-seckill/internal/security/jwt"
	httpresponse "go-seckill/internal/transport/http/response"
)

const currentUserContextKey = "current_user"

type CurrentUser struct {
	UserID   uint64
	Username string
}

func RequireAuth(manager *jwtmanager.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			httpresponse.Error(c, http.StatusUnauthorized, errs.CodeUnauthorized, "missing bearer token")
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		claims, err := manager.ParseToken(token)
		if err != nil {
			httpresponse.Error(c, http.StatusUnauthorized, errs.CodeUnauthorized, "invalid token")
			return
		}

		c.Set(currentUserContextKey, CurrentUser{
			UserID:   claims.UserID,
			Username: claims.Username,
		})
		c.Next()
	}
}

func GetCurrentUser(c *gin.Context) (CurrentUser, bool) {
	value, exists := c.Get(currentUserContextKey)
	if !exists {
		return CurrentUser{}, false
	}

	user, ok := value.(CurrentUser)
	return user, ok
}
