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
	"go-seckill/internal/observability"
	"go-seckill/internal/repository"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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
	orders      repository.OrderRepository
	statusCache *cache.OrderStatusCache
}

type SeckillService struct {
	products    repository.ProductRepository
	activities  repository.ActivityRepository
	orders      repository.OrderRepository
	cache       *cache.ActivityCache
	statusCache *cache.OrderStatusCache
	producer    mq.SeckillOrderProducer
}

type SeckillAcceptedResult struct {
	OrderNo   string    `json:"order_no"`
	MessageID string    `json:"message_id"`
	Status    string    `json:"status"`
	QueuedAt  time.Time `json:"queued_at"`
}

func NewOrderService(orders repository.OrderRepository, statusCache *cache.OrderStatusCache) *OrderService {
	return &OrderService{
		orders:      orders,
		statusCache: statusCache,
	}
}

func NewSeckillService(
	products repository.ProductRepository,
	activities repository.ActivityRepository,
	orders repository.OrderRepository,
	activityCache *cache.ActivityCache,
	statusCache *cache.OrderStatusCache,
	producer mq.SeckillOrderProducer,
) *SeckillService {
	return &SeckillService{
		products:    products,
		activities:  activities,
		orders:      orders,
		cache:       activityCache,
		statusCache: statusCache,
		producer:    producer,
	}
}

func (s *OrderService) GetByOrderNo(ctx context.Context, orderNo string) (*model.Order, error) {
	ctx, span := observability.Tracer("go-seckill/order").Start(ctx, "order.get_by_order_no")
	defer span.End()
	span.SetAttributes(attribute.String("order.no", orderNo))

	order, err := s.orders.GetByOrderNo(ctx, orderNo)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if order == nil {
		span.SetStatus(codes.Error, ErrOrderNotFound.Error())
		return nil, ErrOrderNotFound
	}

	if s.statusCache != nil {
		_ = s.statusCache.Set(ctx, cache.OrderStatusPayload{
			OrderNo:    order.OrderNo,
			UserID:     order.UserID,
			ActivityID: order.ActivityID,
			Status:     "created",
			UpdatedAt:  time.Now(),
		})
	}
	observability.RecordOrderStatus("created")

	return order, nil
}

func (s *OrderService) ListByUserID(ctx context.Context, userID uint64) ([]model.Order, error) {
	return s.orders.ListByUserID(ctx, userID)
}

func (s *OrderService) GetStatus(ctx context.Context, orderNo string) (*cache.OrderStatusPayload, error) {
	if s.statusCache == nil {
		return nil, ErrOrderNotFound
	}

	status, hit, err := s.statusCache.Get(ctx, orderNo)
	if err != nil {
		return nil, err
	}
	if !hit {
		return nil, ErrOrderNotFound
	}

	return status, nil
}

