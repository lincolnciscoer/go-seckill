package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"go-seckill/internal/errs"
	healthcheck "go-seckill/internal/health"
	httpresponse "go-seckill/internal/transport/http/response"
)

// HealthData 表示健康检查接口里的业务数据。
type HealthData struct {
	Status       string             `json:"status"`
	Service      string             `json:"service"`
	Time         time.Time          `json:"time"`
	Dependencies []DependencyStatus `json:"dependencies,omitempty"`
}

type DependencyStatus struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// HealthSuccessResponse 主要用于 Swagger 文档展示。
type HealthSuccessResponse struct {
	Code    string     `json:"code"`
	Message string     `json:"message"`
	Data    HealthData `json:"data"`
}

// NewHealthHandler 返回健康检查处理器。
func NewHealthHandler(serviceName string, checkers ...healthcheck.Checker) gin.HandlerFunc {
	if serviceName == "" {
		serviceName = "go-seckill"
	}

	return func(c *gin.Context) {
		health(c, serviceName, checkers)
	}
}

// health godoc
// @Summary 健康检查
// @Description 返回当前服务及其关键依赖的健康状态
// @Tags base
// @Produce json
// @Success 200 {object} handler.HealthSuccessResponse
// @Failure 503 {object} handler.HealthSuccessResponse
// @Router /healthz [get]
func health(c *gin.Context, serviceName string, checkers []healthcheck.Checker) {
	healthData := HealthData{
		Status:       "ok",
		Service:      serviceName,
		Time:         time.Now().UTC(),
		Dependencies: make([]DependencyStatus, 0, len(checkers)),
	}

	if len(checkers) == 0 {
		httpresponse.Success(c, healthData)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	for _, checker := range checkers {
		if err := checker.Check(ctx); err != nil {
			healthData.Status = "degraded"
			healthData.Dependencies = append(healthData.Dependencies, DependencyStatus{
				Name:    checker.Name(),
				Status:  "down",
				Message: err.Error(),
			})
			continue
		}

		healthData.Dependencies = append(healthData.Dependencies, DependencyStatus{
			Name:   checker.Name(),
			Status: "up",
		})
	}

	if healthData.Status == "degraded" {
		httpresponse.JSON(
			c,
			http.StatusServiceUnavailable,
			errs.CodeDependencyUnavailable,
			errs.DefaultMessage(errs.CodeDependencyUnavailable),
			healthData,
		)
		return
	}

	httpresponse.Success(c, healthData)
}
