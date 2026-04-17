# 第 09 次提交：接入 RocketMQ 实现异步下单与消费幂等

## 本次目标

把秒杀链路从：

`Redis Lua 准入 -> 同步数据库下单`

升级成：

`Redis Lua 准入 -> RocketMQ 入队 -> Consumer 异步落库`

## 我这次做了什么

1. 接入 RocketMQ 本地开发环境
   - `namesrv`
   - `broker`
   - `proxy`
2. 增加 RocketMQ 配置项
3. 抽出公共启动模块 `internal/bootstrap`
   - API 和 consumer 共享配置、日志和基础设施初始化逻辑
4. 增加秒杀订单消息结构
5. 增加 RocketMQ producer 封装
6. 增加 RocketMQ simple consumer 封装
7. 新增 `cmd/consumer`
   - 独立运行异步订单消费者
8. 秒杀接口改成异步入队
   - API 返回 `queued`
   - 返回 `order_no`
9. 增加 `mq_consume_logs` 表
   - 用消息 ID 做唯一约束
   - 在事务最前面记录消费，避免重复消息二次扣库存
10. 活动详情和列表读取时，会优先叠加 Redis 实时库存视图
    - 这样在 MQ 异步落库前，库存展示也不会明显落后

## 为什么要把消费幂等放在事务最前面

因为如果重复消息先扣库存，再发现订单已经存在，就会出现：

- 订单没有重复创建
- 库存却被重复扣减

这是秒杀系统里很危险的错误。

所以正确顺序应该是：

1. 先记录“这条消息处理过没有”
2. 再执行库存扣减和订单写入

## 关键代码入口

- `cmd/api/main.go`
  - API 服务启动入口，接入 RocketMQ producer
- `cmd/consumer/main.go`
  - 异步订单消费者启动入口
- `internal/bootstrap/runtime.go`
  - 公共启动逻辑
- `internal/mq/rocketmq/producer.go`
  - RocketMQ 生产者封装
- `internal/mq/rocketmq/consumer.go`
  - RocketMQ simple consumer 封装
- `internal/service/order_service.go`
  - 秒杀入队和异步订单处理逻辑
- `internal/repository/order_repository.go`
  - 原生 SQL 落库事务与消费幂等

## 本次如何测试

### 1. 自动化测试

```bash
go test ./...
```

### 2. 本地集成验证

1. `docker compose up -d`
2. 用 `mqadmin` 创建 topic 和 consumer group
3. 启动 API
4. 启动 consumer
5. 发起秒杀请求
6. 观察接口先返回 `queued`
7. 稍后查询订单详情，确认订单已被异步消费并落库

## 本次实际验证结果

- `go test ./...`：通过
- API 启动成功，RocketMQ producer 初始化成功
- consumer 启动成功，RocketMQ simple consumer 初始化成功
- 秒杀接口返回 HTTP 200，响应状态为 `queued`
- 稍后查询 `GET /api/v1/orders/{orderNo}`：返回 HTTP 200，说明消息已被异步消费并完成落库

## 这次踩到的真实问题

1. 当前 Docker 环境直接拉官方镜像有网络问题
   - 最终使用 `docker.m.daocloud.io/apache/rocketmq:5.3.2` 镜像代理完成本地联调
2. RocketMQ broker 用最接近官方 Compose 的方式最稳
   - `namesrv + broker + proxy`
3. simple consumer 轮询时，`MESSAGE_NOT_FOUND` 是正常现象
   - 这类情况不应该当成真正错误处理

## 下一步预告

下一次提交会做链路加固：

- 用户 / IP / 活动维度限流
- 更清晰的订单状态查询
- 防重复提交
- 错误码和响应语义继续整理
