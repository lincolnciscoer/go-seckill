package router

import (
	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
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
	Config          *config.Config
	Logger          *zap.Logger
	HealthCheckers  []health.Checker
	AuthService     *service.AuthService
	OrderService    *service.OrderService
	ProductService  *service.ProductService
	ActivityService *service.ActivityService
	SeckillService  *service.SeckillService
	RedisClient     *goredis.Client
	JWTManager      *jwtmanager.Manager
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

		if dep.ProductService != nil {
			productHandler := handler.NewProductHandler(dep.ProductService)
			apiV1.POST("/products", middleware.RequireAuth(dep.JWTManager), productHandler.Create)
			apiV1.GET("/products", productHandler.List)
		}

		if dep.ActivityService != nil {
			activityHandler := handler.NewActivityHandler(dep.ActivityService)
			apiV1.POST("/activities", middleware.RequireAuth(dep.JWTManager), activityHandler.Create)
			apiV1.POST("/activities/:id/preheat", middleware.RequireAuth(dep.JWTManager), activityHandler.Preheat)
			apiV1.GET("/activities", activityHandler.List)
			apiV1.GET("/activities/:id", activityHandler.Detail)
		}

		if dep.OrderService != nil {
			orderHandler := handler.NewOrderHandler(dep.OrderService)
			apiV1.GET("/orders/me", middleware.RequireAuth(dep.JWTManager), orderHandler.ListMine)
			apiV1.GET("/orders/:orderNo", middleware.RequireAuth(dep.JWTManager), orderHandler.Detail)
		}

		if dep.SeckillService != nil {
			seckillHandler := handler.NewSeckillHandler(dep.SeckillService)
			seckillRoute := apiV1.Group("/seckill/activities/:id")
			seckillRoute.Use(middleware.RequireAuth(dep.JWTManager))
			if dep.RedisClient != nil {
				seckillRoute.Use(middleware.NewSeckillGuard(dep.RedisClient, dep.Logger).Middleware())
			}
			seckillRoute.POST("/attempt", seckillHandler.Attempt)
		}
	}
}

func registerDocsRoutes(engine *gin.Engine) {
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
