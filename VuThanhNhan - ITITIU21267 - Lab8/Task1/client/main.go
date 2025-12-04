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
	
	fmt.Println("✓ Connected to echo server")
	fmt.Println("Type messages and press Enter (Ctrl+C to exit)")
	fmt.Println("---")
	
	// Channel for interrupt signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	
	// Goroutine to read messages from server
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("Connection closed:", err)
				return
			}
			fmt.Printf("Echo: %s\n", message)
		}
	}()
	
	// Read input from user
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		
		err := conn.WriteMessage(websocket.TextMessage, []byte(text))
		if err != nil {
			log.Println("Write error:", err)
			return
		}
	}
	
	// Wait for goroutine to finish
	<-done
}
