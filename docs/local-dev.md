# 本地开发环境启动说明

## 依赖准备

- Docker Desktop
- Go 1.26+

## 启动 MySQL、Redis 和 RocketMQ

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
- `go-seckill-rmq-namesrv`、`go-seckill-rmq-broker`、`go-seckill-rmq-proxy` 为 `running`

## 初始化 RocketMQ Topic 和 Consumer Group

首次启动 RocketMQ 后执行：

```bash
docker exec go-seckill-rmq-broker sh mqadmin updatetopic -n rocketmq-namesrv:9876 -c DefaultCluster -t SeckillOrderTopic
docker exec go-seckill-rmq-broker sh mqadmin updateSubGroup -n rocketmq-namesrv:9876 -c DefaultCluster -g go-seckill-order-consumer
```

## 启动 API 服务

```bash
go run ./cmd/api
```

## 启动异步订单消费者

新开一个终端执行：

```bash
go run ./cmd/consumer
```

## 当前本地默认地址

- MySQL：`host.docker.internal:3306`
- Redis：`host.docker.internal:6379`
- RocketMQ Proxy：`host.docker.internal:8081`

这是为了兼容当前 Windows + Docker Desktop 环境。

## 验证服务与依赖

```bash
curl http://127.0.0.1:8080/healthz
```

预期响应中包含：

- `service=go-seckill`
- `dependencies`
- `mysql` 和 `redis` 状态均为 `up`

## 验证异步秒杀

1. 注册并登录用户
2. 创建商品和秒杀活动
3. 调用预热接口
4. 启动 `cmd/consumer`
5. 调用：

```bash
POST /api/v1/seckill/activities/{id}/attempt
```

预期先返回：

- `status=queued`
- `order_no`

稍等片刻后再查询：

```bash
GET /api/v1/orders/{orderNo}
```

如果 consumer 正常消费，订单会被查到。

## 停止本地依赖

```bash
docker compose --env-file .env.example down
```
