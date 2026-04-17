package handler

import (
	"time"

	"github.com/gin-gonic/gin"

	httpresponse "go-seckill/internal/transport/http/response"
)

// HealthData 表示健康检查接口里真正的业务数据部分。
type HealthData struct {
	Status  string    `json:"status"`
	Service string    `json:"service"`
	Time    time.Time `json:"time"`
}

// HealthSuccessResponse 只用于 Swagger 文档描述，方便明确展示统一响应结构。
type HealthSuccessResponse struct {
	Code    string     `json:"code"`
	Message string     `json:"message"`
	Data    HealthData `json:"data"`
}

// NewHealthHandler 返回健康检查处理器。
// 这里把 serviceName 从配置注入进来，后面你会逐渐感受到“依赖通过参数传入”比“到处写死”更容易维护。
func NewHealthHandler(serviceName string) gin.HandlerFunc {
	if serviceName == "" {
		serviceName = "go-seckill"
	}

	return func(c *gin.Context) {
		health(c, serviceName)
	}
}

// health godoc
// @Summary 健康检查
// @Description 返回当前服务的基础健康状态
// @Tags base
// @Produce json
// @Success 200 {object} handler.HealthSuccessResponse
// @Router /healthz [get]
func health(c *gin.Context, serviceName string) {
	httpresponse.Success(c, HealthData{
		Status:  "ok",
		Service: serviceName,
		Time:    time.Now().UTC(),
	})
}
