package service

import (
	"context"
	"testing"
	"time"

	"go-seckill/internal/model"
	"go-seckill/internal/repository"
)

type fakeProductRepository struct {
	products map[uint64]*model.Product
}

func (r *fakeProductRepository) Create(context.Context, *model.Product) error {
	return nil
}

func (r *fakeProductRepository) List(context.Context) ([]model.Product, error) {
	return nil, nil
}

func (r *fakeProductRepository) GetByID(_ context.Context, id uint64) (*model.Product, error) {
	return r.products[id], nil
}

type fakeActivityRepository struct {
	created bool
}

func (r *fakeActivityRepository) CreateWithStock(context.Context, *model.SeckillActivity, *model.SeckillStock) error {
	r.created = true
	return nil
}

func (r *fakeActivityRepository) List(context.Context) ([]repository.ActivityView, error) {
	return nil, nil
}

func (r *fakeActivityRepository) GetByID(context.Context, uint64) (*repository.ActivityView, error) {
	return nil, nil
}

func TestActivityServiceRejectsInvalidTimeWindow(t *testing.T) {
	service := NewActivityService(
		&fakeProductRepository{products: map[uint64]*model.Product{1: {ID: 1}}},
		&fakeActivityRepository{},
		nil,
	)

	now := time.Now()
	err := service.Create(context.Background(), CreateActivityInput{
		ProductID:  1,
		Name:       "test",
		StartTime:  now,
		EndTime:    now,
		TotalStock: 10,
	})
	if err != ErrInvalidActivityTime {
		t.Fatalf("expected ErrInvalidActivityTime, got %v", err)
	}
}

func TestActivityServiceRejectsMissingProduct(t *testing.T) {
	service := NewActivityService(
		&fakeProductRepository{products: map[uint64]*model.Product{}},
		&fakeActivityRepository{},
		nil,
	)

	err := service.Create(context.Background(), CreateActivityInput{
		ProductID:  99,
		Name:       "test",
		StartTime:  time.Now(),
		EndTime:    time.Now().Add(time.Hour),
		TotalStock: 10,
	})
	if err != ErrProductMissing {
		t.Fatalf("expected ErrProductMissing, got %v", err)
	}
}
