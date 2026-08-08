// Command notification-consumer delivers RocketMQ scheduled notifications:
// postsvc publishes a delayed ScheduledNotification after each post is
// created (10s delay level), and this consumer "delivers" it once
// RocketMQ releases the message — standing in for a real notification
// channel (email/push/webhook). RocketMQ is the right fit because delayed
// delivery and consumer-group ordering are native features here, not
// something bolted on with a separate scheduler.
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
	"Post_Analyzer_Webserver/internal/messaging/rocketmq"
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
	if !cfg.Messaging.RocketMQEnabled {
		logger.Error("ROCKETMQ_ENABLED=false — notification-consumer has nothing to consume, exiting")
		os.Exit(1)
	}

	metricsPort := os.Getenv("METRICS_PORT")
	if metricsPort == "" {
		metricsPort = "9103"
	}
	metrics.Serve("notification-consumer", metricsPort)

	consumer, err := rocketmq.NewPushConsumer(cfg.Messaging.RocketMQNsAddrs, "notification-consumer")
	if err != nil {
		logger.Error("failed to create rocketmq consumer", "error", err)
		os.Exit(1)
	}

	err = consumer.Subscribe(rocketmq.NotificationsTopic, func(ctx context.Context, body []byte) error {
		var notif eventpb.ScheduledNotification
		if err := proto.Unmarshal(body, &notif); err != nil {
			logger.Error("failed to unmarshal scheduled notification", "error", err)
			return nil
		}
		metrics.RecordNotificationDelivered(notif.Kind)
		logger.Info("notification delivered",
			"notification_id", notif.NotificationId, "kind", notif.Kind,
			"post_id", notif.PostId, "message", notif.Message,
		)
		return nil
	})
	if err != nil {
		logger.Error("failed to subscribe to rocketmq topic", "error", err)
		os.Exit(1)
	}

	if err := consumer.Start(); err != nil {
		logger.Error("failed to start rocketmq consumer", "error", err)
		os.Exit(1)
	}

	logger.Info("notification-consumer starting",
		"name_servers", cfg.Messaging.RocketMQNsAddrs,
		"topic", rocketmq.NotificationsTopic,
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	if err := consumer.Shutdown(); err != nil {
		logger.Error("error shutting down rocketmq consumer", "error", err)
	}
	logger.Info("notification-consumer stopped gracefully")
}
