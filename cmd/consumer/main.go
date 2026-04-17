package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go-seckill/internal/bootstrap"
	"go-seckill/internal/cache"
	"go-seckill/internal/mq/rocketmq"
	"go-seckill/internal/repository"
	"go-seckill/internal/service"

	"go.uber.org/zap"
)

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

	orderRepository := repository.NewSQLOrderRepository(infra.SQLDB)
	activityCache := cache.NewActivityCache(infra.Redis)
	asyncOrderService := service.NewAsyncOrderService(orderRepository, activityCache)

	consumer, err := rocketmq.NewConsumer(cfg.RocketMQ, appLogger, asyncOrderService)
	if err != nil {
		appLogger.Fatal("failed to initialize rocketmq consumer", zap.Error(err))
	}
	defer func() {
		_ = consumer.Shutdown()
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := consumer.Run(ctx); err != nil {
			appLogger.Fatal("rocketmq consumer stopped unexpectedly", zap.Error(err))
		}
	}()

	appLogger.Info("rocketmq consumer started",
		zap.String("topic", cfg.RocketMQ.Topic),
		zap.String("consumer_group", cfg.RocketMQ.ConsumerGroup),
	)

	stopSignal := make(chan os.Signal, 1)
	signal.Notify(stopSignal, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(stopSignal)

	<-stopSignal
	cancel()
}
