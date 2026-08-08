// Command reanalysis-worker is the competing-consumer side of the
// RabbitMQ reanalysis queue: the gateway's POST /api/v1/posts/reanalyze
// enqueues a ReanalysisJob and returns immediately; this worker pulls
// jobs off reanalysis.jobs and runs the actual analysis against postsvc.
// RabbitMQ's ack/requeue-on-failure semantics are what make it a better
// fit than Kafka here — a failed job should be retried by *some* worker,
// not replayed to every consumer of a log.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"Post_Analyzer_Webserver/config"
	"Post_Analyzer_Webserver/internal/gen/eventpb"
	"Post_Analyzer_Webserver/internal/logger"
	"Post_Analyzer_Webserver/internal/messaging/rabbitmq"
	"Post_Analyzer_Webserver/internal/metrics"
	"Post_Analyzer_Webserver/internal/rpcclient"

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
	if !cfg.Messaging.RabbitMQEnabled {
		logger.Error("RABBITMQ_ENABLED=false — reanalysis-worker has nothing to consume, exiting")
		os.Exit(1)
	}

	metricsPort := os.Getenv("METRICS_PORT")
	if metricsPort == "" {
		metricsPort = "9102"
	}
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", metrics.Handler())
		logger.Info("reanalysis-worker metrics listening", "port", metricsPort)
		if err := http.ListenAndServe(":"+metricsPort, mux); err != nil { //nolint:gosec // internal metrics endpoint
			logger.Error("metrics server failed", "error", err)
		}
	}()

	postClient, err := rpcclient.NewPostClient(cfg.RPC.PostServiceAddr, cfg.RPC.MuxTransport)
	if err != nil {
		logger.Error("failed to create postsvc RPC client", "error", err)
		os.Exit(1)
	}

	mq, err := rabbitmq.Connect(cfg.Messaging.RabbitMQURL)
	if err != nil {
		logger.Error("failed to connect to rabbitmq", "error", err)
		os.Exit(1)
	}
	defer mq.Close()
	if err := mq.DeclareQueue(rabbitmq.ReanalysisQueue); err != nil {
		logger.Error("failed to declare rabbitmq queue", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Info("reanalysis-worker starting", "queue", rabbitmq.ReanalysisQueue)

	err = mq.Consume(ctx, rabbitmq.ReanalysisQueue, "reanalysis-worker", func(ctx context.Context, body []byte) error {
		var job eventpb.ReanalysisJob
		if err := proto.Unmarshal(body, &job); err != nil {
			logger.Error("failed to unmarshal reanalysis job", "error", err)
			metrics.RecordReanalysisJobProcessed("bad_message")
			return nil // don't requeue a message we can never parse
		}

		start := time.Now()
		result, err := postClient.AnalyzeCharacterFrequency(ctx)
		if err != nil {
			metrics.RecordReanalysisJobProcessed("error")
			logger.Error("reanalysis job failed", "job_id", job.JobId, "error", err)
			return err // nack + requeue, postsvc might just be restarting
		}

		metrics.RecordReanalysisJobProcessed("success")
		logger.Info("reanalysis job completed",
			"job_id", job.JobId, "requested_by", job.RequestedBy,
			"total_posts", result.TotalPosts, "duration_ms", time.Since(start).Milliseconds(),
		)
		return nil
	})
	if err != nil {
		logger.Error("reanalysis-worker stopped with error", "error", err)
		os.Exit(1)
	}

	logger.Info("reanalysis-worker stopped gracefully")
}
