package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"go-seckill/internal/config"
	"go-seckill/internal/health"
	mysqlstore "go-seckill/internal/store/mysql"
	redisstore "go-seckill/internal/store/redis"
	"go-seckill/pkg/logger"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Infrastructure struct {
	GormDB         *gorm.DB
	SQLDB          *sql.DB
	Redis          *goredis.Client
	HealthCheckers []health.Checker
}

func DefaultConfigPath() string {
	if path := os.Getenv("GO_SECKILL_CONFIG"); path != "" {
		return path
	}

	return "configs/config.example.yaml"
}

func LoadConfig(path string) (*config.Config, error) {
	return config.Load(path)
}

func InitLogger(cfg *config.Config) (*zap.Logger, error) {
	return logger.New(cfg.Log)
}

func InitInfrastructure(cfg *config.Config, appLogger *zap.Logger) (*Infrastructure, error) {
	gormDB, sqlDB, err := mysqlstore.New(cfg.MySQL)
	if err != nil {
		return nil, err
	}

	if err := mysqlstore.EnsureSchema(sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	appLogger.Info("mysql connected",
		zap.String("host", cfg.MySQL.Host),
		zap.Int("port", cfg.MySQL.Port),
		zap.String("database", cfg.MySQL.Database),
	)

	redisClient, err := redisstore.New(cfg.Redis)
	if err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	appLogger.Info("redis connected",
		zap.String("addr", cfg.Redis.Addr),
		zap.Int("db", cfg.Redis.DB),
	)

	return &Infrastructure{
		GormDB: gormDB,
		SQLDB:  sqlDB,
		Redis:  redisClient,
		HealthCheckers: []health.Checker{
			mysqlstore.NewHealthChecker(sqlDB),
			redisstore.NewHealthChecker(redisClient),
		},
	}, nil
}

func (i *Infrastructure) Close() {
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

func NewHTTPServer(addr string, cfg *config.Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}
}

func WaitForShutdown(server *http.Server, cfg *config.Config, appLogger *zap.Logger) error {
	stopSignal := make(chan os.Signal, 1)
	signal.Notify(stopSignal, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(stopSignal)

	sig := <-stopSignal
	appLogger.Info("received shutdown signal", zap.String("signal", sig.String()))

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	return server.Shutdown(ctx)
}

func ServerAddress(cfg *config.Config) string {
	return fmt.Sprintf(":%d", cfg.Server.Port)
}
