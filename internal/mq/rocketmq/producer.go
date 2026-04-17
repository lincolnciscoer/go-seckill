package rocketmq

import (
	"context"
	"encoding/json"
	"fmt"

	rmqclient "github.com/apache/rocketmq-clients/golang"
	"github.com/apache/rocketmq-clients/golang/credentials"

	"go-seckill/internal/config"
	"go-seckill/internal/mq"
)

type Producer struct {
	producer rmqclient.Producer
	topic    string
}

func NewProducer(cfg config.RocketMQConfig) (*Producer, error) {
	producer, err := rmqclient.NewProducer(&rmqclient.Config{
		Endpoint: cfg.Endpoint,
		Credentials: &credentials.SessionCredentials{
			AccessKey:    cfg.AccessKey,
			AccessSecret: cfg.AccessSecret,
		},
	}, rmqclient.WithTopics(cfg.Topic))
	if err != nil {
		return nil, err
	}

	if err := producer.Start(); err != nil {
		return nil, err
	}

	return &Producer{
		producer: producer,
		topic:    cfg.Topic,
	}, nil
}

func (p *Producer) SendSeckillOrder(ctx context.Context, message *mq.SeckillOrderMessage) (string, error) {
	payload, err := json.Marshal(message)
	if err != nil {
		return "", err
	}

	msg := &rmqclient.Message{
		Topic: p.topic,
		Body:  payload,
	}
	msg.SetKeys(message.OrderNo)
	msg.SetTag(mq.SeckillOrderTopicTag)

	receipts, err := p.producer.Send(ctx, msg)
	if err != nil {
		return "", err
	}
	if len(receipts) == 0 {
		return "", fmt.Errorf("rocketmq returned empty send receipts")
	}

	return receipts[0].MessageID, nil
}

func (p *Producer) Shutdown() error {
	if p == nil || p.producer == nil {
		return nil
	}

	return p.producer.GracefulStop()
}
