package service

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	"go-seckill/internal/cache"
	"go-seckill/internal/model"
	"go-seckill/internal/repository"
)

type countingActivityRepository struct {
	activity    *repository.ActivityView
	delay       time.Duration
	getByIDCall int32
}

func (r *countingActivityRepository) CreateWithStock(context.Context, *model.SeckillActivity, *model.SeckillStock) error {
	return nil
}

func (r *countingActivityRepository) List(context.Context) ([]repository.ActivityView, error) {
	return nil, nil
}

func (r *countingActivityRepository) GetByID(context.Context, uint64) (*repository.ActivityView, error) {
	atomic.AddInt32(&r.getByIDCall, 1)
	if r.delay > 0 {
		time.Sleep(r.delay)
	}
	return r.activity, nil
}

func TestActivityServiceCachesMissingActivity(t *testing.T) {
	server := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: server.Addr()})
	defer func() {
		_ = client.Close()
		server.Close()
	}()

	repo := &countingActivityRepository{}
	service := NewActivityService(
		&fakeProductRepository{products: map[uint64]*model.Product{}},
		repo,
		cache.NewActivityCache(client),
	)

	_, err := service.GetByID(context.Background(), 404)
	if err != ErrActivityNotFound {
		t.Fatalf("expected ErrActivityNotFound, got %v", err)
	}

	_, err = service.GetByID(context.Background(), 404)
	if err != ErrActivityNotFound {
		t.Fatalf("expected ErrActivityNotFound on second call, got %v", err)
	}

	if got := atomic.LoadInt32(&repo.getByIDCall); got != 1 {
		t.Fatalf("expected repository GetByID to be called once, got %d", got)
	}
}

func TestActivityServiceSingleflightPreventsBreakdown(t *testing.T) {
	server := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: server.Addr()})
	defer func() {
		_ = client.Close()
		server.Close()
	}()

	repo := &countingActivityRepository{
		activity: &repository.ActivityView{
			ID:             1,
			ProductID:      2,
			Name:           "hot-activity",
			StartTime:      time.Now(),
			EndTime:        time.Now().Add(time.Hour),
			Status:         1,
			TotalStock:     10,
			AvailableStock: 10,
		},
		delay: 50 * time.Millisecond,
	}
	service := NewActivityService(
		&fakeProductRepository{products: map[uint64]*model.Product{}},
		repo,
		cache.NewActivityCache(client),
	)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			activity, err := service.GetByID(context.Background(), 1)
			if err != nil {
				t.Errorf("get by id: %v", err)
				return
			}
			if activity == nil || activity.ID != 1 {
				t.Errorf("unexpected activity: %#v", activity)
			}
		}()
	}
	wg.Wait()

	if got := atomic.LoadInt32(&repo.getByIDCall); got != 1 {
		t.Fatalf("expected repository GetByID to be called once under hot-key concurrency, got %d", got)
	}
}
