package handler

import (
	"os"
	"time"

	"ai-notetaking-be/internal/pkg/logger"
	"ai-notetaking-be/internal/pkg/serverutils"
	"ai-notetaking-be/internal/service"
	internalWS "ai-notetaking-be/internal/websocket"
	"ai-notetaking-be/pkg/events"
	pktNats "ai-notetaking-be/pkg/nats"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type NotificationHandler struct {
	service   *service.NotificationService
	publisher *pktNats.Publisher
	hub       *internalWS.Hub
	logger    logger.ILogger
}

func NewNotificationHandler(service *service.NotificationService, pub *pktNats.Publisher, hub *internalWS.Hub, log logger.ILogger) *NotificationHandler {
	return &NotificationHandler{
		service:   service,
		publisher: pub,
		hub:       hub,
		logger:    log,
	}
}

// ServeWs handles websocket requests from the peer.
func (h *NotificationHandler) ServeWs(c *fiber.Ctx) error {
	// 1. Get Token source
	// Priority 1: Query Param (Browser standard)
	tokenStr := c.Query("token")

	// Priority 2: Authorization Header (Tooling/Non-browser standard)
	if tokenStr == "" {
		authHeader := c.Get("Authorization")
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenStr = authHeader[7:]
		}
	}

	if tokenStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing token (Query 'token' or Header 'Authorization')"})
	}

	// 2. Parse JWT
	// Note: We need to use the exact same secret as Auth/Middleware
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		// Ensure Signing Method is HMAC
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fiber.ErrUnauthorized
		}
		// In a real app, this secret should come from config/env injected into handler
		// For now we use os.Getenv as per other parts of the app
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil || !token.Valid {
		h.logger.Warn("NotificationHandler", "Invalid Token in WS Handshake", map[string]interface{}{"error": err})
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
	}

	// 3. Extract UserID from Claim
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token claims"})
	}

	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Token missing user_id"})
	}

	// 4. Parse as UUID
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID format in token"})
	}

	// Upgrade via Fiber WebSocket Middleware
	// We handle the upgrade here using the helper which automatically hijacks the connection
	if websocket.IsWebSocketUpgrade(c) {
		return websocket.New(func(c *websocket.Conn) {
			h.logger.Info("NotificationHandler", "Starting WebSocket session", map[string]interface{}{"user_id": userID})
			internalWS.ServeWs(h.hub, c, userID)
			h.logger.Info("NotificationHandler", "WebSocket session ended", map[string]interface{}{"user_id": userID})
		})(c)
	}
	return fiber.ErrUpgradeRequired
}

// GetNotifications returns the user's notifications.
func (h *NotificationHandler) GetNotifications(c *fiber.Ctx) error {
	// Assuming Auth middleware sets "user_id" in locals as a string
	// If it's stored as UUID, we cast differently. Assuming string based on common Fiber patterns.
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok {
		// Try to see if it's already a UUID
		if uid, ok := c.Locals("user_id").(uuid.UUID); ok {
			userIDStr = uid.String()
		} else {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
		}
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)

	notifications, total, err := h.service.GetNotifications(c.UserContext(), userID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"data":  notifications,
		"total": total,
		"page":  offset/limit + 1,
		"limit": limit,
	})
}

// GetUnreadCount returns the number of unread notifications.
func (h *NotificationHandler) GetUnreadCount(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	count, err := h.service.GetUnreadCount(c.UserContext(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"count": count})
}

// MarkAsRead marks a specific notification as read.
func (h *NotificationHandler) MarkAsRead(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID"})
	}

	if err := h.service.MarkAsRead(c.UserContext(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true})
}

// MarkAllAsRead marks all user's notifications as read.
func (h *NotificationHandler) MarkAllAsRead(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	if err := h.service.MarkAllAsRead(c.UserContext(), userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true})
}

// DebugTriggerEvent simulates an event to test the flow.
func (h *NotificationHandler) DebugTriggerEvent(c *fiber.Ctx) error {
	type Request struct {
		Type    string                 `json:"type"`
		Payload map[string]interface{} `json:"payload"`
	}
	var req Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if req.Type == "" {
		req.Type = "TEST_EVENT"
	}
	if req.Payload == nil {
		req.Payload = make(map[string]interface{})
	}

	// If no user_id in payload, use current user if available
	if _, ok := req.Payload["user_id"]; !ok {
		if uid := c.Locals("user_id"); uid != nil {
			req.Payload["user_id"] = uid
		}
	}

	evt := events.BaseEvent{
		Type:       req.Type,
		Data:       req.Payload,
		OccurredAt: time.Now(),
	}

	if h.publisher == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Event publisher not configured"})
	}

	// Use background context for async publish simulation, or UserContext if we want to trace it.
	// Since Publisher uses context for timeout, UserContext is fine.
	if err := h.publisher.Publish(c.UserContext(), evt); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "Event Published", "event": evt})
}

// Broadcast sends a system-wide notification.
func (h *NotificationHandler) Broadcast(c *fiber.Ctx) error {
	type Request struct {
		Title   string `json:"title"`
		Message string `json:"message"`
	}
	var req Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if req.Title == "" || req.Message == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Title and Message are required"})
	}

	evt := events.BaseEvent{
		Type: "SYSTEM_BROADCAST",
		Data: map[string]interface{}{
			"title":   req.Title,
			"message": req.Message,
		},
		OccurredAt: time.Now(),
	}

	if h.publisher != nil {
		if err := h.publisher.Publish(c.UserContext(), evt); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
	}

	return c.JSON(fiber.Map{"status": "Broadcast Queued"})
}

// RegisterRoutes registers the notification routes.
func (h *NotificationHandler) RegisterRoutes(router fiber.Router) {
	notif := router.Group("/notifications")
	notif.Use(serverutils.JwtMiddleware)
	notif.Get("/", h.GetNotifications)
	notif.Get("/unread-count", h.GetUnreadCount) // Order matters if :id conflicts, but specific first usually ok
	notif.Patch("/:id/read", h.MarkAsRead)
	notif.Patch("/read-all", h.MarkAllAsRead)
	notif.Post("/broadcast", h.Broadcast) // Admin only ideally, but keeping open for now (or use middleware)

	debug := router.Group("/debug")
	debug.Post("/trigger-notification", h.DebugTriggerEvent)

	// WebSocket
	router.Get("/ws", h.ServeWs)
}
