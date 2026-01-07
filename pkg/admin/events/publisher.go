package events

import (
	"context"
	"time"

	"ai-notetaking-be/internal/pkg/logger"
	pkgEvents "ai-notetaking-be/pkg/events"
	pktNats "ai-notetaking-be/pkg/nats"

	"github.com/google/uuid"
)

// Publisher abstracts event publishing for admin operations
type Publisher interface {
	PublishUserRegistered(ctx context.Context, userId uuid.UUID, email, fullName, source string)
	PublishRefundApproved(ctx context.Context, refundId, subscriptionId, userId uuid.UUID, amount float64, reason string)
	PublishRefundRejected(ctx context.Context, refundId, subscriptionId, userId uuid.UUID, reason string)
	PublishAiLimitUpdated(ctx context.Context, userId uuid.UUID, email string, oldChat, newChat, oldSearch, newSearch int, description string)
	PublishCancellationProcessed(ctx context.Context, cancellationId, subscriptionId, userId uuid.UUID, planName, status string)
}

// NatsPublisher implements Publisher using NATS
type NatsPublisher struct {
	publisher *pktNats.Publisher
	logger    logger.ILogger
}

// NewNatsPublisher creates a new NATS-based event publisher
func NewNatsPublisher(publisher *pktNats.Publisher, logger logger.ILogger) *NatsPublisher {
	return &NatsPublisher{
		publisher: publisher,
		logger:    logger,
	}
}

// PublishUserRegistered emits USER_REGISTERED event for admin-created users
func (p *NatsPublisher) PublishUserRegistered(ctx context.Context, userId uuid.UUID, email, fullName, source string) {
	if p.publisher == nil {
		return
	}

	evt := pkgEvents.BaseEvent{
		Type: "USER_REGISTERED",
		Data: map[string]interface{}{
			"user_id":   userId,
			"email":     email,
			"full_name": fullName,
			"source":    source,
		},
		OccurredAt: time.Now(),
	}

	if err := p.publisher.Publish(ctx, evt); err != nil {
		p.logger.Error("ADMIN", "Failed to publish USER_REGISTERED event", map[string]interface{}{"error": err.Error()})
	}
}

// PublishRefundApproved emits REFUND_APPROVED event
func (p *NatsPublisher) PublishRefundApproved(ctx context.Context, refundId, subscriptionId, userId uuid.UUID, amount float64, reason string) {
	if p.publisher == nil {
		return
	}

	now := time.Now()
	evt := pkgEvents.BaseEvent{
		Type: "REFUND_APPROVED",
		Data: map[string]interface{}{
			"refund_id":       refundId,
			"subscription_id": subscriptionId,
			"user_id":         userId,
			"amount":          amount,
			"reason":          reason,
			"entity_type":     "refund",
			"entity_id":       refundId.String(),
			"occurred_at":     now,
		},
		OccurredAt: now,
	}

	if err := p.publisher.Publish(ctx, evt); err != nil {
		p.logger.Error("ADMIN", "Failed to publish REFUND_APPROVED event", map[string]interface{}{"error": err.Error()})
	}
}

// PublishRefundRejected emits REFUND_REJECTED event
func (p *NatsPublisher) PublishRefundRejected(ctx context.Context, refundId, subscriptionId, userId uuid.UUID, reason string) {
	if p.publisher == nil {
		return
	}

	now := time.Now()
	evt := pkgEvents.BaseEvent{
		Type: "REFUND_REJECTED",
		Data: map[string]interface{}{
			"refund_id":       refundId,
			"subscription_id": subscriptionId,
			"user_id":         userId,
			"reason":          reason,
			"entity_type":     "refund",
			"entity_id":       refundId.String(),
			"occurred_at":     now,
		},
		OccurredAt: now,
	}

	if err := p.publisher.Publish(ctx, evt); err != nil {
		p.logger.Error("ADMIN", "Failed to publish REFUND_REJECTED event", map[string]interface{}{"error": err.Error()})
	}
}

// PublishAiLimitUpdated emits AI_LIMIT_UPDATED event
func (p *NatsPublisher) PublishAiLimitUpdated(ctx context.Context, userId uuid.UUID, email string, oldChat, newChat, oldSearch, newSearch int, description string) {
	if p.publisher == nil {
		return
	}

	evt := pkgEvents.BaseEvent{
		Type: "AI_LIMIT_UPDATED",
		Data: map[string]interface{}{
			"user_id":                        userId.String(),
			"user_email":                     email,
			"previous_chat_usage":            oldChat,
			"new_chat_usage":                 newChat,
			"previous_semantic_search_usage": oldSearch,
			"new_semantic_search_usage":      newSearch,
			"limit_description":              description,
			"entity_type":                    "user",
			"entity_id":                      userId.String(),
		},
		OccurredAt: time.Now(),
	}

	if err := p.publisher.Publish(ctx, evt); err != nil {
		p.logger.Error("ADMIN", "Failed to publish AI_LIMIT_UPDATED event", map[string]interface{}{"error": err.Error()})
	}
}

// PublishCancellationProcessed emits SUBSCRIPTION_CANCELLATION_PROCESSED event
func (p *NatsPublisher) PublishCancellationProcessed(ctx context.Context, cancellationId, subscriptionId, userId uuid.UUID, planName, status string) {
	if p.publisher == nil {
		return
	}

	now := time.Now()
	evt := pkgEvents.BaseEvent{
		Type: "SUBSCRIPTION_CANCELLATION_PROCESSED",
		Data: map[string]interface{}{
			"cancellation_id": cancellationId,
			"subscription_id": subscriptionId,
			"user_id":         userId,
			"plan_name":       planName,
			"status":          status,
			"entity_type":     "cancellation",
			"entity_id":       cancellationId.String(),
			"occurred_at":     now,
		},
		OccurredAt: now,
	}

	if err := p.publisher.Publish(ctx, evt); err != nil {
		p.logger.Error("ADMIN", "Failed to publish SUBSCRIPTION_CANCELLATION_PROCESSED event", map[string]interface{}{"error": err.Error()})
	}
}
