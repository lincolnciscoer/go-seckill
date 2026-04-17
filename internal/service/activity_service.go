package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"go-seckill/internal/cache"
	"go-seckill/internal/model"
	"go-seckill/internal/repository"
)

var (
	ErrProductMissing      = errors.New("product not found")
	ErrInvalidActivityTime = errors.New("invalid activity time")
	ErrActivityNotFound    = errors.New("activity not found")
)

type ActivityService struct {
	products   repository.ProductRepository
	activities repository.ActivityRepository
	cache      *cache.ActivityCache
}

type CreateActivityInput struct {
	ProductID  uint64
	Name       string
	StartTime  time.Time
	EndTime    time.Time
	Status     int8
	TotalStock int
}

func NewActivityService(products repository.ProductRepository, activities repository.ActivityRepository, activityCache *cache.ActivityCache) *ActivityService {
	return &ActivityService{
		products:   products,
		activities: activities,
		cache:      activityCache,
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
	if s.cache != nil {
		if activities, hit, err := s.cache.GetActivityList(ctx); err == nil && hit {
			return activities, nil
		}
	}

	activities, err := s.activities.List(ctx)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		_ = s.cache.SetActivityList(ctx, activities)
	}

	return activities, nil
}

func (s *ActivityService) GetByID(ctx context.Context, activityID uint64) (*repository.ActivityView, error) {
	if s.cache != nil {
		if activity, hit, err := s.cache.GetActivityDetail(ctx, activityID); err == nil && hit {
			return activity, nil
		}
	}

	activity, err := s.activities.GetByID(ctx, activityID)
	if err != nil {
		return nil, err
	}
	if activity == nil {
		return nil, ErrActivityNotFound
	}

	if s.cache != nil {
		_ = s.cache.SetActivityDetail(ctx, *activity)
	}

	return activity, nil
}

func (s *ActivityService) Preheat(ctx context.Context, activityID uint64) error {
	activity, err := s.activities.GetByID(ctx, activityID)
	if err != nil {
		return err
	}
	if activity == nil {
		return ErrActivityNotFound
	}

	if s.cache != nil {
		if err := s.cache.SetActivityDetail(ctx, *activity); err != nil {
			return err
		}

		activities, err := s.activities.List(ctx)
		if err == nil {
			_ = s.cache.SetActivityList(ctx, activities)
		}
	}

	return nil
}
