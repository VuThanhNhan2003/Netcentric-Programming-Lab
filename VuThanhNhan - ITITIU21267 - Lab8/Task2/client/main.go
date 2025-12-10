package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/gorilla/websocket"
)

func main() {
	// Connect to WebSocket server
	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", nil)
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer conn.Close()

	fmt.Println("✓ Connected to broadcast chat server")
	fmt.Println("Type messages to broadcast to all clients (Ctrl+C to exit)")
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
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("Connection closed:", err)
				return
			}
			// Display broadcast message
			fmt.Printf("%s\n", message)
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

			// Send message to server
			err := conn.WriteMessage(websocket.TextMessage, []byte(text))
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