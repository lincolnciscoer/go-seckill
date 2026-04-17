package main

import (
	"fmt"
	"log"
	"os"

	"go-seckill/internal/transport/http/router"
)

const defaultHTTPPort = "8080"

func main() {
	engine := router.NewEngine()
	port := getHTTPPort()

	// 第一版先把最小 HTTP 服务跑起来。
	// 后续我们会在这里继续接入配置系统、日志系统、优雅退出和观测能力。
	addr := fmt.Sprintf(":%s", port)
	log.Printf("go-seckill api server listening on %s", addr)

	if err := engine.Run(addr); err != nil {
		log.Fatalf("failed to start http server: %v", err)
	}
}

func getHTTPPort() string {
	if port := os.Getenv("APP_PORT"); port != "" {
		return port
	}

	return defaultHTTPPort
}
