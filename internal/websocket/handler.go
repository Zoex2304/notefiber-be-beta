package websocket

import (
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
)

// ServeWs handles websocket requests from the peer.
// ServeWs handles websocket requests from the peer.
func ServeWs(hub *Hub, c *websocket.Conn, userID uuid.UUID) {
	client := &Client{Hub: hub, Conn: c, UserID: userID, Send: make(chan []byte, 256)}
	client.Hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	client.readPump() // Run readPump in current goroutine (handler)
}
