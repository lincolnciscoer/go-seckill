package repository

import (
	"context"
	"errors"
	"time"

	"go-seckill/internal/model"

	"gorm.io/gorm"
)

type ActivityView struct {
	ID             uint64    `json:"id"`
	ProductID      uint64    `json:"product_id"`
	Name           string    `json:"name"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
	Status         int8      `json:"status"`
	TotalStock     int       `json:"total_stock"`
	AvailableStock int       `json:"available_stock"`
	SoldStock      int       `json:"sold_stock"`
	CreatedAt      time.Time `json:"created_at"`
}

type ActivityRepository interface {
	CreateWithStock(ctx context.Context, activity *model.SeckillActivity, stock *model.SeckillStock) error
	List(ctx context.Context) ([]ActivityView, error)
}

type GormActivityRepository struct {
	db *gorm.DB
}

func NewGormActivityRepository(db *gorm.DB) *GormActivityRepository {
	return &GormActivityRepository{db: db}
}

func (r *GormActivityRepository) CreateWithStock(ctx context.Context, activity *model.SeckillActivity, stock *model.SeckillStock) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(activity).Error; err != nil {
			return err
		}

		stock.ActivityID = activity.ID
		return tx.Create(stock).Error
	})
}

func (r *GormActivityRepository) List(ctx context.Context) ([]ActivityView, error) {
	var views []ActivityView
	err := r.db.WithContext(ctx).
		Table("seckill_activities AS a").
		Select(`
			a.id,
			a.product_id,
			a.name,
			a.start_time,
			a.end_time,
			a.status,
			a.created_at,
			COALESCE(s.total_stock, 0) AS total_stock,
			COALESCE(s.available_stock, 0) AS available_stock,
			COALESCE(s.sold_stock, 0) AS sold_stock
		`).
		Joins("LEFT JOIN seckill_stocks AS s ON s.activity_id = a.id").
		Order("a.id DESC").
		Scan(&views).Error
	return views, err
}

func IsRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
