package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go-seckill/internal/repository"

	goredis "github.com/redis/go-redis/v9"
)

const (
	activityListKeyPrefix   = "seckill:activity:list"
	activityDetailKeyPrefix = "seckill:activity:"
	activityStockKeyPrefix  = "seckill:stock:"
	defaultCacheTTL         = 10 * time.Minute
)

type ActivityCache struct {
	client *goredis.Client
	ttl    time.Duration
}

func NewActivityCache(client *goredis.Client) *ActivityCache {
	return &ActivityCache{
		client: client,
		ttl:    defaultCacheTTL,
	}
}

func (c *ActivityCache) GetActivityList(ctx context.Context) ([]repository.ActivityView, bool, error) {
	value, err := c.client.Get(ctx, activityListKeyPrefix).Result()
	if err == goredis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	var activities []repository.ActivityView
	if err := json.Unmarshal([]byte(value), &activities); err != nil {
		return nil, false, err
	}

	return activities, true, nil
}

func (c *ActivityCache) SetActivityList(ctx context.Context, activities []repository.ActivityView) error {
	payload, err := json.Marshal(activities)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, activityListKeyPrefix, payload, c.ttl).Err()
}

func (c *ActivityCache) GetActivityDetail(ctx context.Context, activityID uint64) (*repository.ActivityView, bool, error) {
	value, err := c.client.Get(ctx, activityDetailKey(activityID)).Result()
	if err == goredis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	var activity repository.ActivityView
	if err := json.Unmarshal([]byte(value), &activity); err != nil {
		return nil, false, err
	}

	return &activity, true, nil
}

func (c *ActivityCache) SetActivityDetail(ctx context.Context, activity repository.ActivityView) error {
	payload, err := json.Marshal(activity)
	if err != nil {
		return err
	}

	if err := c.client.Set(ctx, activityDetailKey(activity.ID), payload, c.ttl).Err(); err != nil {
		return err
	}

	return c.client.Set(ctx, activityStockKey(activity.ID), activity.AvailableStock, c.ttl).Err()
}

func activityDetailKey(activityID uint64) string {
	return fmt.Sprintf("%s%d", activityDetailKeyPrefix, activityID)
}

func activityStockKey(activityID uint64) string {
	return fmt.Sprintf("%s%d", activityStockKeyPrefix, activityID)
}
