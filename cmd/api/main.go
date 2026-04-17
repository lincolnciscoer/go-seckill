package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "go-seckill/docs/swagger"
	"go-seckill/internal/cache"
	"go-seckill/internal/config"
	"go-seckill/internal/health"
	"go-seckill/internal/repository"
	jwtmanager "go-seckill/internal/security/jwt"
	"go-seckill/internal/service"
	mysqlstore "go-seckill/internal/store/mysql"
	redisstore "go-seckill/internal/store/redis"
	"go-seckill/internal/transport/http/router"
	"go-seckill/pkg/logger"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// @title go-seckill API
// @version 0.1.0
// @description A step-by-step Go seckill backend project
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	cfg, err := config.Load(defaultConfigPath())
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	appLogger, err := logger.New(cfg.Log)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = appLogger.Sync()
	}()

	resources, err := initInfrastructure(cfg, appLogger)
	if err != nil {
		appLogger.Fatal("failed to initialize infrastructure", zap.Error(err))
	}
	defer resources.Close()

	userRepository := repository.NewGormUserRepository(resources.GormDB)
	productRepository := repository.NewGormProductRepository(resources.GormDB)
	activityRepository := repository.NewGormActivityRepository(resources.GormDB)
	orderRepository := repository.NewSQLOrderRepository(resources.SQLDB)
	activityCache := cache.NewActivityCache(resources.Redis)
	jwtManager := jwtmanager.NewManager(cfg.JWT)
	authService := service.NewAuthService(userRepository, jwtManager)
	productService := service.NewProductService(productRepository)
	activityService := service.NewActivityService(productRepository, activityRepository, activityCache)
	orderService := service.NewOrderService(orderRepository)
	seckillService := service.NewSeckillService(productRepository, activityRepository, orderRepository, activityCache)

	engine := router.NewEngine(router.Dependencies{
		Config:          cfg,
		Logger:          appLogger,
		HealthCheckers:  resources.HealthCheckers,
		AuthService:     authService,
		OrderService:    orderService,
		ProductService:  productService,
		ActivityService: activityService,
		SeckillService:  seckillService,
		JWTManager:      jwtManager,
	})

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	server := newHTTPServer(addr, cfg, engine)

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

	if err := waitForShutdown(server, cfg, appLogger); err != nil {
		appLogger.Fatal("graceful shutdown failed", zap.Error(err))
	}
}

func defaultConfigPath() string {
	if path := os.Getenv("GO_SECKILL_CONFIG"); path != "" {
		return path
	}

	return "configs/config.example.yaml"
}

func waitForShutdown(server *http.Server, cfg *config.Config, logger *zap.Logger) error {
	stopSignal := make(chan os.Signal, 1)
	signal.Notify(stopSignal, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(stopSignal)

	sig := <-stopSignal
	logger.Info("received shutdown signal", zap.String("signal", sig.String()))

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	return server.Shutdown(ctx)
}

func newHTTPServer(addr string, cfg *config.Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}
}

type infrastructure struct {
	GormDB         *gorm.DB
	SQLDB          *sql.DB
	Redis          *goredis.Client
	HealthCheckers []health.Checker
}

func initInfrastructure(cfg *config.Config, logger *zap.Logger) (*infrastructure, error) {
	gormDB, sqlDB, err := mysqlstore.New(cfg.MySQL)
	if err != nil {
		return nil, err
	}

	logger.Info("mysql connected",
		zap.String("host", cfg.MySQL.Host),
		zap.Int("port", cfg.MySQL.Port),
		zap.String("database", cfg.MySQL.Database),
	)

	redisClient, err := redisstore.New(cfg.Redis)
	if err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	logger.Info("redis connected",
		zap.String("addr", cfg.Redis.Addr),
		zap.Int("db", cfg.Redis.DB),
	)

	return &infrastructure{
		GormDB: gormDB,
		SQLDB:  sqlDB,
		Redis:  redisClient,
		HealthCheckers: []health.Checker{
			mysqlstore.NewHealthChecker(sqlDB),
			redisstore.NewHealthChecker(redisClient),
		},
	}, nil
}

func (i *infrastructure) Close() {
	if i == nil {
		return
	}

	if i.Redis != nil {
		_ = i.Redis.Close()
	}

	if i.SQLDB != nil {
		_ = i.SQLDB.Close()
	}
}
