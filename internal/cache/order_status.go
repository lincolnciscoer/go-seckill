package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const (
	orderStatusKeyPrefix = "order:status:"
	orderStatusTTL       = 30 * time.Minute
)

type OrderStatusPayload struct {
	OrderNo    string    `json:"order_no"`
	UserID     uint64    `json:"user_id"`
	ActivityID uint64    `json:"activity_id"`
	Status     string    `json:"status"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type OrderStatusCache struct {
	client *goredis.Client
}

func NewOrderStatusCache(client *goredis.Client) *OrderStatusCache {
	return &OrderStatusCache{client: client}
}

func (c *OrderStatusCache) Set(ctx context.Context, payload OrderStatusPayload) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, orderStatusKey(payload.OrderNo), raw, orderStatusTTL).Err()
}

func (c *OrderStatusCache) Get(ctx context.Context, orderNo string) (*OrderStatusPayload, bool, error) {
	value, err := c.client.Get(ctx, orderStatusKey(orderNo)).Result()
	if err == goredis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	var payload OrderStatusPayload
	if err := json.Unmarshal([]byte(value), &payload); err != nil {
		return nil, false, err
	}

	return &payload, true, nil
}

func orderStatusKey(orderNo string) string {
	return fmt.Sprintf("%s%s", orderStatusKeyPrefix, orderNo)
}
