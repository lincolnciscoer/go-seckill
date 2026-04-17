# 第 04 次提交：实现账号密码登录与 JWT 鉴权

## 本次目标

让系统第一次具备“用户是谁”的能力。

这一提交要完成：

1. 用户注册
2. 用户登录
3. JWT 签发与校验
4. 受保护接口 `/api/v1/me`

## 我这次做了什么

1. 增加了 `users` 对应的领域模型
2. 增加了基于 GORM 的用户仓储
3. 增加了 `AuthService`
   - 注册
   - 登录
   - 根据用户 ID 查询用户
4. 接入了 `bcrypt` 密码哈希
5. 接入了 JWT 签发与解析
6. 增加了鉴权中间件
7. 新增接口：
   - `POST /api/v1/auth/register`
   - `POST /api/v1/auth/login`
   - `GET /api/v1/me`

## 为什么这一步重要

秒杀系统后面所有关键行为几乎都依赖“当前请求是谁发起的”。

如果用户体系不先打通，后面的：

- 一人一单
- 订单归属
- 抢购资格判断
- 订单查询

都没法认真做。

## 关键代码入口

- `internal/model/user.go`
- `internal/repository/user_repository.go`
- `internal/service/auth_service.go`
- `internal/security/jwt/manager.go`
- `internal/transport/http/middleware/auth.go`
- `internal/transport/http/handler/auth.go`

## 本次如何测试

### 1. 自动化测试

```bash
go test ./...
```

### 2. 手动验证

1. 注册用户
2. 登录获取 token
3. 带 `Authorization: Bearer <token>` 请求 `/api/v1/me`
4. 不带 token 请求 `/api/v1/me`

## 本次实际验证结果

- `go test ./...`：通过
- `POST /api/v1/auth/register`：返回 HTTP 200，成功创建新用户并返回 token
- `POST /api/v1/auth/login`：返回 HTTP 200，成功返回 token
- `GET /api/v1/me`：携带 Bearer Token 时返回 HTTP 200
- `GET /api/v1/me`：不带 token 时返回 HTTP 401
- `POST /api/v1/auth/login`：错误密码时返回 HTTP 401
