package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthResponse 表示健康检查接口的返回值。
// 这里刻意保持字段简单，方便你先把“Gin 路由 -> Handler -> JSON 返回”这条最小链路理解清楚。
type HealthResponse struct {
	Status  string    `json:"status"`
	Service string    `json:"service"`
	Time    time.Time `json:"time"`
}

// Health 用于返回当前服务的基础可用性状态。
// 后续接入数据库、Redis、MQ 之后，这里还会逐步扩展出更细的依赖健康检查信息。
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status:  "ok",
		Service: "go-seckill",
		Time:    time.Now().UTC(),
	})
}
