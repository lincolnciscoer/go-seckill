package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"go-seckill/internal/model"
	"go-seckill/internal/repository"
)

var (
	ErrProductMissing      = errors.New("product not found")
	ErrInvalidActivityTime = errors.New("invalid activity time")
)

type ActivityService struct {
	products   repository.ProductRepository
	activities repository.ActivityRepository
}

type CreateActivityInput struct {
	ProductID  uint64
	Name       string
	StartTime  time.Time
	EndTime    time.Time
	Status     int8
	TotalStock int
}

func NewActivityService(products repository.ProductRepository, activities repository.ActivityRepository) *ActivityService {
	return &ActivityService{
		products:   products,
		activities: activities,
	}
}

func (s *ActivityService) Create(ctx context.Context, input CreateActivityInput) error {
	product, err := s.products.GetByID(ctx, input.ProductID)
	if err != nil {
		return err
	}
	if product == nil {
		return ErrProductMissing
	}

	if !input.StartTime.Before(input.EndTime) {
		return ErrInvalidActivityTime
	}

	activity := &model.SeckillActivity{
		ProductID: input.ProductID,
		Name:      strings.TrimSpace(input.Name),
		StartTime: input.StartTime,
		EndTime:   input.EndTime,
		Status:    input.Status,
	}
	if activity.Status == 0 {
		activity.Status = 1
	}

	stock := &model.SeckillStock{
		TotalStock:     input.TotalStock,
		AvailableStock: input.TotalStock,
		SoldStock:      0,
	}

	return s.activities.CreateWithStock(ctx, activity, stock)
}

func (s *ActivityService) List(ctx context.Context) ([]repository.ActivityView, error) {
	return s.activities.List(ctx)
}
