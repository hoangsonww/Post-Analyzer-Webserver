// Command analytics-consumer is the streaming-analytics side of the
// Kafka post-events pipeline: it subscribes to post.events (published by
// postsvc on every create/update/delete/analyze) and maintains live
// Prometheus counters per event type, exposed on its own /metrics
// endpoint. Kafka's durable, replayable, multi-consumer log is what makes
// this a good fit — the same topic could feed other consumers (e.g. a
// data warehouse loader) without postsvc knowing or caring.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"Post_Analyzer_Webserver/config"
	"Post_Analyzer_Webserver/internal/gen/eventpb"
	"Post_Analyzer_Webserver/internal/logger"
	"Post_Analyzer_Webserver/internal/messaging/kafka"
	"Post_Analyzer_Webserver/internal/metrics"

	"google.golang.org/protobuf/proto"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load configuration: %v\n", err)
		os.Exit(1)
	}
	if err := logger.Init(&cfg.Logging); err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	if !cfg.Messaging.KafkaEnabled {
		logger.Error("KAFKA_ENABLED=false — analytics-consumer has nothing to consume, exiting")
		os.Exit(1)
	}

	metricsPort := os.Getenv("METRICS_PORT")
	if metricsPort == "" {
		metricsPort = "9101"
	}
	metrics.Serve("analytics-consumer", metricsPort)

	consumer := kafka.NewConsumer(cfg.Messaging.KafkaBrokers, kafka.PostEventsTopic, "analytics-consumer")
	defer func() { _ = consumer.Close() }()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Info("analytics-consumer starting",
		"brokers", cfg.Messaging.KafkaBrokers,
		"topic", kafka.PostEventsTopic,
		"group", "analytics-consumer",
	)

	err = consumer.Consume(ctx, func(ctx context.Context, key, value []byte) error {
		var evt eventpb.PostEvent
		if err := proto.Unmarshal(value, &evt); err != nil {
			logger.Error("failed to unmarshal post event", "error", err)
			return nil // skip poison message rather than blocking the partition
		}
		metrics.RecordPostEventConsumed(evt.EventType)
		logger.Info("post event consumed",
			"event_id", evt.EventId, "event_type", evt.EventType,
			"post_id", evt.PostId, "user_id", evt.UserId, "title", evt.Title,
		)
		return nil
	})
	if err != nil {
		logger.Error("analytics-consumer stopped with error", "error", err)
		os.Exit(1)
	}

	logger.Info("analytics-consumer stopped gracefully")
}
