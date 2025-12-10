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

// Message types
const (
	MsgChat     = "chat"
	MsgSystem   = "system"
	MsgUserList = "user_list"
	MsgStats    = "stats"
	MsgRooms    = "rooms"
)

// Message structures matching server
type Message struct {
	Type     string `json:"type"`
	Room     string `json:"room"`
	Username string `json:"username"`
	Text     string `json:"text"`
	Time     string `json:"time"`
}

type StatsMessage struct {
	Type        string         `json:"type"`
	TotalUsers  int            `json:"total_users"`
	TotalRooms  int            `json:"total_rooms"`
	RoomDetails map[string]int `json:"room_details"`
}

type UserListMessage struct {
	Type      string   `json:"type"`
	Room      string   `json:"room"`
	UserCount int      `json:"user_count"`
	Users     []string `json:"users"`
}

type RoomsMessage struct {
	Type       string   `json:"type"`
	TotalRooms int      `json:"total_rooms"`
	Rooms      []string `json:"rooms"`
}

func main() {
	// Check command line arguments
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run stats_client.go <username> <room>")
		fmt.Println("Example: go run stats_client.go Alice general")
		os.Exit(1)
	}

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
	fmt.Println("Commands: /users, /stats, /rooms")
	fmt.Println("Type messages to chat (Ctrl+C to exit)")
	fmt.Println("---")

	// Channel for interrupt signal
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

			// Try to determine message type
			var baseMsg map[string]interface{}
			if err := json.Unmarshal(data, &baseMsg); err != nil {
				log.Printf("Failed to parse message: %v", err)
				continue
			}

			msgType, ok := baseMsg["type"].(string)
			if !ok {
				continue
			}

			// Handle different message types
			switch msgType {
			case MsgChat:
				// Regular chat message
				var msg Message
				json.Unmarshal(data, &msg)
				fmt.Printf("[%s] %s: %s\n", msg.Time, msg.Username, msg.Text)

			case MsgSystem:
				// System notification
				var msg Message
				json.Unmarshal(data, &msg)
				fmt.Printf("[%s] * %s\n", msg.Time, msg.Text)

			case MsgUserList:
				// User list response
				var msg UserListMessage
				json.Unmarshal(data, &msg)
				fmt.Printf("\n=== Users in '%s' (%d) ===\n", msg.Room, msg.UserCount)
				for _, user := range msg.Users {
					fmt.Printf("  - %s\n", user)
				}
				fmt.Println()

			case MsgStats:
				// Statistics response
				var msg StatsMessage
				json.Unmarshal(data, &msg)
				fmt.Println("\n=== Statistics ===")
				fmt.Printf("Total Users: %d\n", msg.TotalUsers)
				fmt.Printf("Total Rooms: %d\n", msg.TotalRooms)
				fmt.Println("\nRoom Details:")
				for roomName, count := range msg.RoomDetails {
					fmt.Printf("  %s: %d users\n", roomName, count)
				}
				fmt.Println()

			case MsgRooms:
				// Rooms list response
				var msg RoomsMessage
				json.Unmarshal(data, &msg)
				fmt.Printf("\n=== All Rooms (%d) ===\n", msg.TotalRooms)
				for _, roomName := range msg.Rooms {
					fmt.Printf("  - %s\n", roomName)
				}
				fmt.Println()
			}
		}
	}()

	// Read input from user and send to server
	scanner := bufio.NewScanner(os.Stdin)
	go func() {
		for scanner.Scan() {
			text := strings.TrimSpace(scanner.Text())
			if text == "" {
				continue
			}

			// Create message (commands or regular chat)
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
		fmt.Println("\nServer closed the connection")
	case <-interrupt:
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