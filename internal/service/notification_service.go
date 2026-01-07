package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ai-notetaking-be/internal/model"
	"ai-notetaking-be/internal/pkg/logger"
	"ai-notetaking-be/internal/repository"
	"ai-notetaking-be/pkg/events"
	pktNats "ai-notetaking-be/pkg/nats" // Renamed to avoid collision

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// NotificationDelivery defines how to push real-time updates.
// Typically implemented by the WebSocket Hub.
type NotificationDelivery interface {
	Send(userID uuid.UUID, notification model.Notification)
	Broadcast(notification model.Notification)
}

type NotificationService struct {
	repo       repository.NotificationRepository
	subscriber *pktNats.Subscriber
	delivery   NotificationDelivery
	logger     logger.ILogger
}

func NewNotificationService(repo repository.NotificationRepository, sub *pktNats.Subscriber, delivery NotificationDelivery, log logger.ILogger) *NotificationService {
	return &NotificationService{
		repo:       repo,
		subscriber: sub,
		delivery:   delivery,
		logger:     log,
	}
}

// Start begins listening to the event bus.
func (s *NotificationService) Start() {
	// Subscribe to all events with a durable consumer
	err := s.subscriber.Subscribe("events.>", "notif-service-worker", s.handleEvent)
	if err != nil {
		s.logger.Error("NotificationService", "Failed to start notification subscriber", map[string]interface{}{"error": err})
		// If critical background service fails, maybe panic/fatal?
		// log.Fatalf("Failed to start notification subscriber: %v", err)
		return
	}
	s.logger.Info("NotificationService", "Notification service started, listening to events.>", nil)
}

func (s *NotificationService) handleEvent(ctx context.Context, event events.Event) error {
	s.logger.Info("NotificationService", fmt.Sprintf("Processing event: %s", event.EventType()), map[string]interface{}{"type": event.EventType()})

	// 1. Get Config
	// Strip "events." prefix from type if present (NATS subject includes stream name)
	typeCode := strings.TrimPrefix(event.EventType(), "events.")

	config, err := s.repo.GetNotificationTypeByCode(ctx, typeCode)
	if err != nil {
		s.logger.Warn("NotificationService", fmt.Sprintf("Config not found for code: '%s'", typeCode), map[string]interface{}{"error": err.Error()})
		return nil
	}
	if !config.IsActive {
		s.logger.Info("NotificationService", fmt.Sprintf("Notification type '%s' is inactive", typeCode), nil)
		return nil
	}

	// SPECIAL HANDLING: Social Proof for Subscriptions
	// Independently of the configured notification (which might be for Admins),
	// we want to broadcast to ALL users that someone subscribed.
	if event.EventType() == "SUBSCRIPTION_CREATED" {
		payload := event.Payload()
		fullName, _ := payload["full_name"].(string)
		avatarURL, _ := payload["avatar_url"].(string)

		if fullName != "" {
			socialProofNotif := model.Notification{
				ID:        uuid.New(),
				UserID:    uuid.Nil, // Broadcast target
				TypeCode:  "SOCIAL_PROOF",
				Title:     "New Subscriber!",
				Message:   fmt.Sprintf("%s just subscribed to Pro Plan!", fullName),
				Metadata:  datatypes.JSON(config.Template), // Or custom metadata
				CreatedAt: time.Now(),
				IsRead:    false,
			}
			// Inject metadata properly
			metaMap := map[string]interface{}{
				"avatar_url": avatarURL,
				"full_name":  fullName,
				"plan_name":  payload["plan_name"],
				"type":       "flexing",
			}
			metaJSON, _ := json.Marshal(metaMap)
			socialProofNotif.Metadata = datatypes.JSON(metaJSON)

			if s.delivery != nil {
				s.delivery.Broadcast(socialProofNotif)
			}
		}
	}

	// 2. Broadcast Handling
	if config.TargetType == "BROADCAST" {
		// Create ephemeral notification object (without saving to user_id one by one, scaling concern)
		// Or create one with nil UserID if schema allows?
		// Schema requires UserID not null.
		// For true broadcast, we might not want to save millions of rows.
		// For now, we only push via WebSocket and don't save to individual inbox to save DB space?
		// OR we iterate all users? Iterating all users is slow for "real-time".
		// Decision: Broadcast = Push Only (Ephemeral) OR we need a "SystemNotification" table.
		// Let's implement Push Only for "SYSTEM_BROADCAST" for performance in this iteration.

		notif := s.buildNotification(uuid.Nil, config, event)

		if s.delivery != nil {
			s.delivery.Broadcast(notif)
		}
		return nil
	}

	// 3. Resolve Recipients
	recipients, err := s.resolveRecipients(ctx, config, event)
	if err != nil {
		s.logger.Error("NotificationService", fmt.Sprintf("Error resolving recipients for %s", event.EventType()), map[string]interface{}{"error": err})
		return err // NATS will retry if we return error
	}
	s.logger.Info("NotificationService", "Recipients resolved", map[string]interface{}{"count": len(recipients), "type": config.TargetType})

	// 3. Process Per Recipient
	for _, userID := range recipients {
		// Create Notification
		notif := s.buildNotification(userID, config, event)

		// Save to DB
		if err := s.repo.CreateNotification(ctx, &notif); err != nil {
			s.logger.Error("NotificationService", fmt.Sprintf("Error saving notification for user %s", userID), map[string]interface{}{"error": err})
			continue // Partial failure? Should we retry entire batch? For now continue.
		}

		// Real-time Delivery
		if s.delivery != nil {
			s.delivery.Send(userID, notif)
		}
	}

	return nil
}

