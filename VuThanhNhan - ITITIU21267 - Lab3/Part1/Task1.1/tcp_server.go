package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
)

type ChatRoom struct {
	clients map[net.Conn]string
	mu      sync.Mutex
}

func NewChatRoom() *ChatRoom {
	return &ChatRoom{
		clients: make(map[net.Conn]string),
	}
}
func (cr *ChatRoom) broadcast(message string, sender net.Conn) {
	// TODO: Send message to all clients except sender
	cr.mu.Lock()
	defer cr.mu.Unlock()

	// Send message to all clients except the sender
	for conn := range cr.clients {
		if conn != sender {
			_, err := conn.Write([]byte(message))
			if err != nil {
				fmt.Printf("Error broadcasting to client: %v\n", err)
			}
		}
	}
}
func (cr *ChatRoom) addClient(conn net.Conn, username string) {
	// TODO: Add client to chat room
	cr.mu.Lock() // Locking the mutex to protect shared data
	defer cr.mu.Unlock() // Ensuring the mutex is unlocked after the function completes

	cr.clients[conn] = username // Add the new client to the map
	fmt.Printf("%s joined the chat\n", username)	

}
func (cr *ChatRoom) removeClient(conn net.Conn) {
	// TODO: Remove client from chat room
	cr.mu.Lock()
	defer cr.mu.Unlock()

	username, exists := cr.clients[conn]
	if !exists {
		return
	}

	delete(cr.clients, conn)
	conn.Close()

	fmt.Printf("%s left the chat\n", username)

}
func handleChatClient(conn net.Conn, cr *ChatRoom) {
	// TODO: Handle individual chat client
	// - Read username
	// - Announce join
	// - Read and broadcast messages
	// - Handle disconnection
	defer cr.removeClient(conn)
	
	// Read username from client
	reader := bufio.NewReader(conn)
	username, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Error reading username: %v\n", err)
		return
	}
	username = strings.TrimSpace(username)
	
	// Add client to chat room
	cr.addClient(conn, username)
	joinMsg := fmt.Sprintf("*** %s joined the chat ***\n", username)
	conn.Write([]byte(joinMsg))
	
	// Read and broadcast messages
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		message := scanner.Text()
		message = strings.TrimSpace(message)
		
		if message == "" {
			continue
		}
		
		fmt.Printf("Broadcasting from %s: %s\n", username, message)
		
		// Format and broadcast message
		broadcastMsg := fmt.Sprintf("[%s]: %s\n", username, message)
		cr.broadcast(broadcastMsg, conn)
	}
	
	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading from %s: %v\n", username, err)
	}
}
func startChatServer() {
	// TODO: Start TCP server on port 9000
	// TODO: Accept connections and handle in goroutines
	chatRoom := NewChatRoom()
	
	// Listen on port 9000
	listener, err := net.Listen("tcp", ":9000")
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		return
	}
	defer listener.Close()
	
	fmt.Println("Chat server listening on :9000")
	
	// Accept connections in a loop
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Error accepting connection: %v\n", err)
			continue
		}
		
		// Handle each client in a separate goroutine
		go handleChatClient(conn, chatRoom)
	}
}
func main() {
	startChatServer()
}
