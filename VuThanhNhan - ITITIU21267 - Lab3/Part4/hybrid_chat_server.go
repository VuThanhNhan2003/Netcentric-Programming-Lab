package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

type Message struct {
	Username  string
	Content   string
	Timestamp time.Time
}

type UserStatus struct {
	Username string
	Status   string // "online", "typing", "away"
	LastSeen time.Time
}

type HybridChatServer struct {
	tcpClients     map[net.Conn]string
	messageHistory []Message
	userStatuses   map[string]*UserStatus
	mu             sync.RWMutex
	udpConn        *net.UDPConn
	clientAddrs    map[string]*net.UDPAddr
}

func NewHybridChatServer() *HybridChatServer {
	return &HybridChatServer{
		tcpClients:     make(map[net.Conn]string),
		messageHistory: make([]Message, 0, 100),
		userStatuses:   make(map[string]*UserStatus),
		clientAddrs:    make(map[string]*net.UDPAddr),
	}
}

// Add a message to history (max 100 messages)
func (s *HybridChatServer) addMessage(username, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	msg := Message{
		Username:  username,
		Content:   content,
		Timestamp: time.Now(),
	}

	s.messageHistory = append(s.messageHistory, msg)

	// Keep only the 100 most recent messages
	if len(s.messageHistory) > 100 {
		s.messageHistory = s.messageHistory[1:]
	}
}

// Update user status
func (s *HybridChatServer) updateStatus(username, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.userStatuses[username] = &UserStatus{
		Username: username,
		Status:   status,
		LastSeen: time.Now(),
	}
}

// Broadcast a message to all TCP clients
func (s *HybridChatServer) broadcastMessage(msg Message) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	timestamp := msg.Timestamp.Format("15:04:05")
	formatted := fmt.Sprintf("[%s] %s: %s\n", timestamp, msg.Username, msg.Content)

	for conn := range s.tcpClients {
		conn.Write([]byte(formatted))
	}
}

// Broadcast status updates to all UDP clients
func (s *HybridChatServer) broadcastStatus(username, status string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	statusMsg := fmt.Sprintf("STATUS:%s:%s", username, status)

	for user, addr := range s.clientAddrs {
		if user != username { // Do not send back to the sender
			s.udpConn.WriteToUDP([]byte(statusMsg), addr)
		}
	}
}

// Handle a TCP client connection
func (s *HybridChatServer) handleTCPClient(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Read username
	conn.Write([]byte("Enter username: "))
	username, err := reader.ReadString('\n')
	if err != nil {
		return
	}
	username = strings.TrimSpace(username)

	// Save client
	s.mu.Lock()
	s.tcpClients[conn] = username
	s.mu.Unlock()

	fmt.Printf("%s connected via TCP\n", username)
	conn.Write([]byte("*** Joined chat room ***\n"))

	// Send message history
	s.mu.RLock()
	for _, msg := range s.messageHistory {
		timestamp := msg.Timestamp.Format("15:04:05")
		formatted := fmt.Sprintf("[%s] %s: %s\n", timestamp, msg.Username, msg.Content)
		conn.Write([]byte(formatted))
	}
	s.mu.RUnlock()

	// Read messages from client
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		message = strings.TrimSpace(message)
		if message == "" {
			continue
		}

		// Handle special commands
		if strings.HasPrefix(message, "HISTORY:") {
			// Message history already sent on connection
			continue
		}

		// Save and broadcast message
		fmt.Printf("[TCP] %s: %s\n", username, message)
		s.addMessage(username, message)

		msg := Message{
			Username:  username,
			Content:   message,
			Timestamp: time.Now(),
		}
		s.broadcastMessage(msg)
	}

	// Remove client on disconnect
	s.mu.Lock()
	delete(s.tcpClients, conn)
	s.mu.Unlock()

	fmt.Printf("%s disconnected\n", username)
}

// Start TCP server
func (s *HybridChatServer) startTCPServer() {
	listener, err := net.Listen("tcp", ":9000")
	if err != nil {
		fmt.Println("Error starting TCP server:", err)
		return
	}
	defer listener.Close()

	fmt.Println("TCP Server listening on :9000")

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go s.handleTCPClient(conn)
	}
}

// Start UDP server
func (s *HybridChatServer) startUDPServer() {
	addr, err := net.ResolveUDPAddr("udp", ":9001")
	if err != nil {
		fmt.Println("Error resolving UDP address:", err)
		return
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println("Error starting UDP server:", err)
		return
	}
	defer conn.Close()

	s.udpConn = conn
	fmt.Println("UDP Server listening on :9001")

	buffer := make([]byte, 1024)

	for {
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			continue
		}

		data := string(buffer[:n])
		parts := strings.Split(data, ":")

		if len(parts) >= 2 && parts[0] == "STATUS" {
			username := parts[1]
			status := "online"
			if len(parts) >= 3 {
				status = parts[2]
			}

			// Save client address
			s.mu.Lock()
			s.clientAddrs[username] = clientAddr
			s.mu.Unlock()

			// Update status
			s.updateStatus(username, status)
			fmt.Printf("[UDP] %s: %s\n", username, status)

			// Broadcast status
			s.broadcastStatus(username, status)
		}
	}
}

func main() {
	fmt.Println("=== Hybrid Chat Server ===")
	server := NewHybridChatServer()

	// Start both servers
	go server.startTCPServer()
	go server.startUDPServer()

	// Keep server running
	select {}
}
