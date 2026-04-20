package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strconv"
	"time"

	"go-seckill/internal/repository"

	goredis "github.com/redis/go-redis/v9"
)

const (
	activityListKeyPrefix   = "seckill:activity:list"
	activityDetailKeyPrefix = "seckill:activity:"
	activityStockKeyPrefix  = "seckill:stock:"
	activityEmptyMarker     = "__nil__"
	defaultCacheTTL         = 10 * time.Minute
	defaultCacheTTLJitter   = 2 * time.Minute
	emptyCacheTTL           = 1 * time.Minute
	emptyCacheTTLJitter     = 30 * time.Second
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

func (c *ActivityCache) Client() *goredis.Client {
	return c.client
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

	return c.client.Set(ctx, activityListKeyPrefix, payload, ttlWithJitter(c.ttl, defaultCacheTTLJitter)).Err()
}

func (c *ActivityCache) GetActivityDetail(ctx context.Context, activityID uint64) (*repository.ActivityView, bool, error) {
	value, err := c.client.Get(ctx, activityDetailKey(activityID)).Result()
	if err == goredis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	if value == activityEmptyMarker {
		return nil, true, nil
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

	if err := c.client.Set(ctx, activityDetailKey(activity.ID), payload, ttlWithJitter(c.ttl, defaultCacheTTLJitter)).Err(); err != nil {
		return err
	}

	return c.client.Set(ctx, activityStockKey(activity.ID), activity.AvailableStock, ttlWithJitter(c.ttl, defaultCacheTTLJitter)).Err()
}

// SetActivityEmpty 用于缓存“不存在的活动”。
// 这样对于大量访问不存在 activityId 的请求，就不会每次都穿透到数据库。
func (c *ActivityCache) SetActivityEmpty(ctx context.Context, activityID uint64) error {
	return c.client.Set(
		ctx,
		activityDetailKey(activityID),
		activityEmptyMarker,
		ttlWithJitter(emptyCacheTTL, emptyCacheTTLJitter),
	).Err()
}

func (c *ActivityCache) InvalidateActivity(ctx context.Context, activityID uint64) error {
	return c.client.Del(
		ctx,
		activityListKeyPrefix,
		activityDetailKey(activityID),
		activityStockKey(activityID),
	).Err()
}

func (c *ActivityCache) GetActivityStock(ctx context.Context, activityID uint64) (int, bool, error) {
	value, err := c.client.Get(ctx, activityStockKey(activityID)).Result()
	if err == goredis.Nil {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}

	stock, err := strconv.Atoi(value)
	if err != nil {
		return 0, false, err
	}

	return stock, true, nil
}

func activityDetailKey(activityID uint64) string {
	return fmt.Sprintf("%s%d", activityDetailKeyPrefix, activityID)
}

func activityStockKey(activityID uint64) string {
	return fmt.Sprintf("%s%d", activityStockKeyPrefix, activityID)
}

func ttlWithJitter(base time.Duration, jitter time.Duration) time.Duration {
	if jitter <= 0 {
		return base
	}

	return base + time.Duration(rand.Int64N(int64(jitter)+1))
}
