package router

import (
	"github.com/gin-gonic/gin"

	"go-seckill/internal/transport/http/handler"
)

// NewEngine 负责集中管理 HTTP 路由注册。
// 项目后面功能变多之后，路由会按模块继续拆分，但统一入口最好从一开始就固定下来。
func NewEngine() *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery())

	registerBaseRoutes(engine)

	return engine
}

func registerBaseRoutes(engine *gin.Engine) {
	engine.GET("/healthz", handler.Health)
}
