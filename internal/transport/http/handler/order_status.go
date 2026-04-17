package handler

import "time"

type OrderStatusResponse struct {
	OrderNo    string    `json:"order_no"`
	UserID     uint64    `json:"user_id"`
	ActivityID uint64    `json:"activity_id"`
	Status     string    `json:"status"`
	UpdatedAt  time.Time `json:"updated_at"`
}
