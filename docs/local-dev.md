# 本地开发环境启动说明

## 依赖准备

- Docker Desktop
- Go 1.26+

## 启动 MySQL 和 Redis

在项目根目录执行：

```bash
docker compose --env-file .env.example up -d
```

查看容器状态：

```bash
docker compose --env-file .env.example ps
```

预期：

- `go-seckill-mysql` 为 `running (healthy)`
- `go-seckill-redis` 为 `running (healthy)`

## 启动应用

```bash
go run ./cmd/api
```

说明：

- 当前默认配置针对 Windows + Docker Desktop 调整了 MySQL 和 Redis 主机为 `host.docker.internal`

## 验证服务与依赖

```bash
curl http://127.0.0.1:8080/healthz
```

预期响应中包含：

- `service=go-seckill`
- `dependencies` 数组
- `mysql` 和 `redis` 的状态都为 `up`

## 停止本地依赖

```bash
docker compose --env-file .env.example down
```
