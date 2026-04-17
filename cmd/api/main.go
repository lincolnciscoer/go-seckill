package main

import (
	"errors"
	"net/http"
	"os"

	_ "go-seckill/docs/swagger"
	"go-seckill/internal/bootstrap"
	"go-seckill/internal/cache"
	"go-seckill/internal/config"
	"go-seckill/internal/mq/rocketmq"
	"go-seckill/internal/repository"
	jwtmanager "go-seckill/internal/security/jwt"
	"go-seckill/internal/service"
	"go-seckill/internal/transport/http/router"

	"go.uber.org/zap"
)

// @title go-seckill API
// @version 0.1.0
// @description A step-by-step Go seckill backend project
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	cfg, err := bootstrap.LoadConfig(bootstrap.DefaultConfigPath())
	if err != nil {
		_, _ = os.Stderr.WriteString("failed to load config: " + err.Error() + "\n")
		os.Exit(1)
	}

	appLogger, err := bootstrap.InitLogger(cfg)
	if err != nil {
		_, _ = os.Stderr.WriteString("failed to init logger: " + err.Error() + "\n")
		os.Exit(1)
	}
	defer func() {
		_ = appLogger.Sync()
	}()

	infra, err := bootstrap.InitInfrastructure(cfg, appLogger)
	if err != nil {
		appLogger.Fatal("failed to initialize infrastructure", zap.Error(err))
	}
	defer infra.Close()

	orderProducer, err := rocketmq.NewProducer(cfg.RocketMQ)
	if err != nil {
		appLogger.Fatal("failed to initialize rocketmq producer", zap.Error(err))
	}
	defer func() {
		_ = orderProducer.Shutdown()
	}()

	engine := buildRouter(cfg, appLogger, infra, orderProducer)
	addr := bootstrap.ServerAddress(cfg)
	server := bootstrap.NewHTTPServer(addr, cfg, engine)

	appLogger.Info("starting api server",
		zap.String("addr", addr),
		zap.String("app_name", cfg.App.Name),
		zap.String("app_env", cfg.App.Env),
	)

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			appLogger.Fatal("http server stopped unexpectedly", zap.Error(err))
		}
	}()

	if err := bootstrap.WaitForShutdown(server, cfg, appLogger); err != nil {
		appLogger.Fatal("graceful shutdown failed", zap.Error(err))
	}
}

func buildRouter(
	cfg *config.Config,
	appLogger *zap.Logger,
	infra *bootstrap.Infrastructure,
	orderProducer *rocketmq.Producer,
) http.Handler {
	userRepository := repository.NewGormUserRepository(infra.GormDB)
	productRepository := repository.NewGormProductRepository(infra.GormDB)
	activityRepository := repository.NewGormActivityRepository(infra.GormDB)
	orderRepository := repository.NewSQLOrderRepository(infra.SQLDB)
	activityCache := cache.NewActivityCache(infra.Redis)
	orderStatusCache := cache.NewOrderStatusCache(infra.Redis)
	jwtManager := jwtmanager.NewManager(cfg.JWT)
	authService := service.NewAuthService(userRepository, jwtManager)
	productService := service.NewProductService(productRepository)
	activityService := service.NewActivityService(productRepository, activityRepository, activityCache)
	orderService := service.NewOrderService(orderRepository, orderStatusCache)
	seckillService := service.NewSeckillService(productRepository, activityRepository, orderRepository, activityCache, orderStatusCache, orderProducer)

	return router.NewEngine(router.Dependencies{
		Config:          cfg,
		Logger:          appLogger,
		HealthCheckers:  infra.HealthCheckers,
		AuthService:     authService,
		OrderService:    orderService,
		ProductService:  productService,
		ActivityService: activityService,
		SeckillService:  seckillService,
		RedisClient:     infra.Redis,
		JWTManager:      jwtManager,
	})
}