func (s *NotificationService) resolveRecipients(ctx context.Context, config *model.NotificationType, event events.Event) ([]uuid.UUID, error) {
	var userIDs []uuid.UUID

	switch config.TargetType {
	case "SELF":
		// Expect "user_id" in payload or we assume the event has a way to identify the owner.
		// For our Event interface, Payload is a map. We rely on convention.
		if uidStr, ok := event.Payload()["user_id"].(string); ok {
			uid, err := uuid.Parse(uidStr)
			if err == nil {
				userIDs = append(userIDs, uid)
			}
		} else {
			// Try "userID" camelCase?
			// Or check if base event struct has it (but interface doesn't expose it directly)
			s.logger.Warn("NotificationService", fmt.Sprintf("TargetType SELF but no user_id found in payload for event %s", event.EventType()), nil)
		}

	case "ADMIN":
		admins, err := s.repo.GetUsersByRole(ctx, "admin")
		if err != nil {
			return nil, err
		}
		s.logger.Info("NotificationService", "Resolved admins", map[string]interface{}{"count": len(admins)})
		for _, u := range admins {
			userIDs = append(userIDs, u.Id)
		}

	case "ROLE":
		// Use config.TargetRole
		users, err := s.repo.GetUsersByRole(ctx, config.TargetRole)
		if err != nil {
			return nil, err
		}
		for _, u := range users {
			userIDs = append(userIDs, u.Id)
		}
	}

	return userIDs, nil
}

func (s *NotificationService) buildNotification(userID uuid.UUID, config *model.NotificationType, event events.Event) model.Notification {
	// Simple Template Engine
	msg := config.Template
	payload := event.Payload()

	for k, v := range payload {
		placeholder := fmt.Sprintf("{%s}", k)
		valStr := fmt.Sprintf("%v", v)
		msg = strings.ReplaceAll(msg, placeholder, valStr)
	}

	// Actor?
	var actorID *uuid.UUID
	if actorStr, ok := payload["actor_id"].(string); ok {
		if aid, err := uuid.Parse(actorStr); err == nil {
			actorID = &aid
		}
	}

	// Entity?
	entityType := ""
	var entityID *uuid.UUID

	if et, ok := payload["entity_type"].(string); ok {
		entityType = et
	}
	if eidStr, ok := payload["entity_id"].(string); ok {
		if eid, err := uuid.Parse(eidStr); err == nil {
			entityID = &eid
		}
	}

	// Metadata - enrich with action_url for deep linking
	metaMap := make(map[string]interface{})
	for k, v := range payload {
		metaMap[k] = v
	}
	// Generate action_url if entity info is present
	if entityType != "" && entityID != nil {
		metaMap["action_url"] = fmt.Sprintf("/%ss/%s", entityType, entityID.String())
	}
	metaJSON, _ := json.Marshal(metaMap)

	return model.Notification{
		ID:         uuid.New(),
		UserID:     userID,
		ActorID:    actorID,
		TypeCode:   config.Code,
		Title:      config.DisplayName,
		Message:    msg,
		Metadata:   datatypes.JSON(metaJSON),
		EntityType: entityType,
		EntityID:   entityID,
		CreatedAt:  time.Now(),
		IsRead:     false,
	}
}

// GetNotifications fetches notifications for a user.
func (s *NotificationService) GetNotifications(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.Notification, int64, error) {
	return s.repo.GetNotificationsByUserID(ctx, userID, limit, offset)
}

// GetUnreadCount fetches unread count.
func (s *NotificationService) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	return s.repo.GetUnreadCount(ctx, userID)
}

// MarkAsRead marks a notification as read.
func (s *NotificationService) MarkAsRead(ctx context.Context, id uuid.UUID) error {
	return s.repo.MarkAsRead(ctx, id)
}

// MarkAllAsRead marks all notifications as read for a user.
func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	return s.repo.MarkAllAsRead(ctx, userID)
}
