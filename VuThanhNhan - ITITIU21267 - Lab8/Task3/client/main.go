package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"

	"github.com/gorilla/websocket"
)

// Message structure matching server
type Message struct {
	Type     string `json:"type"`
	Room     string `json:"room"`
	Username string `json:"username"`
	Text     string `json:"text"`
	Time     string `json:"time"`
}

func main() {
	// Check command line arguments
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run room_client.go <username> <room>")
		fmt.Println("Example: go run room_client.go Alice general")
		os.Exit(1)
	}

	// Get username and room from command line arguments
	username := os.Args[1]
	room := os.Args[2]

	// Build WebSocket URL with query parameters
	u := url.URL{
		Scheme:   "ws",
		Host:     "localhost:8080",
		Path:     "/ws",
		RawQuery: fmt.Sprintf("username=%s&room=%s", username, room),
	}

	// Connect to WebSocket server
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer conn.Close()

	fmt.Printf("✓ Connected to room '%s' as '%s'\n", room, username)
	fmt.Println("Type messages and press Enter (Ctrl+C to exit)")
	fmt.Println("---")

	// Channel for interrupt signal (Ctrl+C)
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Channel to signal when reading is done
	done := make(chan struct{})

	// Goroutine to read messages from server
	go func() {
		defer close(done)
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				log.Println("Connection closed:", err)
				return
			}

			// Parse JSON message
			var msg Message
			if err := json.Unmarshal(data, &msg); err != nil {
				log.Printf("Failed to parse message: %v", err)
				continue
			}

			// Display message based on type
			switch msg.Type {
			case "chat":
				// Regular chat message
				fmt.Printf("[%s] %s: %s\n", msg.Time, msg.Username, msg.Text)
			case "system":
				// System notification (join/leave)
				fmt.Printf("[%s] * %s\n", msg.Time, msg.Text)
			}
		}
	}()

	// Read input from user and send to server
	scanner := bufio.NewScanner(os.Stdin)
	go func() {
		for scanner.Scan() {
			text := strings.TrimSpace(scanner.Text())
			if text == "" {
				continue // Skip empty messages
			}

			// Create message (only Text field needed, server sets others)
			msg := Message{
				Text: text,
			}

			// Marshal to JSON
			data, err := json.Marshal(msg)
			if err != nil {
				log.Printf("Failed to marshal message: %v", err)
				continue
			}

			// Send message to server
			err = conn.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				log.Println("Write error:", err)
				return
			}
		}
	}()

	// Wait for interrupt signal or connection close
	select {
	case <-done:
		// Connection closed by server
		fmt.Println("\nServer closed the connection")
	case <-interrupt:
		// User pressed Ctrl+C - graceful shutdown
		fmt.Println("\nShutting down gracefully...")
		err := conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		)
		if err != nil {
			log.Println("Write close error:", err)
		}
	}
}
