# go-seckill

`go-seckill` 是一个按阶段实现的 Go 电商秒杀系统后端项目。

当前已经完成的核心阶段：

1. 仓库初始化与最小 Gin 服务
2. 配置、日志、优雅退出、Swagger
3. MySQL / Redis 本地环境与健康检查
4. 账号密码登录与 JWT 鉴权
5. 商品与秒杀活动管理
6. 活动预热与缓存化查询
7. 订单领域与同步版秒杀闭环
8. Redis Lua 秒杀准入与库存预扣减
9. RocketMQ 异步下单与消费幂等
10. 限流、防重复提交与订单处理中状态

## 当前技术栈

- Go
- Gin
- GORM
- MySQL
- Redis
- Lua
- RocketMQ
- JWT
- Swagger
- Docker Compose

## 当前项目结构

```text
.
|-- cmd
|   |-- api
|   |   `-- main.go
|   `-- consumer
|       `-- main.go
|-- configs
|   `-- config.example.yaml
|-- deploy
|   `-- mysql
|       `-- init
|           `-- 001_init.sql
|-- docs
|   |-- commit-notes
|   `-- local-dev.md
|-- internal
|   |-- bootstrap
|   |-- cache
|   |-- config
|   |-- errs
|   |-- health
|   |-- model
|   |-- mq
|   |-- repository
|   |-- security
|   |-- service
|   |-- store
|   `-- transport
|-- .env.example
`-- docker-compose.yml
```

## 快速开始

1. 启动依赖：

```bash
docker compose --env-file .env.example up -d
```

2. 初始化 RocketMQ Topic 和 Consumer Group：

```bash
docker exec go-seckill-rmq-broker sh mqadmin updatetopic -n rocketmq-namesrv:9876 -c DefaultCluster -t SeckillOrderTopic
docker exec go-seckill-rmq-broker sh mqadmin updateSubGroup -n rocketmq-namesrv:9876 -c DefaultCluster -g go-seckill-order-consumer
```

3. 启动 API：

```bash
go run ./cmd/api
```

4. 启动异步订单消费者：

```bash
go run ./cmd/consumer
```

5. 打开 Swagger：

```text
http://127.0.0.1:8080/swagger/index.html
```

更多本地运行说明见 [docs/local-dev.md](C:/Users/TY/Desktop/tyai/go-seckill/docs/local-dev.md)。

## 关键接口

- `GET /healthz`
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `GET /api/v1/me`
- `POST /api/v1/products`
- `GET /api/v1/products`
- `POST /api/v1/activities`
- `GET /api/v1/activities`
- `GET /api/v1/activities/:id`
- `POST /api/v1/activities/:id/preheat`
- `POST /api/v1/seckill/activities/:id/attempt`
- `GET /api/v1/orders/:orderNo`
- `GET /api/v1/orders/me`

## 学习方式

这个项目按“每次只实现一小块”的节奏推进。

每次提交都会在 `docs/commit-notes/` 下附一篇教学型文档，说明：

- 本次目标
- 做了什么
- 为什么这么做
- 如何测试
- 这一步你应该掌握什么
