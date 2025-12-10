package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// Message types
const (
	MsgChat         = "chat"
	MsgSystem       = "system"
	MsgNotification = "notification"
)

// Message structure for chat events
type Message struct {
	Type     string `json:"type"`
	Room     string `json:"room"`
	Username string `json:"username"`
	Text     string `json:"text"`
	Time     string `json:"time"`
}

// Notification structure for admin notifications
type Notification struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`      // "info", "warning", "error", "success"
	Title     string    `json:"title"`     // Notification title
	Message   string    `json:"message"`   // Notification content
	Timestamp time.Time `json:"timestamp"` // When notification was created
	Target    string    `json:"target"`    // "all", "room:name", "user:username"
}

// Client represents a connected user
type Client struct {
	ID       string
	Username string
	Conn     *websocket.Conn
	Room     string
	Send     chan []byte
}

// Room represents a chat room with multiple clients
type Room struct {
	Name    string
	Clients map[*Client]bool
	mu      sync.RWMutex
}

// Hub manages all rooms and clients
type Hub struct {
	rooms              map[string]*Room
	register           chan *Client
	unregister         chan *Client
	notificationChan   chan *Notification
	notificationHistory []Notification // Last 50 notifications
	mu                 sync.RWMutex
}

// Create new hub instance
func newHub() *Hub {
	return &Hub{
		rooms:               make(map[string]*Room),
		register:            make(chan *Client),
		unregister:          make(chan *Client),
		notificationChan:    make(chan *Notification, 100),
		notificationHistory: make([]Notification, 0, 50),
	}
}

// Run starts the hub's main event loop
func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.addClientToRoom(client)
			// Send notification history to new client
			h.sendNotificationHistory(client)

		case client := <-h.unregister:
			h.removeClientFromRoom(client)

		case notif := <-h.notificationChan:
			// Add to history (keep last 50)
			h.addToHistory(notif)
			// Route notification to target
			h.routeNotification(notif)
		}
	}
}

// addClientToRoom adds a client to specified room
func (h *Hub) addClientToRoom(client *Client) {
	h.mu.Lock()

	// Get existing room or create new one
	room, exists := h.rooms[client.Room]
	if !exists {
		room = &Room{
			Name:    client.Room,
			Clients: make(map[*Client]bool),
		}
		h.rooms[client.Room] = room
		log.Printf("Created new room: %s", client.Room)
	}

	// Add client to room
	room.mu.Lock()
	room.Clients[client] = true
	clientCount := len(room.Clients)
	room.mu.Unlock()

	h.mu.Unlock() // ⭐ UNLOCK TRƯỚC KHI BROADCAST!

	log.Printf("Client %s joined room %s (Total: %d)",
		client.Username, client.Room, clientCount)

	// Send join notification
	msg := Message{
		Type:     MsgSystem,
		Room:     client.Room,
		Username: client.Username,
		Text:     fmt.Sprintf("%s joined the room", client.Username),
		Time:     time.Now().Format("15:04:05"),
	}
	h.broadcastToRoom(client.Room, msg)
}

// removeClientFromRoom removes a client from their room
func (h *Hub) removeClientFromRoom(client *Client) {
	h.mu.RLock()
	room, exists := h.rooms[client.Room]
	h.mu.RUnlock() // ⭐ UNLOCK NGAY

	if !exists {
		return
	}

	// Remove client from room
	room.mu.Lock()
	if _, ok := room.Clients[client]; ok {
		delete(room.Clients, client)
		close(client.Send)
	}
	clientCount := len(room.Clients)
	room.mu.Unlock()

	log.Printf("Client %s left room %s (Remaining: %d)",
		client.Username, client.Room, clientCount)

	// Send leave notification
	msg := Message{
		Type:     MsgSystem,
		Room:     client.Room,
		Username: client.Username,
		Text:     fmt.Sprintf("%s left the room", client.Username),
		Time:     time.Now().Format("15:04:05"),
	}
	h.broadcastToRoom(client.Room, msg)

	// Delete room if empty
	if clientCount == 0 {
		h.mu.Lock()
		delete(h.rooms, client.Room)
		h.mu.Unlock()
		log.Printf("Deleted empty room: %s", client.Room)
	}
}

// broadcastToRoom sends message to all clients in specified room
func (h *Hub) broadcastToRoom(roomName string, msg Message) {
	h.mu.RLock()
	room, exists := h.rooms[roomName]
	h.mu.RUnlock()

	if !exists {
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	for client := range room.Clients {
		select {
		case client.Send <- data:
		default:
			close(client.Send)
			delete(room.Clients, client)
		}
	}
}

// addToHistory adds notification to history (keep last 50)
func (h *Hub) addToHistory(notif *Notification) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.notificationHistory = append(h.notificationHistory, *notif)

	// Keep only last 50 notifications
	if len(h.notificationHistory) > 50 {
		h.notificationHistory = h.notificationHistory[len(h.notificationHistory)-50:]
	}

	log.Printf("Added notification to history: %s (Total: %d)", notif.ID, len(h.notificationHistory))
}

// sendNotificationHistory sends last notifications to new client
func (h *Hub) sendNotificationHistory(client *Client) {
	h.mu.RLock()
	history := make([]Notification, len(h.notificationHistory))
	copy(history, h.notificationHistory)
	h.mu.RUnlock()

	// Send each notification to client
	for _, notif := range history {
		// Only send relevant notifications
		if h.shouldReceiveNotification(client, &notif) {
			data, _ := json.Marshal(notif)
			select {
			case client.Send <- data:
			default:
				// Skip if channel full
			}
		}
	}

	log.Printf("Sent %d historical notifications to %s", len(history), client.Username)
}

// shouldReceiveNotification checks if client should receive notification
func (h *Hub) shouldReceiveNotification(client *Client, notif *Notification) bool {
	// Parse target
	if notif.Target == "all" {
		return true
	}

	if strings.HasPrefix(notif.Target, "room:") {
		roomName := strings.TrimPrefix(notif.Target, "room:")
		return client.Room == roomName
	}

	if strings.HasPrefix(notif.Target, "user:") {
		username := strings.TrimPrefix(notif.Target, "user:")
		return client.Username == username
	}

	return false
}

// routeNotification routes notification to appropriate clients
func (h *Hub) routeNotification(notif *Notification) {
	data, err := json.Marshal(notif)
	if err != nil {
		log.Printf("Failed to marshal notification: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	// Route based on target
	if notif.Target == "all" {
		// Broadcast to all clients in all rooms
		for _, room := range h.rooms {
			room.mu.RLock()
			for client := range room.Clients {
				select {
				case client.Send <- data:
				default:
					// Skip if channel full
				}
			}
			room.mu.RUnlock()
		}
		log.Printf("Broadcasted notification %s to all users", notif.ID)

	} else if strings.HasPrefix(notif.Target, "room:") {
		// Send to specific room
		roomName := strings.TrimPrefix(notif.Target, "room:")
		room, exists := h.rooms[roomName]
		if exists {
			room.mu.RLock()
			for client := range room.Clients {
				select {
				case client.Send <- data:
				default:
					// Skip if channel full
				}
			}
			room.mu.RUnlock()
			log.Printf("Sent notification %s to room %s", notif.ID, roomName)
		}

	} else if strings.HasPrefix(notif.Target, "user:") {
		// Send to specific user
		username := strings.TrimPrefix(notif.Target, "user:")
		sent := false
		for _, room := range h.rooms {
			room.mu.RLock()
			for client := range room.Clients {
				if client.Username == username {
					select {
					case client.Send <- data:
						sent = true
					default:
						// Skip if channel full
					}
				}
			}
			room.mu.RUnlock()
		}
		if sent {
			log.Printf("Sent notification %s to user %s", notif.ID, username)
		} else {
			log.Printf("User %s not found for notification %s", username, notif.ID)
		}
	}
}

// readPump reads messages from WebSocket connection
func (c *Client) readPump(hub *Hub) {
	defer func() {
		hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, data, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error: %v", err)
			}
			break
		}

		// Parse incoming JSON message
		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			continue
		}

		// Regular chat message - set metadata
		msg.Username = c.Username
		msg.Room = c.Room
		msg.Type = MsgChat
		msg.Time = time.Now().Format("15:04:05")

		// Broadcast to room
		hub.broadcastToRoom(c.Room, msg)
	}
}

// writePump writes messages to WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Global hub instance
var hub = newHub()

// handleWebSocket handles WebSocket connection upgrades
func handleWebSocket(c *gin.Context) {
	username := c.Query("username")
	room := c.Query("room")

	if username == "" || room == "" {
		c.JSON(400, gin.H{"error": "username and room required"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Upgrade failed: %v", err)
		return
	}

	client := &Client{
		ID:       fmt.Sprintf("%s-%d", username, time.Now().Unix()),
		Username: username,
		Room:     room,
		Conn:     conn,
		Send:     make(chan []byte, 256),
	}

	// Start goroutines BEFORE registering to hub
	go client.writePump()
	go client.readPump(hub)
	
	// Register client to hub (this will send join notification + history)
	hub.register <- client
}

// handleNotification handles admin notification requests
func handleNotification(c *gin.Context) {
	var notif Notification
	if err := c.BindJSON(&notif); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Validate notification
	if notif.Message == "" {
		c.JSON(400, gin.H{"error": "message is required"})
		return
	}

	if notif.Target == "" {
		c.JSON(400, gin.H{"error": "target is required"})
		return
	}

	// Set default values
	if notif.ID == "" {
		notif.ID = fmt.Sprintf("notif-%d", time.Now().UnixNano())
	}
	if notif.Type == "" {
		notif.Type = "info"
	}
	notif.Timestamp = time.Now()

	// Send notification to hub
	hub.notificationChan <- &notif

	log.Printf("Received notification: ID=%s, Type=%s, Target=%s", notif.ID, notif.Type, notif.Target)

	c.JSON(200, gin.H{
		"status": "sent",
		"id":     notif.ID,
	})
}

// getStats returns current statistics
func getStats(c *gin.Context) {
	hub.mu.RLock()
	defer hub.mu.RUnlock()

	totalUsers := 0
	roomDetails := make(map[string]int)

	for roomName, room := range hub.rooms {
		room.mu.RLock()
		userCount := len(room.Clients)
		room.mu.RUnlock()

		totalUsers += userCount
		roomDetails[roomName] = userCount
	}

	c.JSON(200, gin.H{
		"total_users":         totalUsers,
		"total_rooms":         len(hub.rooms),
		"room_details":        roomDetails,
		"notification_history": len(hub.notificationHistory),
	})
}

func main() {
	// Start hub in background goroutine
	go hub.run()

	// Create Gin router
	router := gin.Default()

	// WebSocket endpoint for clients
	router.GET("/ws", handleWebSocket)

	// HTTP API endpoints for admin
	router.POST("/api/notify", handleNotification)
	router.GET("/api/stats", getStats)

	fmt.Println("🚀 Notification Server on :8080")
	fmt.Println("📱 WebSocket: ws://localhost:8080/ws")

	// Start server
	router.Run(":8080")
}