package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"ai-notetaking-be/internal/model"
	"ai-notetaking-be/internal/pkg/logger"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Hub struct {
	// Registered clients map: UserID -> List of Clients (multi-device)
	clients map[uuid.UUID][]*Client

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	// Lock for safe map access
	mu sync.RWMutex

	// Redis connection for cross-instance communication
	rdb *redis.Client

	// Dedicated Logger
	logger logger.ILogger
}

func NewHub(rdb *redis.Client, log logger.ILogger) *Hub {
	return &Hub{
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[uuid.UUID][]*Client),
		rdb:        rdb,
		logger:     log,
	}
}

func (h *Hub) Run() {
	// Start Redis Subscriber if Redis is available
	if h.rdb != nil {
		go h.subscribeToRedis()
	}

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.UserID] = append(h.clients[client.UserID], client)
			h.mu.Unlock()
			h.logger.Info("Hub", "Client registered", map[string]interface{}{"user_id": client.UserID})

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.UserID]; ok {
				for i, c := range clients {
					if c == client {
						// Remove from slice
						h.clients[client.UserID] = append(clients[:i], clients[i+1:]...)
						close(client.Send)
						break
					}
				}
				if len(h.clients[client.UserID]) == 0 {
					delete(h.clients, client.UserID)
					h.logger.Info("Hub", "Client completely unregistered", map[string]interface{}{"user_id": client.UserID})
				}
			}
			h.mu.Unlock()
		}
	}
}

// Broadcast sends a notification to ALL connected clients.
func (h *Hub) Broadcast(notification model.Notification) {
	// 1. Serialize
	data, _ := json.Marshal(map[string]interface{}{
		"type": "notification",
		"data": notification,
	})

	// 2. Send to all local clients
	h.mu.RLock()
	for _, clients := range h.clients {
		for _, client := range clients {
			select {
			case client.Send <- data:
			default:
				close(client.Send)
				h.unregister <- client
			}
		}
	}
	h.mu.RUnlock()

	// 3. Publish to Redis for other instances
	// Use special "broadcast" channel or payload flag
	if h.rdb != nil {
		payload := map[string]interface{}{
			"target_user_id": "*", // Wildcard for broadcast
			"message":        data,
		}
		jsonPayload, _ := json.Marshal(payload)
		h.rdb.Publish(context.Background(), "cluster_events", jsonPayload)
	}
}

// Send (NotificationDelivery interface implementation)
func (h *Hub) Send(userID uuid.UUID, notification model.Notification) {
	// 1. Serialize
	data, _ := json.Marshal(map[string]interface{}{
		"type": "notification",
		"data": notification,
	})

	// 2. Check locally
	h.mu.RLock()
	clients, localFound := h.clients[userID]
	h.mu.RUnlock()

	if localFound {
		for _, client := range clients {
			select {
			case client.Send <- data:
			default:
				h.logger.Warn("Hub", "Client Send buffer full, dropping message", map[string]interface{}{"user_id": userID})
				close(client.Send)
				h.unregister <- client
			}
		}
	}

	// 3. Publish to Redis depending on strategy
	// (Always publish for multi-device support as discussed)
	if h.rdb != nil {
		payload := map[string]interface{}{
			"target_user_id": userID.String(),
			"message":        data,
		}
		jsonPayload, _ := json.Marshal(payload)
		h.rdb.Publish(context.Background(), "cluster_events", jsonPayload)
	}
}

func (h *Hub) subscribeToRedis() {
	// Subscribe to a wildcard pattern? No, Redis Cluster doesnt support keyspace notifications easily for this.
	// We need a mechanism.
	// Option A: Subscribe to "global_notifications" and filter? Too heavy.
	// Option B: Every instance subscribes to ALL users it has locally.
	// This is complex.

	// SIMPLIFIED APPROACH:
	// Use a single "global_broadcast" channel is bad.
	// Use "user_events" channel where we send {target_user_id, data}.
	// All instances subscribe to "user_events".
	// When message arrives, check if we have the user locally. If yes, send.

	ctx := context.Background()
	pubsub := h.rdb.Subscribe(ctx, "cluster_events")
	defer pubsub.Close()

	ch := pubsub.Channel()

	for msg := range ch {
		// Parse message
		var payload struct {
			TargetUserID string          `json:"target_user_id"`
			Message      json.RawMessage `json:"message"`
		}
		if err := json.Unmarshal([]byte(msg.Payload), &payload); err != nil {
			log.Printf("Redis msg parse error: %v", err)
			continue
		}

		// Check for Broadcast
		if payload.TargetUserID == "*" {
			// Broadcast to all local clients
			h.mu.RLock()
			for _, clients := range h.clients {
				for _, client := range clients {
					select {
					case client.Send <- payload.Message:
					default:
						close(client.Send)
						h.unregister <- client
					}
				}
			}
			h.mu.RUnlock()
			continue
		}

		uid, err := uuid.Parse(payload.TargetUserID)
		if err != nil {
			continue
		}

		// Check local
		h.mu.RLock()
		clients, ok := h.clients[uid]
		h.mu.RUnlock()

		if ok {
			for _, client := range clients {
				select {
				case client.Send <- payload.Message:
				default:
					close(client.Send)
					h.unregister <- client
				}
			}
		}
	}
}
