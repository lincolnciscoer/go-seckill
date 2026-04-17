package response

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"go-seckill/internal/errs"
)

// Envelope 是接口层统一返回结构。
type Envelope struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func Success(c *gin.Context, data any) {
	JSON(c, http.StatusOK, errs.CodeOK, errs.DefaultMessage(errs.CodeOK), data)
}

func JSON(c *gin.Context, httpStatus int, code string, message string, data any) {
	c.JSON(httpStatus, Envelope{
		Code:    code,
		Message: message,
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
