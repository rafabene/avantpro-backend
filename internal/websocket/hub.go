package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/rafabene/avantpro-backend/internal/models"
)

// Client represents a single websocket client connection
type Client struct {
	ID     string
	UserID uuid.UUID
	Conn   *websocket.Conn
	Send   chan []byte
	Hub    *Hub
}

// Hub manages all websocket connections and broadcasts
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mutex      sync.RWMutex
}

// Message represents a websocket message
type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// NotificationMessage represents a notification broadcast message
type NotificationMessage struct {
	Type         string                       `json:"type"`
	Notification *models.NotificationResponse `json:"notification"`
	UnreadCount  int                          `json:"unread_count"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in development
		// In production, you should check the origin properly
		return true
	},
}

// NewHub creates a new websocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()
			log.Printf("WebSocket client connected: %s (UserID: %s)", client.ID, client.UserID)

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
				log.Printf("WebSocket client disconnected: %s (UserID: %s)", client.ID, client.UserID)
			}
			h.mutex.Unlock()

		case message := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					delete(h.clients, client)
					close(client.Send)
				}
			}
			h.mutex.RUnlock()
		}
	}
}

// BroadcastToUser sends a message to all connections of a specific user
func (h *Hub) BroadcastToUser(userID uuid.UUID, message []byte) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for client := range h.clients {
		if client.UserID == userID {
			select {
			case client.Send <- message:
			default:
				delete(h.clients, client)
				close(client.Send)
			}
		}
	}
}

// BroadcastNotification sends a notification to a specific user
func (h *Hub) BroadcastNotification(userID uuid.UUID, notification *models.NotificationResponse, unreadCount int) {
	message := NotificationMessage{
		Type:         "notification",
		Notification: notification,
		UnreadCount:  unreadCount,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling notification message: %v", err)
		return
	}

	h.BroadcastToUser(userID, data)
}

// ServeWS handles websocket requests from clients
func (h *Hub) ServeWS(c *gin.Context) {
	// Get user ID from JWT middleware
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		ID:     uuid.New().String(),
		UserID: userUUID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		Hub:    h,
	}

	client.Hub.register <- client

	// Start goroutines to handle the connection
	go client.writePump()
	go client.readPump()
}

// readPump handles incoming messages from the client
func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
		// Handle incoming messages if needed
	}
}

// writePump handles outgoing messages to the client
func (c *Client) writePump() {
	defer c.Conn.Close()

	for message := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("WebSocket write error: %v", err)
			return
		}
	}

	// Send close message when channel is closed
	if err := c.Conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
		log.Printf("WebSocket close message error: %v", err)
	}
}
