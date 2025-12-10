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
	"time"

	"github.com/gorilla/websocket"
)

// Message types
const (
	MsgChat         = "chat"
	MsgSystem       = "system"
	MsgNotification = "notification"
)

// Message structure for chat
type Message struct {
	Type     string `json:"type"`
	Room     string `json:"room"`
	Username string `json:"username"`
	Text     string `json:"text"`
	Time     string `json:"time"`
}

// Notification structure
type Notification struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Target    string    `json:"target"`
}

// ANSI color codes for terminal
const (
	ColorReset   = "\033[0m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorCyan    = "\033[36m"
	ColorWhite   = "\033[37m"
	ColorBold    = "\033[1m"
)

func main() {
	// Check command line arguments
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run notification_client.go <username> <room>")
		fmt.Println("Example: go run notification_client.go Alice general")
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

	fmt.Printf("%s✓ Connected to room '%s' as '%s'%s\n", ColorGreen, room, username, ColorReset)
	fmt.Println("Type messages to chat (Ctrl+C to exit)")
	fmt.Println(strings.Repeat("-", 60))

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
				// Check if it's a notification (has 'id' field)
				if _, hasID := baseMsg["id"]; hasID {
					msgType = MsgNotification
				} else {
					continue
				}
			}

			// Handle different message types
			switch msgType {
			case MsgChat:
				// Regular chat message
				var msg Message
				json.Unmarshal(data, &msg)
				fmt.Printf("%s[%s] %s:%s %s\n", ColorCyan, msg.Time, msg.Username, ColorReset, msg.Text)

			case MsgSystem:
				// System notification
				var msg Message
				json.Unmarshal(data, &msg)
				fmt.Printf("%s[%s] * %s%s\n", ColorBlue, msg.Time, msg.Text, ColorReset)

			case MsgNotification:
				// Admin notification
				var notif Notification
				json.Unmarshal(data, &notif)
				displayNotification(notif)
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

			// Create message
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

// displayNotification displays notification with color coding
func displayNotification(notif Notification) {
	// Print separator
	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))

	// Choose color based on notification type
	var color string
	var icon string

	switch notif.Type {
	case "info":
		color = ColorBlue
		icon = "ℹ"
	case "warning":
		color = ColorYellow
		icon = "⚠"
	case "error":
		color = ColorRed
		icon = "✖"
	case "success":
		color = ColorGreen
		icon = "✓"
	default:
		color = ColorWhite
		icon = "•"
	}

	// Display notification
	fmt.Printf("%s%s%s %s[%s]%s %s%s%s\n",
		color, ColorBold, icon,
		ColorReset, strings.ToUpper(notif.Type), ColorReset,
		color, notif.Title, ColorReset)

	fmt.Printf("%sMessage:%s %s\n", ColorBold, ColorReset, notif.Message)

	// Display metadata
	fmt.Printf("%sTarget:%s %s | %sTime:%s %s\n",
		ColorBold, ColorReset, notif.Target,
		ColorBold, ColorReset, notif.Timestamp.Format("15:04:05"))

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()
}