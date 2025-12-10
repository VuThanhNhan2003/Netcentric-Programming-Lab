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
	MsgChat     = "chat"
	MsgSystem   = "system"
	MsgUserList = "user_list"
	MsgStats    = "stats"
	MsgCommand  = "command"
	MsgRooms    = "rooms"
)

// Message structure for chat events
type Message struct {
	Type     string `json:"type"`
	Room     string `json:"room"`
	Username string `json:"username"`
	Text     string `json:"text"`
	Time     string `json:"time"`
}

// StatsMessage structure for statistics
type StatsMessage struct {
	Type        string         `json:"type"`
	TotalUsers  int            `json:"total_users"`
	TotalRooms  int            `json:"total_rooms"`
	RoomDetails map[string]int `json:"room_details"` // room -> user count
}

// UserListMessage structure for user list
type UserListMessage struct {
	Type      string   `json:"type"`
	Room      string   `json:"room"`
	UserCount int      `json:"user_count"`
	Users     []string `json:"users"`
}

// RoomsMessage structure for room list
type RoomsMessage struct {
	Type       string   `json:"type"`
	TotalRooms int      `json:"total_rooms"`
	Rooms      []string `json:"rooms"`
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
			h.addClientToRoom(client)
		case client := <-h.unregister:
			h.removeClientFromRoom(client)
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

	// Send join notification to all clients in room
	msg := Message{
		Type:     MsgSystem,
		Room:     client.Room,
		Username: client.Username,
		Text:     fmt.Sprintf("%s joined the room", client.Username),
		Time:     time.Now().Format("15:04:05"),
	}
	h.broadcastToRoom(client.Room, msg)

	// Send updated user count to room
	h.sendUserCountUpdate(client.Room)
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

	// Send leave notification to remaining clients in room
	msg := Message{
		Type:     MsgSystem,
		Room:     client.Room,
		Username: client.Username,
		Text:     fmt.Sprintf("%s left the room", client.Username),
		Time:     time.Now().Format("15:04:05"),
	}
	h.broadcastToRoom(client.Room, msg)

	// Send updated user count to room
	h.sendUserCountUpdate(client.Room)

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
		default:
			close(client.Send)
			delete(room.Clients, client)
		}
	}
}

// sendUserCountUpdate sends updated user count to all clients in room
func (h *Hub) sendUserCountUpdate(roomName string) {
	h.mu.RLock()
	room, exists := h.rooms[roomName]
	h.mu.RUnlock()

	if !exists {
		return
	}

	room.mu.RLock()
	userCount := len(room.Clients)
	room.mu.RUnlock()

	// Send system message with user count
	msg := Message{
		Type: MsgSystem,
		Room: roomName,
		Text: fmt.Sprintf("Users online: %d", userCount),
		Time: time.Now().Format("15:04:05"),
	}

	h.broadcastToRoom(roomName, msg)
}

// handleCommand processes commands like /users, /stats, /rooms
func (h *Hub) handleCommand(client *Client, cmd string) {
	cmd = strings.TrimSpace(strings.ToLower(cmd))

	switch cmd {
	case "/users":
		// Send user list for client's room
		h.sendUserList(client)

	case "/stats":
		// Send global statistics
		h.sendStats(client)

	case "/rooms":
		// Send list of all rooms
		h.sendRooms(client)

	default:
		// Unknown command
		msg := Message{
			Type: MsgSystem,
			Room: client.Room,
			Text: fmt.Sprintf("Unknown command: %s. Available: /users, /stats, /rooms", cmd),
			Time: time.Now().Format("15:04:05"),
		}
		data, _ := json.Marshal(msg)
		client.Send <- data
	}
}

// sendUserList sends list of users in client's room
func (h *Hub) sendUserList(client *Client) {
	h.mu.RLock()
	room, exists := h.rooms[client.Room]
	h.mu.RUnlock()

	if !exists {
		return
	}

	// Collect usernames
	room.mu.RLock()
	users := make([]string, 0, len(room.Clients))
	for c := range room.Clients {
		users = append(users, c.Username)
	}
	room.mu.RUnlock()

	// Create user list message
	userListMsg := UserListMessage{
		Type:      MsgUserList,
		Room:      client.Room,
		UserCount: len(users),
		Users:     users,
	}

	// Marshal and send
	data, err := json.Marshal(userListMsg)
	if err != nil {
		log.Printf("Failed to marshal user list: %v", err)
		return
	}

	client.Send <- data
}

// sendStats sends global statistics to client
func (h *Hub) sendStats(client *Client) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Calculate total users and room details
	totalUsers := 0
	roomDetails := make(map[string]int)

	for roomName, room := range h.rooms {
		room.mu.RLock()
		userCount := len(room.Clients)
		room.mu.RUnlock()

		totalUsers += userCount
		roomDetails[roomName] = userCount
	}

	// Create stats message
	statsMsg := StatsMessage{
		Type:        MsgStats,
		TotalUsers:  totalUsers,
		TotalRooms:  len(h.rooms),
		RoomDetails: roomDetails,
	}

	// Marshal and send
	data, err := json.Marshal(statsMsg)
	if err != nil {
		log.Printf("Failed to marshal stats: %v", err)
		return
	}

	client.Send <- data
}

// sendRooms sends list of all rooms to client
func (h *Hub) sendRooms(client *Client) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Collect room names
	rooms := make([]string, 0, len(h.rooms))
	for roomName := range h.rooms {
		rooms = append(rooms, roomName)
	}

	// Create rooms message
	roomsMsg := RoomsMessage{
		Type:       MsgRooms,
		TotalRooms: len(rooms),
		Rooms:      rooms,
	}

	// Marshal and send
	data, err := json.Marshal(roomsMsg)
	if err != nil {
		log.Printf("Failed to marshal rooms: %v", err)
		return
	}

	client.Send <- data
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

		// Check if message is a command
		if strings.HasPrefix(msg.Text, "/") {
			hub.handleCommand(c, msg.Text)
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
	
	// Small delay to ensure goroutines are ready
	time.Sleep(10 * time.Millisecond)
	
	// Register client to hub (this will send join notification)
	hub.register <- client
}

func main() {
	go hub.run()

	router := gin.Default()
	router.GET("/ws", handleWebSocket)

	fmt.Println("🚀 Chat Server with Statistics on :8080")
	fmt.Println("📱 Commands: /users, /stats, /rooms")
	fmt.Println("Example: go run client/stats_client.go Alice general")

	router.Run(":8080")
}