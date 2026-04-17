package router

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	"go-seckill/internal/config"
	"go-seckill/internal/health"
	jwtmanager "go-seckill/internal/security/jwt"
	"go-seckill/internal/service"
	"go-seckill/internal/transport/http/handler"
	"go-seckill/internal/transport/http/middleware"
)

type Dependencies struct {
	Config         *config.Config
	Logger         *zap.Logger
	HealthCheckers []health.Checker
	AuthService    *service.AuthService
	JWTManager     *jwtmanager.Manager
}

// NewEngine 负责集中管理 HTTP 路由注册。
// 项目后面功能变多之后，路由会按模块继续拆分，但统一入口最好从一开始就固定下来。
func NewEngine(dep Dependencies) *gin.Engine {
	engine := gin.New()
	engine.Use(middleware.AccessLogger(dep.Logger), middleware.Recovery(dep.Logger))

	registerBaseRoutes(engine, dep)
	registerDocsRoutes(engine)

	return engine
}

func registerBaseRoutes(engine *gin.Engine, dep Dependencies) {
	serviceName := "go-seckill"
	if dep.Config != nil && dep.Config.App.Name != "" {
		serviceName = dep.Config.App.Name
	}

	engine.GET("/healthz", handler.NewHealthHandler(serviceName, dep.HealthCheckers...))

	if dep.AuthService != nil && dep.JWTManager != nil {
		authHandler := handler.NewAuthHandler(dep.AuthService)
		apiV1 := engine.Group("/api/v1")
		apiV1.POST("/auth/register", authHandler.Register)
		apiV1.POST("/auth/login", authHandler.Login)
		apiV1.GET("/me", middleware.RequireAuth(dep.JWTManager), authHandler.Me)
	}
}

func registerDocsRoutes(engine *gin.Engine) {
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
