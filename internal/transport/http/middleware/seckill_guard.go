package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"go-seckill/internal/cache"
	"go-seckill/internal/errs"
	httpresponse "go-seckill/internal/transport/http/response"
)

type SeckillGuard struct {
	redis   *goredis.Client
	limiter *cache.RateLimiter
	logger  *zap.Logger
}

func NewSeckillGuard(redisClient *goredis.Client, logger *zap.Logger) *SeckillGuard {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &SeckillGuard{
		redis:   redisClient,
		limiter: cache.NewRateLimiter(redisClient),
		logger:  logger,
	}
}

func (g *SeckillGuard) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		currentUser, ok := GetCurrentUser(c)
		if !ok {
			httpresponse.Error(c, http.StatusUnauthorized, errs.CodeUnauthorized, "")
			return
		}

		activityID := c.Param("id")
		if _, err := strconv.ParseUint(activityID, 10, 64); err != nil {
			httpresponse.Error(c, http.StatusBadRequest, errs.CodeBadRequest, "invalid activity id")
			return
		}

		ctx := c.Request.Context()
		if !g.allow(ctx, "seckill:rl:user:"+activityID+":"+strconv.FormatUint(currentUser.UserID, 10), 5, 5*time.Second) {
			httpresponse.Error(c, http.StatusTooManyRequests, errs.CodeRateLimited, "user rate limit exceeded")
			return
		}

		if !g.allow(ctx, "seckill:rl:ip:"+activityID+":"+c.ClientIP(), 20, 5*time.Second) {
			httpresponse.Error(c, http.StatusTooManyRequests, errs.CodeRateLimited, "ip rate limit exceeded")
			return
		}

		if !g.allow(ctx, "seckill:rl:activity:"+activityID, 200, time.Second) {
			httpresponse.Error(c, http.StatusTooManyRequests, errs.CodeRateLimited, "activity rate limit exceeded")
			return
		}

		guardKey := "seckill:submit:" + activityID + ":" + strconv.FormatUint(currentUser.UserID, 10)
		acquired, err := g.redis.SetNX(ctx, guardKey, "1", 2*time.Second).Result()
		if err != nil {
			g.logger.Warn("duplicate submit guard failed open", zap.Error(err))
			c.Next()
			return
		}
		if !acquired {
			httpresponse.Error(c, http.StatusConflict, errs.CodeDuplicateSubmit, "")
			return
		}

		defer func() {
			_ = g.redis.Del(context.Background(), guardKey).Err()
		}()

		c.Next()
	}
}

func (g *SeckillGuard) allow(ctx context.Context, key string, limit int64, window time.Duration) bool {
	allowed, err := g.limiter.Allow(ctx, key, limit, window)
	if err != nil {
		g.logger.Warn("rate limiter failed open", zap.String("key", key), zap.Error(err))
		return true
	}

	return allowed
}
