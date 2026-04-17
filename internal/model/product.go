package model

import "time"

type Product struct {
	ID          uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"column:name;size:128;not null" json:"name"`
	Description string    `gorm:"column:description" json:"description"`
	Price       int64     `gorm:"column:price;not null" json:"price"`
	Status      int8      `gorm:"column:status;not null" json:"status"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Product) TableName() string {
	return "products"
}
