package model

import "time"

const (
	OrderStatusCreated int8 = 1
)

// Order 对应数据库里的订单记录。
// 这一步先围绕秒杀下单需要的字段建模，后面接支付时再扩展支付状态等信息。
type Order struct {
	ID         uint64    `json:"id"`
	OrderNo    string    `json:"order_no"`
	UserID     uint64    `json:"user_id"`
	ActivityID uint64    `json:"activity_id"`
	ProductID  uint64    `json:"product_id"`
	Quantity   int       `json:"quantity"`
	Amount     int64     `json:"amount"`
	Status     int8      `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
