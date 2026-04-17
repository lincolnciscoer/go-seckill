package rocketmq

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	rmqclient "github.com/apache/rocketmq-clients/golang"
	"github.com/apache/rocketmq-clients/golang/credentials"
	"go.uber.org/zap"

	"go-seckill/internal/config"
	"go-seckill/internal/mq"
)

type SeckillOrderHandler interface {
	HandleSeckillOrder(ctx context.Context, messageID string, payload *mq.SeckillOrderMessage) error
}

type Consumer struct {
	consumer rmqclient.SimpleConsumer
	config   config.RocketMQConfig
	logger   *zap.Logger
	handler  SeckillOrderHandler
}

func NewConsumer(cfg config.RocketMQConfig, logger *zap.Logger, handler SeckillOrderHandler) (*Consumer, error) {
	consumer, err := rmqclient.NewSimpleConsumer(&rmqclient.Config{
		Endpoint:      cfg.Endpoint,
		ConsumerGroup: cfg.ConsumerGroup,
		Credentials: &credentials.SessionCredentials{
			AccessKey:    cfg.AccessKey,
			AccessSecret: cfg.AccessSecret,
		},
	},
		rmqclient.WithAwaitDuration(cfg.AwaitDuration),
		rmqclient.WithSubscriptionExpressions(map[string]*rmqclient.FilterExpression{
			cfg.Topic: rmqclient.SUB_ALL,
		}),
	)
	if err != nil {
		return nil, err
	}

	if err := consumer.Start(); err != nil {
		return nil, err
	}

	return &Consumer{
		consumer: consumer,
		config:   cfg,
		logger:   logger,
		handler:  handler,
	}, nil
}

func (c *Consumer) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		messages, err := c.consumer.Receive(ctx, c.config.MaxMessageNum, c.config.InvisibleDuration)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			switch {
			case strings.Contains(err.Error(), "MESSAGE_NOT_FOUND"):
				time.Sleep(time.Second)
				continue
			case strings.Contains(err.Error(), "NullPointerException"):
				c.logger.Warn("rocketmq proxy is still warming up", zap.Error(err))
			default:
				c.logger.Warn("rocketmq receive failed", zap.Error(err))
			}
			time.Sleep(time.Second)
			continue
		}

		for _, messageView := range messages {
			if err := c.handleMessage(ctx, messageView); err != nil {
				c.logger.Error("rocketmq consume failed",
					zap.String("message_id", messageView.GetMessageId()),
					zap.Error(err),
				)
				continue
			}

			if err := c.consumer.Ack(ctx, messageView); err != nil {
				c.logger.Error("rocketmq ack failed",
					zap.String("message_id", messageView.GetMessageId()),
					zap.Error(err),
				)
			}
		}
	}
}

func (c *Consumer) Shutdown() error {
	if c == nil || c.consumer == nil {
		return nil
	}

	return c.consumer.GracefulStop()
}

func (c *Consumer) handleMessage(ctx context.Context, messageView *rmqclient.MessageView) error {
	var payload mq.SeckillOrderMessage
	if err := json.Unmarshal(messageView.GetBody(), &payload); err != nil {
		return fmt.Errorf("decode message body: %w", err)
	}

	return c.handler.HandleSeckillOrder(ctx, messageView.GetMessageId(), &payload)
}
