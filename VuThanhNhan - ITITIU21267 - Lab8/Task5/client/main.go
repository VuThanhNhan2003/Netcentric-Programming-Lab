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
	MsgChat   = "chat"
	MsgSystem = "system"
)

// Message structure
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

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run notification_client.go <username> <room>")
		os.Exit(1)
	}

	username := os.Args[1]
	room := os.Args[2]

	// Connect to server
	u := url.URL{
		Scheme:   "ws",
		Host:     "localhost:8080",
		Path:     "/ws",
		RawQuery: fmt.Sprintf("username=%s&room=%s", username, room),
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer conn.Close()

	fmt.Printf("Connected to room '%s' as '%s'\n", room, username)
	fmt.Println("---")

	// Interrupt signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Done channel
	done := make(chan struct{})

	// Read messages
	go func() {
		defer close(done)
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}

			// Try parse as notification first
			var notif Notification
			if err := json.Unmarshal(data, &notif); err == nil && notif.ID != "" {
				displayNotification(notif)
				continue
			}

			// Parse as regular message
			var msg Message
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}

			// Display based on type
			if msg.Type == MsgChat {
				fmt.Printf("[%s] %s: %s\n", msg.Time, msg.Username, msg.Text)
			} else if msg.Type == MsgSystem {
				fmt.Printf("[%s] * %s\n", msg.Time, msg.Text)
			}
		}
	}()

	// Send messages
	scanner := bufio.NewScanner(os.Stdin)
	go func() {
		for scanner.Scan() {
			text := strings.TrimSpace(scanner.Text())
			if text == "" {
				continue
			}

			msg := Message{Text: text}
			data, _ := json.Marshal(msg)
			conn.WriteMessage(websocket.TextMessage, data)
		}
	}()

	// Wait
	select {
	case <-done:
	case <-interrupt:
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	}
}

// Display notification
func displayNotification(notif Notification) {
	fmt.Println()
	fmt.Println("==========")
	fmt.Printf("[%s] %s\n", strings.ToUpper(notif.Type), notif.Title)
	fmt.Printf("Message: %s\n", notif.Message)
	fmt.Printf("Target: %s | Time: %s\n", notif.Target, notif.Timestamp.Format("15:04:05"))
	fmt.Println("==========")
	fmt.Println()
}