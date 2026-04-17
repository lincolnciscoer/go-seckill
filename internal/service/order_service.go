package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"go-seckill/internal/cache"
	"go-seckill/internal/model"
	"go-seckill/internal/mq"
	"go-seckill/internal/repository"
)

var (
	ErrOrderNotFound      = errors.New("order not found")
	ErrActivityNotStarted = errors.New("activity not started")
	ErrActivityEnded      = errors.New("activity ended")
	ErrActivityInactive   = errors.New("activity inactive")
	ErrSoldOut            = errors.New("sold out")
	ErrRepeatOrder        = errors.New("repeat order")
)

type OrderService struct {
	orders repository.OrderRepository
}

type SeckillService struct {
	products   repository.ProductRepository
	activities repository.ActivityRepository
	orders     repository.OrderRepository
	cache      *cache.ActivityCache
	producer   mq.SeckillOrderProducer
}

type SeckillAcceptedResult struct {
	OrderNo   string    `json:"order_no"`
	MessageID string    `json:"message_id"`
	Status    string    `json:"status"`
	QueuedAt  time.Time `json:"queued_at"`
}

func NewOrderService(orders repository.OrderRepository) *OrderService {
	return &OrderService{orders: orders}
}

func NewSeckillService(
	products repository.ProductRepository,
	activities repository.ActivityRepository,
	orders repository.OrderRepository,
	activityCache *cache.ActivityCache,
	producer mq.SeckillOrderProducer,
) *SeckillService {
	return &SeckillService{
		products:   products,
		activities: activities,
		orders:     orders,
		cache:      activityCache,
		producer:   producer,
	}
}

func (s *OrderService) GetByOrderNo(ctx context.Context, orderNo string) (*model.Order, error) {
	order, err := s.orders.GetByOrderNo(ctx, orderNo)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}

	return order, nil
}

func (s *OrderService) ListByUserID(ctx context.Context, userID uint64) ([]model.Order, error) {
	return s.orders.ListByUserID(ctx, userID)
}

// Attempt 在数据库层直接完成库存扣减和订单创建。
// 这是同步版秒杀闭环：简单、易理解，但高并发下数据库压力会较大，后面我们会用 Redis+Lua+MQ 继续演进。
func (s *SeckillService) Attempt(ctx context.Context, userID uint64, activityID uint64) (*SeckillAcceptedResult, error) {
	activity, err := s.loadActivity(ctx, activityID)
	if err != nil {
		return nil, err
	}
	if activity == nil {
		return nil, ErrActivityNotFound
	}

	now := time.Now()
	if activity.Status != 1 {
		return nil, ErrActivityInactive
	}
	if now.Before(activity.StartTime) {
		return nil, ErrActivityNotStarted
	}
	if now.After(activity.EndTime) {
		return nil, ErrActivityEnded
	}

	existingOrder, err := s.orders.GetByUserActivity(ctx, userID, activity.ID)
	if err != nil {
		return nil, err
	}
	if existingOrder != nil {
		return nil, ErrRepeatOrder
	}

	if s.cache != nil {
		attemptResult, err := s.attemptWithRedis(ctx, activity, userID)
		if err != nil {
			return nil, err
		}

		switch attemptResult.Code {
		case cache.SeckillAllowed:
			// Redis 预扣成功，继续走数据库下单。
		case cache.SeckillSoldOut:
			return nil, ErrSoldOut
		case cache.SeckillRepeatOrder:
			return nil, ErrRepeatOrder
		case cache.SeckillActivityClosed:
			return nil, ErrActivityInactive
		case cache.SeckillNotStarted:
			return nil, ErrActivityNotStarted
		case cache.SeckillEnded:
			return nil, ErrActivityEnded
		default:
			return nil, fmt.Errorf("unexpected redis seckill code: %d", attemptResult.Code)
		}
	} else if activity.AvailableStock <= 0 {
		return nil, ErrSoldOut
	}

	product, err := s.products.GetByID(ctx, activity.ProductID)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrProductMissing
	}

	if s.producer == nil {
		return nil, fmt.Errorf("seckill order producer is not configured")
	}

	orderNo := generateOrderNo()
	messageID, err := s.producer.SendSeckillOrder(ctx, &mq.SeckillOrderMessage{
		OrderNo:    orderNo,
		UserID:     userID,
		ActivityID: activity.ID,
		ProductID:  product.ID,
		Quantity:   1,
		Amount:     product.Price,
		Status:     model.OrderStatusCreated,
		CreatedAt:  time.Now(),
	})
	if err != nil {
		if s.cache != nil {
			_ = s.cache.CompensateSeckill(ctx, activity.ID, userID)
		}
		return nil, err
	}

	return &SeckillAcceptedResult{
		OrderNo:   orderNo,
		MessageID: messageID,
		Status:    "queued",
		QueuedAt:  time.Now(),
	}, nil
}

func (s *SeckillService) loadActivity(ctx context.Context, activityID uint64) (*repository.ActivityView, error) {
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
		return nil, nil
	}

	if s.cache != nil {
		_ = s.cache.SetActivityDetail(ctx, *activity)
	}

	return activity, nil
}

func (s *SeckillService) attemptWithRedis(ctx context.Context, activity *repository.ActivityView, userID uint64) (*cache.SeckillAttemptResult, error) {
	if s.cache == nil {
		return &cache.SeckillAttemptResult{Code: cache.SeckillAllowed}, nil
	}

	result, err := s.cache.AttemptSeckill(ctx, cache.SeckillAttemptInput{
		ActivityID: activity.ID,
		UserID:     userID,
		Status:     activity.Status,
		Now:        time.Now(),
		StartTime:  activity.StartTime,
		EndTime:    activity.EndTime,
	})
	if err != nil {
		return nil, err
	}

	if result.Code == cache.SeckillNotPreheated {
		if err := s.cache.SetActivityDetail(ctx, *activity); err != nil {
			return nil, err
		}

		return s.cache.AttemptSeckill(ctx, cache.SeckillAttemptInput{
			ActivityID: activity.ID,
			UserID:     userID,
			Status:     activity.Status,
			Now:        time.Now(),
			StartTime:  activity.StartTime,
			EndTime:    activity.EndTime,
		})
	}

	return result, nil
}

func generateOrderNo() string {
	return fmt.Sprintf("SK%s%04d", time.Now().Format("20060102150405"), rand.IntN(10000))
}

type AsyncOrderService struct {
	orders repository.OrderRepository
	cache  *cache.ActivityCache
}

func NewAsyncOrderService(orders repository.OrderRepository, activityCache *cache.ActivityCache) *AsyncOrderService {
	return &AsyncOrderService{
		orders: orders,
		cache:  activityCache,
	}
}

func (s *AsyncOrderService) HandleSeckillOrder(ctx context.Context, messageID string, payload *mq.SeckillOrderMessage) error {
	_, err := s.orders.CreateSeckillOrder(ctx, repository.CreateSeckillOrderInput{
		MessageID:  messageID,
		OrderNo:    payload.OrderNo,
		UserID:     payload.UserID,
		ActivityID: payload.ActivityID,
		ProductID:  payload.ProductID,
		Quantity:   payload.Quantity,
		Amount:     payload.Amount,
		Status:     payload.Status,
	})
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrDuplicateConsume):
			return nil
		case errors.Is(err, repository.ErrDuplicateOrder):
			return nil
		default:
			return err
		}
	}

	if s.cache != nil {
		_ = s.cache.InvalidateActivityViews(ctx, payload.ActivityID)
	}

	return nil
}
