# go-seckill

`go-seckill` 是一个循序渐进实现的 Go 电商秒杀系统后端项目。

当前阶段目标：

- 初始化 Git 仓库和 Go 模块
- 跑通最小 Gin 服务，并逐步补齐工程基础设施
- 提供基础健康检查接口 `GET /healthz`
- 接入配置系统、结构化日志、优雅退出和 Swagger 骨架
- 接入 MySQL 和 Redis 本地开发环境
- 为后续鉴权、秒杀、缓存、消息队列等模块预留清晰目录结构

## 当前目录结构

```text
.
|-- cmd
|   `-- api
|       `-- main.go
|-- configs
|   `-- config.example.yaml
|-- docs
|   |-- local-dev.md
|   `-- commit-notes
|       `-- 01-初始化仓库并跑通最小服务.md
|       `-- 02-接入配置系统日志优雅退出和Swagger骨架.md
|       `-- 03-接入MySQL和Redis本地开发环境.md
|-- docs/swagger
|-- deploy
|   `-- mysql
|       `-- init
|           `-- 001_init.sql
|-- internal
|   |-- config
|   |   |-- config.go
|   |   `-- config_test.go
|   |-- errs
|   |   `-- code.go
|   |-- health
|   |   `-- checker.go
|   |-- store
|   |   |-- mysql
|   |   |   |-- client.go
|   |   |   `-- health.go
|   |   `-- redis
|   |       |-- client.go
|   |       `-- health.go
|-- internal
|   `-- transport
|       `-- http
|           |-- handler
|           |   |-- health.go
|           |   `-- health_test.go
|           |-- middleware
|           |   |-- access_log.go
|           |   `-- recovery.go
|           |-- response
|           |   `-- response.go
|           `-- router
|               `-- router.go
|-- .env.example
|-- docker-compose.yml
`-- pkg
    `-- logger
        `-- logger.go
```

## 快速开始

1. 安装 Go 1.26+
2. 在项目根目录执行：

```bash
docker compose --env-file .env.example up -d
go run ./cmd/api
```

3. 打开浏览器或使用 curl 访问：

```bash
curl http://127.0.0.1:8080/healthz
```

预期返回：

```json
{
  "code": "OK",
  "message": "success",
  "data": {
    "status": "ok",
    "service": "go-seckill",
    "time": "2026-01-01T00:00:00Z",
    "dependencies": [
      {
        "name": "mysql",
        "status": "up"
      },
      {
        "name": "redis",
        "status": "up"
      }
    ]
  }
}
```

4. 查看 Swagger 文档：

```text
http://127.0.0.1:8080/swagger/index.html
```

## 学习路线

这个项目会采用“每次只实现一小块”的方式推进：

1. 先把基础服务骨架搭起来
2. 再接配置、日志、Swagger
3. 再接入 MySQL、Redis
4. 再实现登录、商品、活动、订单、秒杀主链路
5. 最后补监控、链路追踪、压测和项目展示材料

每次提交都会在 `docs/commit-notes/` 下配一篇教学型文档，帮助你理解本次为什么这么做、怎么测试、你应该学会什么。
