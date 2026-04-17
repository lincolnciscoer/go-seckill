# 第 11 次提交：接入 Prometheus、Grafana、OpenTelemetry 和 pprof

## 本次目标

让项目第一次具备完整的可观测能力：

1. Prometheus 指标采集
2. Grafana 可视化面板
3. OpenTelemetry Trace 初始化
4. pprof 性能分析入口

## 我这次做了什么

1. 增加了 `internal/observability`
   - HTTP 指标中间件
   - 业务指标记录函数
   - OTel tracing 初始化
   - pprof 路由注册
   - consumer 侧轻量 metrics / pprof server
2. API 新增：
   - `GET /metrics`
   - `GET /debug/pprof/*`
3. consumer 新增：
   - `GET /metrics`
   - `GET /debug/pprof/*`
4. 为关键业务链路增加了指标：
   - 秒杀尝试结果
   - MQ 消费结果
   - 订单状态流转
5. 为 API 和 consumer 初始化了独立的 tracing service name
6. 在 Docker Compose 中增加了：
   - Prometheus
   - Grafana
   - Jaeger
7. 增加了 Prometheus 抓取配置和 Grafana provisioning

## 本次如何测试

### 1. 自动化测试

```bash
go test ./...
```

### 2. 本地集成验证

1. `docker compose up -d`
2. 启动 API 与 consumer
3. 访问：
   - `http://127.0.0.1:8080/metrics`
   - `http://127.0.0.1:8080/debug/pprof/`
   - `http://127.0.0.1:19090/metrics`
   - `http://127.0.0.1:9090`
   - `http://127.0.0.1:3000`
   - `http://127.0.0.1:16686`
4. 发起几次秒杀请求
5. 在 Grafana 中查看指标变化，在 Jaeger 中查看 trace

## 本次实际验证结果

- `go test ./...`：通过
- API `/metrics` 可访问，并包含：
  - `go_seckill_http_requests_total`
  - `go_seckill_seckill_attempts_total`
  - `go_seckill_order_status_transitions_total`
- Consumer `/metrics` 可访问，并包含：
  - `go_seckill_mq_consume_total`
- API `/debug/pprof/` 可访问
- Prometheus `/-/ready` 返回 HTTP `200`
- Prometheus 查询 `go_seckill_http_requests_total` 返回 10 条时间序列
- Prometheus 查询 `go_seckill_mq_consume_total` 返回 1 条时间序列
- Grafana `/api/health` 返回 HTTP `200`
- Jaeger `/api/services` 返回服务列表，包含：
  - `go-seckill-api`
  - `go-seckill-consumer`
- 发起真实秒杀请求后，异步下单链路仍能正常落库
