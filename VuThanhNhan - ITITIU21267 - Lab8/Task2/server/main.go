package main

import (
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

// Client represents a WebSocket client
type Client struct {
	ID   string
	Conn *websocket.Conn
	Send chan []byte
}

// Hub manages all connected clients
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// Create new hub instance
func newHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub's main event loop
func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			// Add client to map (thread-safe)
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("Client registered: %s (Total: %d)", client.ID, len(h.clients))

		case client := <-h.unregister:
			// Remove client from map and close channel
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mu.Unlock()
			log.Printf("Client unregistered: %s (Total: %d)", client.ID, len(h.clients))

		case message := <-h.broadcast:
			// Send message to all connected clients
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
					// Message sent successfully
				default:
					// Channel full, client slow/dead - close it
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
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
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			// Check if it's unexpected close error
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error: %v", err)
			}
			break
		}

		// Add sender ID to message
		fullMessage := fmt.Sprintf("[%s]: %s", c.ID, message)
		
		// Broadcast message to all clients via hub
		hub.broadcast <- []byte(fullMessage)
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
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Upgrade failed: %v", err)
		return
	}

	// Create new client with unique ID
	client := &Client{
		ID:   fmt.Sprintf("client-%d", time.Now().Unix()),
		Conn: conn,
		Send: make(chan []byte, 256), // Buffered channel
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

	fmt.Println("🚀 Broadcast Chat Server started on :8080")
	
	// Start server
	router.Run(":8080")
}