// Package kafka wraps segmentio/kafka-go for the post-events analytics
// stream: postsvc produces a PostEvent (protobuf, idl/proto/events.proto)
// on every create/update/delete/analyze; cmd/analytics-consumer consumes
// the topic to maintain live aggregate counters. Kafka is the right fit
// here specifically because this is a durable, replayable, multi-consumer
// event log — not a work queue — which is what an analytics pipeline needs.
package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

const PostEventsTopic = "post.events"

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			BatchTimeout: 50 * time.Millisecond,
			WriteTimeout: 5 * time.Second,
		},
	}
}

// Publish marshals msg as protobuf and writes it, keyed by key (e.g. the
// post ID, so all events for one post land on the same partition and stay
// ordered relative to each other).
func (p *Producer) Publish(ctx context.Context, key string, msg proto.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal kafka message: %w", err)
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: data,
		Time:  time.Now(),
	})
}

func (p *Producer) Close() error {
	return p.writer.Close()
}

type Consumer struct {
	reader *kafka.Reader
}

func NewConsumer(brokers []string, topic, groupID string) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:  brokers,
			Topic:    topic,
			GroupID:  groupID,
			MinBytes: 1,
			MaxBytes: 10e6,
		}),
	}
}

// Consume blocks, invoking handler for each message until ctx is
// cancelled or handler/read returns a non-recoverable error.
func (c *Consumer) Consume(ctx context.Context, handler func(ctx context.Context, key, value []byte) error) error {
	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("fetch kafka message: %w", err)
		}
		if err := handler(ctx, m.Key, m.Value); err != nil {
			return err
		}
		if err := c.reader.CommitMessages(ctx, m); err != nil {
			return fmt.Errorf("commit kafka message: %w", err)
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
