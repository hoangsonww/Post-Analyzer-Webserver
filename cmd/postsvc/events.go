package main

import (
	"context"
	"fmt"
	"time"

	"Post_Analyzer_Webserver/internal/gen/eventpb"
	"Post_Analyzer_Webserver/internal/logger"
	"Post_Analyzer_Webserver/internal/messaging/kafka"
	"Post_Analyzer_Webserver/internal/messaging/rocketmq"
	"Post_Analyzer_Webserver/internal/models"

	"github.com/google/uuid"
)

// EventPublisher fans out post-lifecycle events to Kafka (analytics
// stream) and RocketMQ (delayed scheduled-recheck notifications). Both
// producers are optional — a nil producer makes the corresponding publish
// a no-op, so postsvc runs fine with either or both brokers disabled.
type EventPublisher struct {
	kafka    *kafka.Producer
	rocketmq *rocketmq.Producer
}

// PublishPostEvent sends a PostEvent to Kafka's post.events topic. Errors
// are logged, not returned: a broker outage must never fail the CRUD
// operation that triggered the event.
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
