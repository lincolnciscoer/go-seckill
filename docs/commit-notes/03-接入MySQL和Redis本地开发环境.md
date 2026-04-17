# 第 03 次提交：接入 MySQL 和 Redis 本地开发环境

## 本次目标

把秒杀系统最基础的两个外部依赖接进来：

- MySQL：承担最终数据落库
- Redis：承担高并发缓存与后续 Lua 脚本执行

同时保证本地开发环境可以统一启动，而不是靠手工逐个安装和配置。

## 我这次做了什么

1. 新增 `docker-compose.yml`
   - 统一启动 MySQL 8.4 和 Redis 7.4
2. 新增 `.env.example`
   - 给 Docker Compose 提供本地开发默认参数
3. 新增 MySQL 初始化 SQL
   - 预创建后续会用到的核心表
4. 扩展配置系统
   - 增加 `mysql` 和 `redis` 配置项
5. 接入 MySQL 连接初始化
   - 使用 GORM 建立连接
   - 启动时做 ping 校验
6. 接入 Redis 连接初始化
   - 使用 `go-redis/v9`
   - 启动时做 ping 校验
7. 扩展健康检查接口
   - 返回外部依赖状态
   - MySQL / Redis 异常时返回 `503`
8. 补充本地开发文档
   - 增加 `docs/local-dev.md`

## 为什么这一步要这样做

### 1. 为什么现在就接 Docker Compose

因为 MySQL 和 Redis 不是“某天再说”的附属品，它们是后面登录、商品、活动、订单、秒杀链路的基础设施。

如果没有一个统一的本地依赖启动方式，后面每一步都可能被环境问题打断。

### 2. 为什么 MySQL 现在就用 GORM 建连接

因为我们后面普通业务链路本来就打算使用 GORM。

现在先把连接层接进来，后续做用户、商品、活动时就不用再重复搭这一层。

### 3. 为什么健康检查要扩展成“依赖探活”

如果健康检查只返回“服务进程活着”，但数据库和 Redis 已经挂了，那这个服务其实并不算真正可用。

所以这一步开始，健康检查不再只是看进程本身，还要看关键依赖是否可达。

## 关键代码入口

- `docker-compose.yml`
  - 本地基础设施编排入口
- `deploy/mysql/init/001_init.sql`
  - 初始化库表结构
- `internal/store/mysql/`
  - MySQL 连接与健康检查
- `internal/store/redis/`
  - Redis 连接与健康检查
- `cmd/api/main.go`
  - 启动时初始化外部依赖
- `internal/transport/http/handler/health.go`
  - 汇总服务与依赖健康状态

## 你现在应该掌握什么

### 1. 启动失败不一定是坏事

当前设计是：如果 MySQL 或 Redis 连不上，应用直接启动失败。

这是“快速失败”思路，优点是问题暴露得更早、更直接。

### 2. 健康检查的两个层次

- 进程健康：服务程序是否活着
- 依赖健康：数据库、缓存是否可用

实际项目里，这两个层次都很重要。

### 3. 为什么要在 SQL 脚本里提前建表

因为后面每一阶段都要围绕这些核心表继续迭代。

先把最小表结构落下来，后面做业务时会更顺。

## 本次如何测试

### 1. 自动化测试

```bash
go test ./...
```

### 2. 启动本地依赖

```bash
docker compose --env-file .env.example up -d
```

### 3. 启动应用

```bash
go run ./cmd/api
```

### 4. 验证健康检查

```bash
curl http://127.0.0.1:8080/healthz
```

预期：

- 返回 HTTP 200
- `dependencies` 中包含 `mysql` 和 `redis`
- 两者状态均为 `up`

## 本次实际验证结果

- `go test ./...`：通过
- `docker compose --env-file .env.example up -d`：MySQL 和 Redis 成功启动并进入健康状态
- `go run ./cmd/api`：启动成功
- `GET /healthz`：返回 HTTP 200，`dependencies` 中的 `mysql` 和 `redis` 均为 `up`
- `GET /swagger/index.html`：返回 HTTP 200，Swagger UI 页面可打开

## 这次踩到的真实问题

这次联调里我们遇到了两个非常真实的环境问题：

1. `mysql:8.4` 本地启动参数兼容性不好
   - 原本尝试使用 `MySQL 8.4`
   - 但它对本地开发常见的认证插件和启动参数不够友好
   - 最终切换为更稳妥的 `mysql:8.0.41`

2. Windows + Docker Desktop 下，Go 应用连接 MySQL 不能直接用 `127.0.0.1`
   - 实际验证中，MySQL 通过 `127.0.0.1:3306` 会返回鉴权失败
   - 改成 `host.docker.internal:3306` 后连接恢复正常
   - 所以当前默认配置里，MySQL host 已调整为 `host.docker.internal`

## 下一步预告

下一次提交会实现账号密码登录和 JWT，把“用户是谁”这条链路先打通。
