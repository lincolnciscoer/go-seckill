package model

import "time"

type SeckillActivity struct {
	ID        uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProductID uint64    `gorm:"column:product_id;not null" json:"product_id"`
	Name      string    `gorm:"column:name;size:128;not null" json:"name"`
	StartTime time.Time `gorm:"column:start_time;not null" json:"start_time"`
	EndTime   time.Time `gorm:"column:end_time;not null" json:"end_time"`
	Status    int8      `gorm:"column:status;not null" json:"status"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (SeckillActivity) TableName() string {
	return "seckill_activities"
}

type SeckillStock struct {
	ID             uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ActivityID     uint64    `gorm:"column:activity_id;not null" json:"activity_id"`
	TotalStock     int       `gorm:"column:total_stock;not null" json:"total_stock"`
	AvailableStock int       `gorm:"column:available_stock;not null" json:"available_stock"`
	SoldStock      int       `gorm:"column:sold_stock;not null" json:"sold_stock"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (SeckillStock) TableName() string {
	return "seckill_stocks"
}
