package service

import (
	"context"
	"testing"
	"time"

	"go-seckill/internal/model"
	"go-seckill/internal/mq"
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

type fakeSeckillProducer struct {
	messageID string
	err       error
	lastSent  *mq.SeckillOrderMessage
}

func (p *fakeSeckillProducer) SendSeckillOrder(_ context.Context, message *mq.SeckillOrderMessage) (string, error) {
	p.lastSent = message
	if p.err != nil {
		return "", p.err
	}
	if p.messageID == "" {
		p.messageID = "msg-1"
	}
	return p.messageID, nil
}

func (p *fakeSeckillProducer) Shutdown() error {
	return nil
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
	producer := &fakeSeckillProducer{}
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

	service := NewSeckillService(productRepo, activityRepo, orderRepo, nil, nil, producer)

	result, err := service.Attempt(context.Background(), 99, 10)
	if err != nil {
		t.Fatalf("attempt: %v", err)
	}

	if result.Status != "queued" {
		t.Fatalf("expected queued status, got %q", result.Status)
	}

	if producer.lastSent == nil || producer.lastSent.UserID != 99 {
		t.Fatalf("expected producer to receive user id 99, got %#v", producer.lastSent)
	}

	if producer.lastSent.Amount != 9999 {
		t.Fatalf("expected amount 9999, got %d", producer.lastSent.Amount)
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
		&fakeOrderRepository{
			order: &model.Order{
				OrderNo:    "SK-repeat",
				UserID:     1,
				ActivityID: 10,
			},
		},
		nil,
		nil,
		&fakeSeckillProducer{},
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
		nil,
		&fakeSeckillProducer{},
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
		nil,
		&fakeSeckillProducer{},
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
		nil,
		&fakeSeckillProducer{},
	)

	_, err := service.Attempt(context.Background(), 1, 10)
	if err != ErrSoldOut {
		t.Fatalf("expected ErrSoldOut, got %v", err)
	}
}

func TestAsyncOrderServiceHandlesDuplicateConsume(t *testing.T) {
	service := NewAsyncOrderService(&fakeOrderRepository{createErr: repository.ErrDuplicateConsume}, nil, nil)

	err := service.HandleSeckillOrder(context.Background(), "msg-1", &mq.SeckillOrderMessage{
		OrderNo:    "SK1",
		UserID:     1,
		ActivityID: 10,
		ProductID:  11,
		Quantity:   1,
		Amount:     100,
		Status:     model.OrderStatusCreated,
	})
	if err != nil {
		t.Fatalf("expected nil error on duplicate consume, got %v", err)
	}
}

func TestOrderServiceGetByOrderNo(t *testing.T) {
	service := NewOrderService(&fakeOrderRepository{
		order: &model.Order{OrderNo: "SK1"},
	}, nil)

	order, err := service.GetByOrderNo(context.Background(), "SK1")
	if err != nil {
		t.Fatalf("get by order no: %v", err)
	}
	if order.OrderNo != "SK1" {
		t.Fatalf("expected order no SK1, got %s", order.OrderNo)
	}
}