// Attempt 在数据库层直接完成库存扣减和订单创建。
// 这是同步版秒杀闭环：简单、易理解，但高并发下数据库压力会较大，后面我们会用 Redis+Lua+MQ 继续演进。
func (s *SeckillService) Attempt(ctx context.Context, userID uint64, activityID uint64) (*SeckillAcceptedResult, error) {
	ctx, span := observability.Tracer("go-seckill/seckill").Start(ctx, "seckill.attempt")
	defer span.End()
	span.SetAttributes(
		attribute.Int64("activity.id", int64(activityID)),
		attribute.Int64("user.id", int64(userID)),
	)

	resultLabel := "unknown"
	defer func() {
		observability.RecordSeckillAttempt(resultLabel)
	}()

	activity, err := s.loadActivity(ctx, activityID)
	if err != nil {
		resultLabel = "load_error"
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if activity == nil {
		resultLabel = "activity_not_found"
		span.SetStatus(codes.Error, ErrActivityNotFound.Error())
		return nil, ErrActivityNotFound
	}

	now := time.Now()
	if activity.Status != 1 {
		resultLabel = "inactive"
		return nil, ErrActivityInactive
	}
	if now.Before(activity.StartTime) {
		resultLabel = "not_started"
		return nil, ErrActivityNotStarted
	}
	if now.After(activity.EndTime) {
		resultLabel = "ended"
		return nil, ErrActivityEnded
	}

	existingOrder, err := s.orders.GetByUserActivity(ctx, userID, activity.ID)
	if err != nil {
		resultLabel = "order_lookup_error"
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if existingOrder != nil {
		resultLabel = "repeat"
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
			resultLabel = "sold_out"
			return nil, ErrSoldOut
		case cache.SeckillRepeatOrder:
			resultLabel = "repeat"
			return nil, ErrRepeatOrder
		case cache.SeckillActivityClosed:
			resultLabel = "inactive"
			return nil, ErrActivityInactive
		case cache.SeckillNotStarted:
			resultLabel = "not_started"
			return nil, ErrActivityNotStarted
		case cache.SeckillEnded:
			resultLabel = "ended"
			return nil, ErrActivityEnded
		default:
			resultLabel = "lua_error"
			return nil, fmt.Errorf("unexpected redis seckill code: %d", attemptResult.Code)
		}
	} else if activity.AvailableStock <= 0 {
		resultLabel = "sold_out"
		return nil, ErrSoldOut
	}

	product, err := s.products.GetByID(ctx, activity.ProductID)
	if err != nil {
		resultLabel = "product_lookup_error"
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if product == nil {
		resultLabel = "product_missing"
		return nil, ErrProductMissing
	}

	if s.producer == nil {
		resultLabel = "producer_missing"
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
		resultLabel = "enqueue_error"
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	resultLabel = "queued"

	return &SeckillAcceptedResult{
		OrderNo:   orderNo,
		MessageID: messageID,
		Status:    "queued",
		QueuedAt:  time.Now(),
	}, s.markQueuedStatus(ctx, orderNo, userID, activity.ID)
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
	orders      repository.OrderRepository
	cache       *cache.ActivityCache
	statusCache *cache.OrderStatusCache
}

func NewAsyncOrderService(
	orders repository.OrderRepository,
	activityCache *cache.ActivityCache,
	statusCache *cache.OrderStatusCache,
) *AsyncOrderService {
	return &AsyncOrderService{
		orders:      orders,
		cache:       activityCache,
		statusCache: statusCache,
	}
}

func (s *AsyncOrderService) HandleSeckillOrder(ctx context.Context, messageID string, payload *mq.SeckillOrderMessage) error {
	ctx, span := observability.Tracer("go-seckill/consumer").Start(ctx, "mq.handle_seckill_order")
	defer span.End()
	span.SetAttributes(
		attribute.String("message.id", messageID),
		attribute.String("order.no", payload.OrderNo),
		attribute.Int64("activity.id", int64(payload.ActivityID)),
		attribute.Int64("user.id", int64(payload.UserID)),
	)

	resultLabel := "success"
	defer func() {
		observability.RecordMQConsume(resultLabel)
	}()

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
			resultLabel = "duplicate_consume"
			return nil
		case errors.Is(err, repository.ErrDuplicateOrder):
			resultLabel = "duplicate_order"
			return nil
		default:
			resultLabel = "error"
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}

	if s.cache != nil {
		_ = s.cache.InvalidateActivityViews(ctx, payload.ActivityID)
	}

	if s.statusCache != nil {
		_ = s.statusCache.Set(ctx, cache.OrderStatusPayload{
			OrderNo:    payload.OrderNo,
			UserID:     payload.UserID,
			ActivityID: payload.ActivityID,
			Status:     "created",
			UpdatedAt:  time.Now(),
		})
	}
	observability.RecordOrderStatus("created")

	return nil
}

func (s *SeckillService) markQueuedStatus(ctx context.Context, orderNo string, userID uint64, activityID uint64) error {
	if s.statusCache == nil {
		return nil
	}

	observability.RecordOrderStatus("queued")
	return s.statusCache.Set(ctx, cache.OrderStatusPayload{
		OrderNo:    orderNo,
		UserID:     userID,
		ActivityID: activityID,
		Status:     "queued",
		UpdatedAt:  time.Now(),
	})
}
