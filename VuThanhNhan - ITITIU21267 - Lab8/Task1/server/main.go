package main

import (
	"fmt"
	"log"
	"net/http"
	
	"github.com/gorilla/websocket"
)

// Upgrader upgrades HTTP connection to WebSocket
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Handle WebSocket connections
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()
	
	log.Printf("New client connected: %s", conn.RemoteAddr())
	
	// Read messages in a loop
	for {
		// Read message from client
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Client disconnected: %v", err)
			break
		}
		
		log.Printf("Received: %s", message)
		
		// Echo message back to client
		err = conn.WriteMessage(messageType, message)
		if err != nil {
			log.Printf("Failed to write message: %v", err)
			break
		}
		
		log.Printf("Echoed: %s", message)
	}
}

func main() {
	// Route for WebSocket connection
	http.HandleFunc("/ws", handleWebSocket)
	
	fmt.Println("🚀 WebSocket Echo Server started on :8080")
	fmt.Println("📱 Connect using CLI client: go run client/echo_client.go")
	
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("Server failed:", err)
	}
}
