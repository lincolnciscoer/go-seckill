package service

import (
	"context"
	"testing"
	"time"

	"go-seckill/internal/model"
	"go-seckill/internal/repository"
)

type fakeOrderRepository struct {
	order           *model.Order
	createErr       error
	listOrders      []model.Order
	lastCreateInput repository.CreateSeckillOrderInput
}

func (r *fakeOrderRepository) CreateSeckillOrder(_ context.Context, input repository.CreateSeckillOrderInput) (*model.Order, error) {
	r.lastCreateInput = input
	if r.createErr != nil {
		return nil, r.createErr
	}
	if r.order != nil {
		return r.order, nil
	}

	return &model.Order{
		ID:         1,
		OrderNo:    input.OrderNo,
		UserID:     input.UserID,
		ActivityID: input.ActivityID,
		ProductID:  input.ProductID,
		Quantity:   input.Quantity,
		Amount:     input.Amount,
		Status:     input.Status,
	}, nil
}

func (r *fakeOrderRepository) GetByOrderNo(_ context.Context, orderNo string) (*model.Order, error) {
	if r.order != nil && r.order.OrderNo == orderNo {
		return r.order, nil
	}

	return nil, nil
}

func (r *fakeOrderRepository) GetByUserActivity(_ context.Context, userID uint64, activityID uint64) (*model.Order, error) {
	if r.order != nil && r.order.UserID == userID && r.order.ActivityID == activityID {
		return r.order, nil
	}

	return nil, nil
}

func (r *fakeOrderRepository) ListByUserID(context.Context, uint64) ([]model.Order, error) {
	return r.listOrders, nil
}

type fakeActivityLookupRepository struct {
	activity *repository.ActivityView
}

func (r *fakeActivityLookupRepository) CreateWithStock(context.Context, *model.SeckillActivity, *model.SeckillStock) error {
	return nil
}

func (r *fakeActivityLookupRepository) List(context.Context) ([]repository.ActivityView, error) {
	return nil, nil
}

func (r *fakeActivityLookupRepository) GetByID(context.Context, uint64) (*repository.ActivityView, error) {
	return r.activity, nil
}

func TestSeckillServiceAttemptCreatesOrder(t *testing.T) {
	orderRepo := &fakeOrderRepository{}
	productRepo := &fakeProductRepository{
		products: map[uint64]*model.Product{
			1: {ID: 1, Price: 9999},
		},
	}
	activityRepo := &fakeActivityLookupRepository{
		activity: &repository.ActivityView{
			ID:             10,
			ProductID:      1,
			Status:         1,
			StartTime:      time.Now().Add(-time.Hour),
			EndTime:        time.Now().Add(time.Hour),
			AvailableStock: 5,
		},
	}

	service := NewSeckillService(productRepo, activityRepo, orderRepo, nil)

	order, err := service.Attempt(context.Background(), 99, 10)
	if err != nil {
		t.Fatalf("attempt: %v", err)
	}

	if order.UserID != 99 {
		t.Fatalf("expected user id 99, got %d", order.UserID)
	}

	if orderRepo.lastCreateInput.Amount != 9999 {
		t.Fatalf("expected amount 9999, got %d", orderRepo.lastCreateInput.Amount)
	}
}

func TestSeckillServiceAttemptRejectsRepeatOrder(t *testing.T) {
	service := NewSeckillService(
		&fakeProductRepository{products: map[uint64]*model.Product{1: {ID: 1, Price: 100}}},
		&fakeActivityLookupRepository{
			activity: &repository.ActivityView{
				ID:             10,
				ProductID:      1,
				Status:         1,
				StartTime:      time.Now().Add(-time.Hour),
				EndTime:        time.Now().Add(time.Hour),
				AvailableStock: 5,
			},
		},
		&fakeOrderRepository{createErr: repository.ErrDuplicateOrder},
		nil,
	)

	_, err := service.Attempt(context.Background(), 1, 10)
	if err != ErrRepeatOrder {
		t.Fatalf("expected ErrRepeatOrder, got %v", err)
	}
}

func TestSeckillServiceAttemptRejectsExistingOrderBeforeStockCheck(t *testing.T) {
	service := NewSeckillService(
		&fakeProductRepository{products: map[uint64]*model.Product{1: {ID: 1, Price: 100}}},
		&fakeActivityLookupRepository{
			activity: &repository.ActivityView{
				ID:             10,
				ProductID:      1,
				Status:         1,
				StartTime:      time.Now().Add(-time.Hour),
				EndTime:        time.Now().Add(time.Hour),
				AvailableStock: 0,
			},
		},
		&fakeOrderRepository{
			order: &model.Order{
				OrderNo:    "SK1",
				UserID:     1,
				ActivityID: 10,
			},
		},
		nil,
	)

	_, err := service.Attempt(context.Background(), 1, 10)
	if err != ErrRepeatOrder {
		t.Fatalf("expected ErrRepeatOrder, got %v", err)
	}
}

func TestSeckillServiceAttemptRejectsNotStartedActivity(t *testing.T) {
	service := NewSeckillService(
		&fakeProductRepository{products: map[uint64]*model.Product{1: {ID: 1, Price: 100}}},
		&fakeActivityLookupRepository{
			activity: &repository.ActivityView{
				ID:             10,
				ProductID:      1,
				Status:         1,
				StartTime:      time.Now().Add(time.Hour),
				EndTime:        time.Now().Add(2 * time.Hour),
				AvailableStock: 1,
			},
		},
		&fakeOrderRepository{},
		nil,
	)

	_, err := service.Attempt(context.Background(), 1, 10)
	if err != ErrActivityNotStarted {
		t.Fatalf("expected ErrActivityNotStarted, got %v", err)
	}
}

func TestSeckillServiceAttemptRejectsSoldOut(t *testing.T) {
	service := NewSeckillService(
		&fakeProductRepository{products: map[uint64]*model.Product{1: {ID: 1, Price: 100}}},
		&fakeActivityLookupRepository{
			activity: &repository.ActivityView{
				ID:             10,
				ProductID:      1,
				Status:         1,
				StartTime:      time.Now().Add(-time.Hour),
				EndTime:        time.Now().Add(time.Hour),
				AvailableStock: 0,
			},
		},
		&fakeOrderRepository{},
		nil,
	)

	_, err := service.Attempt(context.Background(), 1, 10)
	if err != ErrSoldOut {
		t.Fatalf("expected ErrSoldOut, got %v", err)
	}
}

func TestOrderServiceGetByOrderNo(t *testing.T) {
	service := NewOrderService(&fakeOrderRepository{
		order: &model.Order{OrderNo: "SK1"},
	})

	order, err := service.GetByOrderNo(context.Background(), "SK1")
	if err != nil {
		t.Fatalf("get by order no: %v", err)
	}
	if order.OrderNo != "SK1" {
		t.Fatalf("expected order no SK1, got %s", order.OrderNo)
	}
}
