package redis

import (
	"context"

	goredis "github.com/redis/go-redis/v9"
)

type healthChecker struct {
	client *goredis.Client
}

func NewHealthChecker(client *goredis.Client) *healthChecker {
	return &healthChecker{client: client}
}

func (c *healthChecker) Name() string {
	return "redis"
}

func (c *healthChecker) Check(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}
