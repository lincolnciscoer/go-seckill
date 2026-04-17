# 压测与性能报告

## 压测环境

- API：本地 `go-seckill-api`
- Consumer：本地 `go-seckill-consumer`
- MySQL / Redis / RocketMQ：Docker Desktop
- 可观测组件：Prometheus / Grafana / Jaeger
- 压测工具：仓库内置 `cmd/loadtest`

## 压测命令

正常区间样例：

```bash
go run ./cmd/loadtest --base-url http://127.0.0.1:18092 --users 20 --concurrency 10 --stock 20
```

触发限流样例：

```bash
go run ./cmd/loadtest --base-url http://127.0.0.1:18092 --users 40 --concurrency 20 --stock 20
```

## 样例结果

### 样例一：正常区间

- 用户数：`20`
- 并发度：`10`
- 库存：`20`
- HTTP 结果：
  - `200`: `20`
- 业务结果：
  - `OK`: `20`
- 平均延迟：`25.35 ms`
- P50：`21 ms`
- P95：`38 ms`

### 样例二：主动触发限流

- 用户数：`40`
- 并发度：`20`
- 库存：`20`
- HTTP 结果：
  - `200`: `20`
  - `429`: `20`
- 业务结果：
  - `OK`: `20`
  - `RATE_LIMITED`: `20`
- 平均延迟：`34.8 ms`
- P50：`2 ms`
- P95：`79 ms`

## 结果解读

- 在正常区间内，异步秒杀入口能够稳定返回 `queued`，消费者随后完成异步落库。
- 在更高压的流量下，系统不会无上限放行请求，而是通过限流中间件主动返回 `429`。
- 这说明当前项目已经具备了“高并发下保护后端核心资源”的基础能力，而不是单纯依赖数据库硬抗。
- 从项目展示角度看，这两组结果能分别说明：
  - 系统在合理流量内可以正常工作
  - 系统在超出阈值时会主动保护自己
