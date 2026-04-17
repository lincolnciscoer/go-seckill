package response

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"go-seckill/internal/errs"
)

// Envelope 是接口层统一返回结构。
// 从这一版开始，所有 HTTP API 都尽量通过同一种外层格式返回，方便前端、测试和日志排查保持一致。
type Envelope struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Envelope{
		Code:    errs.CodeOK,
		Message: errs.DefaultMessage(errs.CodeOK),
		Data:    data,
	})
}

func Error(c *gin.Context, httpStatus int, code string, message string) {
	if message == "" {
		message = errs.DefaultMessage(code)
	}

	c.AbortWithStatusJSON(httpStatus, Envelope{
		Code:    code,
		Message: message,
	})
}
