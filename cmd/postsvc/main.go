// Command postsvc is the Kitex RPC microservice owning post CRUD and
// character-frequency analysis. It speaks Thrift over TTHeader, uses
// Kitex's default Netpoll network layer, and (when RPC_MUX=true) a
// single multiplexed connection per caller instead of one conn per call.
package main

import (
	"fmt"
	"os"

	"Post_Analyzer_Webserver/config"
	"Post_Analyzer_Webserver/internal/bootstrap"
	"Post_Analyzer_Webserver/internal/cache"
	"Post_Analyzer_Webserver/internal/logger"
	"Post_Analyzer_Webserver/internal/messaging/kafka"
	"Post_Analyzer_Webserver/internal/messaging/rocketmq"
	"Post_Analyzer_Webserver/internal/service"
	post "Post_Analyzer_Webserver/kitex_gen/post/postservice"

	"github.com/cloudwego/kitex/pkg/utils"
	"github.com/cloudwego/kitex/server"
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

	store, err := bootstrap.InitStorage(cfg)
	if err != nil {
		logger.Error("failed to initialize storage", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	postCache := cache.NewCache(cfg)
	svc := service.NewPostService(store, postCache)

	events := &EventPublisher{}
	if cfg.Messaging.KafkaEnabled {
		events.kafka = kafka.NewProducer(cfg.Messaging.KafkaBrokers, kafka.PostEventsTopic)
		defer events.kafka.Close()
		logger.Info("kafka producer enabled", "brokers", cfg.Messaging.KafkaBrokers, "topic", kafka.PostEventsTopic)
	}
	if cfg.Messaging.RocketMQEnabled {
		rmqProducer, err := rocketmq.NewProducer(cfg.Messaging.RocketMQNsAddrs)
		if err != nil {
			logger.Error("failed to start rocketmq producer", "error", err)
			os.Exit(1)
		}
		events.rocketmq = rmqProducer
		defer events.rocketmq.Close()
		logger.Info("rocketmq producer enabled", "name_servers", cfg.Messaging.RocketMQNsAddrs, "topic", rocketmq.NotificationsTopic)
	}

	handler := NewPostServiceImpl(svc, events)

	addr := utils.NewNetAddr("tcp", cfg.RPC.PostServiceAddr)

	opts := []server.Option{
		server.WithServiceAddr(addr),
	}
	if cfg.RPC.MuxTransport {
		opts = append(opts, server.WithMuxTransport())
	}

	logger.Info("starting postsvc RPC server",
		"addr", cfg.RPC.PostServiceAddr,
		"transport", "TTHeader",
		"mux", cfg.RPC.MuxTransport,
		"storage", cfg.Database.Type,
	)

	svr := post.NewServer(handler, opts...)
	if err := svr.Run(); err != nil {
		logger.Error("postsvc server stopped with error", "error", err)
		os.Exit(1)
	}
}
