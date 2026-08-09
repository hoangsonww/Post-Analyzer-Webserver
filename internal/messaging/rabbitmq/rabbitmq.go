// Package rabbitmq wraps amqp091-go for the reanalysis work queue: the
// gateway enqueues a ReanalysisJob (protobuf, idl/proto/events.proto) when
// a client asks for an on-demand reanalysis, and cmd/reanalysis-worker
// pulls jobs off the queue and runs them against postsvc. RabbitMQ is the
// right fit here — unlike the Kafka event log, this is classic
// competing-consumers task-queue semantics: each job must be handled by
// exactly one worker, with ack/requeue on failure.
package rabbitmq

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/protobuf/proto"
)

const ReanalysisQueue = "reanalysis.jobs"

type Client struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func Connect(url string) (*Client, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("dial rabbitmq: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("open rabbitmq channel: %w", err)
	}
	return &Client{conn: conn, ch: ch}, nil
}

// DeclareQueue declares a durable queue — jobs survive a broker restart,
// matching the "don't silently drop a reanalysis request" requirement.
func (c *Client) DeclareQueue(name string) error {
	_, err := c.ch.QueueDeclare(name, true, false, false, false, nil)
	return err
}

func (c *Client) Publish(ctx context.Context, queue string, msg proto.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal rabbitmq message: %w", err)
	}
	return c.ch.PublishWithContext(ctx, "", queue, false, false, amqp.Publishing{
		ContentType:  "application/x-protobuf",
		Body:         data,
		DeliveryMode: amqp.Persistent,
	})
}

// Consume registers handler as the consumer for queue. It blocks until ctx
// is cancelled. Messages are acked on success and nacked+requeued on
// handler error.
func (c *Client) Consume(ctx context.Context, queue, consumerTag string, handler func(ctx context.Context, body []byte) error) error {
	msgs, err := c.ch.Consume(queue, consumerTag, false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("register rabbitmq consumer: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case d, ok := <-msgs:
			if !ok {
				return fmt.Errorf("rabbitmq delivery channel closed")
			}
			if err := handler(ctx, d.Body); err != nil {
				_ = d.Nack(false, true)
				continue
			}
			_ = d.Ack(false)
		}
	}
}

func (c *Client) Close() error {
	_ = c.ch.Close()
	return c.conn.Close()
}
