package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

// Message types for different chat events
type Message struct {
	Type     string `json:"type"`     // "join", "leave", "chat", "system"
	Room     string `json:"room"`     // Room name
	Username string `json:"username"` // Sender's username
	Text     string `json:"text"`     // Message content
	Time     string `json:"time"`     // Timestamp HH:MM:SS
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
	rooms      map[string]*Room
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// Create new hub instance
func newHub() *Hub {
	return &Hub{
		rooms:      make(map[string]*Room),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub's main event loop
func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			// Add client to their room
			h.addClientToRoom(client)

		case client := <-h.unregister:
			// Remove client from their room
			h.removeClientFromRoom(client)
		}
	}
}

// addClientToRoom adds a client to specified room (creates room if needed)
func (h *Hub) addClientToRoom(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

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
	room.mu.Unlock()

	log.Printf("Client %s joined room %s (Total: %d)",
		client.Username, client.Room, len(room.Clients))

	// Send join notification to all clients in room
	msg := Message{
		Type:     "system",
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
	h.mu.RUnlock()

	if !exists {
		return
	}

	// Remove client from room
	room.mu.Lock()
	if _, ok := room.Clients[client]; ok {
		delete(room.Clients, client)
		close(client.Send)
	}
	room.mu.Unlock()

	log.Printf("Client %s left room %s (Remaining: %d)",
		client.Username, client.Room, len(room.Clients))

	// Send leave notification to remaining clients in room
	msg := Message{
		Type:     "system",
		Room:     client.Room,
		Username: client.Username,
		Text:     fmt.Sprintf("%s left the room", client.Username),
		Time:     time.Now().Format("15:04:05"),
	}
	h.broadcastToRoom(client.Room, msg)

	// Delete room if empty
	if len(room.Clients) == 0 {
		h.mu.Lock()
		delete(h.rooms, client.Room)
		h.mu.Unlock()
		log.Printf("Deleted empty room: %s", client.Room)
	}
}

// broadcastToRoom sends message to all clients in specified room only
func (h *Hub) broadcastToRoom(roomName string, msg Message) {
	h.mu.RLock()
	room, exists := h.rooms[roomName]
	h.mu.RUnlock()

	if !exists {
		return
	}

	// Marshal message to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return
	}

	// Send to all clients in this room
	room.mu.RLock()
	defer room.mu.RUnlock()

	for client := range room.Clients {
		select {
		case client.Send <- data:
			// Message sent successfully
		default:
			// Channel full, client slow/dead - close it
			close(client.Send)
			delete(room.Clients, client)
		}
	}
}

// readPump reads messages from WebSocket connection
func (c *Client) readPump(hub *Hub) {
	defer func() {
		// Cleanup on exit: unregister and close connection
		hub.unregister <- c
		c.Conn.Close()
	}()

	// Set read deadline - connection times out after 60 seconds
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	// Reset read deadline when pong received (keepalive)
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Continuously read messages
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

		// Set message metadata (server-side, not trusted from client)
		msg.Username = c.Username
		msg.Room = c.Room
		msg.Type = "chat"
		msg.Time = time.Now().Format("15:04:05")

		// Broadcast to room only
		hub.broadcastToRoom(c.Room, msg)
	}
}

// writePump writes messages to WebSocket connection
func (c *Client) writePump() {
	// Create ticker for sending pings every 54 seconds
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			// Set write deadline - 10 seconds to write
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

			if !ok {
				// Channel closed, send close message
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Write text message to client
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			// Send ping to keep connection alive
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
	// Get username and room from URL query parameters
	username := c.Query("username")
	room := c.Query("room")

	// Validate required parameters
	if username == "" || room == "" {
		c.JSON(400, gin.H{"error": "username and room required"})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Upgrade failed: %v", err)
		return
	}

	// Create new client with unique ID
	client := &Client{
		ID:       fmt.Sprintf("%s-%d", username, time.Now().Unix()),
		Username: username,
		Room:     room,
		Conn:     conn,
		Send:     make(chan []byte, 256), // Buffered channel
	}

	// Register client with hub
	hub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump(hub)
}

func main() {
	// Start hub in background goroutine
	go hub.run()

	// Create Gin router
	router := gin.Default()

	// WebSocket endpoint
	router.GET("/ws", handleWebSocket)

	fmt.Println("🚀 Chat Rooms Server started on :8080")
	fmt.Println("📱 Connect using: go run client/room_client.go <username> <room>")
	fmt.Println("Example: go run client/room_client.go Alice general")

	// Start server
	router.Run(":8080")
}