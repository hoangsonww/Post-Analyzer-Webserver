// Package rocketmq wraps apache/rocketmq-client-go for scheduled
// re-checks and transactional-style notifications: postsvc publishes a
// ScheduledNotification (protobuf, idl/proto/events.proto) with a delay
// level after a post is created, and cmd/notification-consumer delivers
// it once the delay elapses. RocketMQ is the right fit here because it
// natively supports scheduled/delayed delivery and strict per-topic
// ordering, which Kafka and RabbitMQ don't offer as first-class features.
package rocketmq

import (
	"context"
	"fmt"

	rmq "github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"google.golang.org/protobuf/proto"
)

// NotificationsTopic intentionally has no dots: RocketMQ topic names are
// restricted to ^[%|a-zA-Z0-9_-]+$.
const NotificationsTopic = "post-notifications"

// DelayLevelTenSeconds is RocketMQ's built-in delay level 3 (10s), used
// for the "scheduled recheck" notification fired after post creation.
// RocketMQ's default levels are 1s 5s 10s 30s 1m 2m 3m 4m 5m 6m 7m 8m 9m
// 10m 20m 30m 1h 2h — level N maps to the Nth entry in that list.
const DelayLevelTenSeconds = 3

type Producer struct {
	p rmq.Producer
}

func NewProducer(nameServers []string) (*Producer, error) {
	p, err := rmq.NewProducer(
		producer.WithNsResolver(primitive.NewPassthroughResolver(nameServers)),
		producer.WithRetry(2),
	)
	if err != nil {
		return nil, fmt.Errorf("create rocketmq producer: %w", err)
	}
	if err := p.Start(); err != nil {
		return nil, fmt.Errorf("start rocketmq producer: %w", err)
	}
	return &Producer{p: p}, nil
}

// PublishDelayed marshals msg as protobuf and sends it to topic with the
// given RocketMQ delay level (0 = no delay).
func (pr *Producer) PublishDelayed(ctx context.Context, topic string, msg proto.Message, delayLevel int) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal rocketmq message: %w", err)
	}
	m := primitive.NewMessage(topic, data)
	if delayLevel > 0 {
		m.WithDelayTimeLevel(delayLevel)
	}
	_, err = pr.p.SendSync(ctx, m)
	return err
}

func (pr *Producer) Close() error {
	return pr.p.Shutdown()
}

type PushConsumer struct {
	c rmq.PushConsumer
}

func NewPushConsumer(nameServers []string, group string) (*PushConsumer, error) {
	c, err := rmq.NewPushConsumer(
		consumer.WithGroupName(group),
		consumer.WithNsResolver(primitive.NewPassthroughResolver(nameServers)),
		consumer.WithConsumerModel(consumer.Clustering),
	)
	if err != nil {
		return nil, fmt.Errorf("create rocketmq consumer: %w", err)
	}
	return &PushConsumer{c: c}, nil
}

// Subscribe registers handler for topic. Must be called before Start.
func (pc *PushConsumer) Subscribe(topic string, handler func(ctx context.Context, body []byte) error) error {
	return pc.c.Subscribe(topic, consumer.MessageSelector{}, func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
		for _, m := range msgs {
			if err := handler(ctx, m.Body); err != nil {
				return consumer.ConsumeRetryLater, err
			}
		}
		return consumer.ConsumeSuccess, nil
	})
}

func (pc *PushConsumer) Start() error {
	return pc.c.Start()
}

func (pc *PushConsumer) Shutdown() error {
	return pc.c.Shutdown()
}
