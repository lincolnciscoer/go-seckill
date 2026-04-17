package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"go-seckill/internal/errs"
	httpresponse "go-seckill/internal/transport/http/response"
)

// Recovery 捕获未处理的 panic，避免单个请求把整个服务带崩。
// 这是 Web 服务最基础但也最重要的一层保护。
func Recovery(logger *zap.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = zap.NewNop()
	}

	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		logger.Error("panic recovered",
			zap.Any("panic", recovered),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
		)

		httpresponse.Error(c, http.StatusInternalServerError, errs.CodeInternalError, "")
	})
}
