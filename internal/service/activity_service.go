package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go-seckill/internal/cache"
	"go-seckill/internal/model"
	"go-seckill/internal/repository"
	"golang.org/x/sync/singleflight"
)

var (
	ErrProductMissing      = errors.New("product not found")
	ErrInvalidActivityTime = errors.New("invalid activity time")
	ErrActivityNotFound    = errors.New("activity not found")
)

type ActivityService struct {
	products    repository.ProductRepository
	activities  repository.ActivityRepository
	cache       *cache.ActivityCache
	listGroup   singleflight.Group
	detailGroup singleflight.Group
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

	if err := s.activities.CreateWithStock(ctx, activity, stock); err != nil {
		return err
	}

	if s.cache != nil {
		_ = s.cache.InvalidateActivity(ctx, activity.ID)
	}

	return nil
}

func (s *ActivityService) List(ctx context.Context) ([]repository.ActivityView, error) {
	if s.cache != nil {
		if activities, hit, err := s.cache.GetActivityList(ctx); err == nil && hit {
			s.applyStockSnapshot(ctx, activities)
			return activities, nil
		}
	}

	result, err, _ := s.listGroup.Do("activity:list", func() (any, error) {
		if s.cache != nil {
			if activities, hit, cacheErr := s.cache.GetActivityList(ctx); cacheErr == nil && hit {
				return activities, nil
			}
		}

		activities, queryErr := s.activities.List(ctx)
		if queryErr != nil {
			return nil, queryErr
		}

		if s.cache != nil {
			_ = s.cache.SetActivityList(ctx, activities)
		}

		return activities, nil
	})
	if err != nil {
		return nil, err
	}

	activities := cloneActivityViews(result.([]repository.ActivityView))
	s.applyStockSnapshot(ctx, activities)
	return activities, nil
}

func (s *ActivityService) GetByID(ctx context.Context, activityID uint64) (*repository.ActivityView, error) {
	if s.cache != nil {
		if activity, hit, err := s.cache.GetActivityDetail(ctx, activityID); err == nil && hit {
			if activity == nil {
				return nil, ErrActivityNotFound
			}
			s.applyActivityStock(ctx, activity)
			return activity, nil
		}
	}

	result, err, _ := s.detailGroup.Do(fmt.Sprintf("activity:%d", activityID), func() (any, error) {
		if s.cache != nil {
			if activity, hit, cacheErr := s.cache.GetActivityDetail(ctx, activityID); cacheErr == nil && hit {
				return activity, nil
			}
		}

		activity, queryErr := s.activities.GetByID(ctx, activityID)
		if queryErr != nil {
			return nil, queryErr
		}
		if activity == nil {
			if s.cache != nil {
				_ = s.cache.SetActivityEmpty(ctx, activityID)
			}
			return nil, nil
		}

		if s.cache != nil {
			_ = s.cache.SetActivityDetail(ctx, *activity)
		}

		return activity, nil
	})
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, ErrActivityNotFound
	}

	activity := cloneActivityView(result.(*repository.ActivityView))
	s.applyActivityStock(ctx, activity)
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

func (s *ActivityService) applyStockSnapshot(ctx context.Context, activities []repository.ActivityView) {
	for idx := range activities {
		s.applyActivityStock(ctx, &activities[idx])
	}
}

func (s *ActivityService) applyActivityStock(ctx context.Context, activity *repository.ActivityView) {
	if s.cache == nil || activity == nil {
		return
	}

	stock, hit, err := s.cache.GetActivityStock(ctx, activity.ID)
	if err != nil || !hit {
		return
	}

	activity.AvailableStock = stock
	if activity.TotalStock >= stock {
		activity.SoldStock = activity.TotalStock - stock
	}
}

func cloneActivityView(activity *repository.ActivityView) *repository.ActivityView {
	if activity == nil {
		return nil
	}

	cloned := *activity
	return &cloned
}

func cloneActivityViews(activities []repository.ActivityView) []repository.ActivityView {
	if activities == nil {
		return nil
	}

	cloned := make([]repository.ActivityView, len(activities))
	copy(cloned, activities)
	return cloned
}
