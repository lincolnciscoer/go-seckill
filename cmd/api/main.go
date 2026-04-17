package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "go-seckill/docs/swagger"
	"go-seckill/internal/config"
	"go-seckill/internal/transport/http/router"
	"go-seckill/pkg/logger"

	"go.uber.org/zap"
)

// @title go-seckill API
// @version 0.1.0
// @description A step-by-step Go seckill backend project
// @BasePath /
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

	engine := router.NewEngine(router.Dependencies{
		Config: cfg,
		Logger: appLogger,
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
