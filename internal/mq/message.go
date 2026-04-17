package mq

import (
	"context"
	"time"
)

const (
	SeckillOrderTopicTag = "seckill-order"
)

type SeckillOrderMessage struct {
	OrderNo    string    `json:"order_no"`
	UserID     uint64    `json:"user_id"`
	ActivityID uint64    `json:"activity_id"`
	ProductID  uint64    `json:"product_id"`
	Quantity   int       `json:"quantity"`
	Amount     int64     `json:"amount"`
	Status     int8      `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

type SeckillOrderProducer interface {
	SendSeckillOrder(ctx context.Context, message *SeckillOrderMessage) (string, error)
	Shutdown() error
}
