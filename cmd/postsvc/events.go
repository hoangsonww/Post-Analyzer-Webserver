package main

import (
	"context"
	"fmt"
	"time"

	"Post_Analyzer_Webserver/internal/gen/eventpb"
	"Post_Analyzer_Webserver/internal/logger"
	"Post_Analyzer_Webserver/internal/messaging/kafka"
	"Post_Analyzer_Webserver/internal/messaging/rocketmq"
	"Post_Analyzer_Webserver/internal/ml/triton"
	"Post_Analyzer_Webserver/internal/models"

	"github.com/google/uuid"
)

// EventPublisher fans out post-lifecycle events to Kafka (analytics
// stream) and RocketMQ (delayed scheduled-recheck notifications), and —
// on creation — enriches the Kafka event with a Triton-classified
// sentiment label. All three dependencies are optional — a nil one makes
// its part a no-op, so postsvc runs fine with any subset disabled.
type EventPublisher struct {
	kafka    *kafka.Producer
	rocketmq *rocketmq.Producer
	triton   *triton.Client
}

// PublishPostEvent sends a PostEvent to Kafka's post.events topic. Errors
// are logged, not returned: a broker outage must never fail the CRUD
// operation that triggered the event. On "created" events, when Triton is
// enabled, the event carries the classified sentiment as an attribute —
// this is the "optional enrichment" the ML integration provides: it rides
// along on the existing event stream rather than gating post creation on
// an inference call.
func (e *EventPublisher) PublishPostEvent(ctx context.Context, eventType string, p models.Post) {
	if e == nil || e.kafka == nil {
		return
	}
	evt := &eventpb.PostEvent{
		EventId:        uuid.NewString(),
		EventType:      eventType,
		PostId:         int64(p.ID),
		UserId:         int64(p.UserID),
		Title:          p.Title,
		BodyLength:     int64(len(p.Body)),
		OccurredAtUnix: time.Now().Unix(),
	}

	if eventType == "created" && e.triton != nil {
		sctx, scancel := context.WithTimeout(ctx, 2*time.Second)
		result, err := e.triton.ClassifySentiment(sctx, p.Title+" "+p.Body)
		scancel()
		if err != nil {
			logger.WarnContext(ctx, "triton sentiment classification failed", "post_id", p.ID, "error", err)
		} else {
			evt.Attributes = map[string]string{"sentiment": result.Label}
			logger.InfoContext(ctx, "post sentiment classified", "post_id", p.ID, "sentiment", result.Label)
		}
	}

	pctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := e.kafka.Publish(pctx, fmt.Sprintf("%d", p.ID), evt); err != nil {
		logger.WarnContext(ctx, "failed to publish post event to kafka", "event_type", eventType, "post_id", p.ID, "error", err)
	}
}

// PublishScheduledRecheck sends a delayed ScheduledNotification to
// RocketMQ, fired ~10s after post creation as a "come back and recheck
// this post" reminder.
func (e *EventPublisher) PublishScheduledRecheck(ctx context.Context, p models.Post) {
	if e == nil || e.rocketmq == nil {
		return
	}
	notif := &eventpb.ScheduledNotification{
		NotificationId: uuid.NewString(),
		Kind:           "scheduled_recheck",
		PostId:         int64(p.ID),
		DeliverAtUnix:  time.Now().Add(10 * time.Second).Unix(),
		Message:        fmt.Sprintf("Recheck post %d (%q) for updated analysis", p.ID, p.Title),
	}
	pctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := e.rocketmq.PublishDelayed(pctx, rocketmq.NotificationsTopic, notif, rocketmq.DelayLevelTenSeconds); err != nil {
		logger.WarnContext(ctx, "failed to publish scheduled recheck to rocketmq", "post_id", p.ID, "error", err)
	}
}
