package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

type ChatClient struct {
	username   string
	tcpConn    net.Conn
	udpConn    *net.UDPConn
	serverAddr *net.UDPAddr
}

func NewChatClient() *ChatClient {
	return &ChatClient{}
}

// Connect to TCP server
func (c *ChatClient) connectTCP() error {
	conn, err := net.Dial("tcp", "localhost:9000")
	if err != nil {
		return fmt.Errorf("cannot connect TCP: %v", err)
	}
	c.tcpConn = conn
	return nil
}

// Connect to UDP server
func (c *ChatClient) connectUDP() error {
	serverAddr, err := net.ResolveUDPAddr("udp", "localhost:9001")
	if err != nil {
		return fmt.Errorf("cannot resolve UDP address: %v", err)
	}
	c.serverAddr = serverAddr

	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		return fmt.Errorf("cannot connect UDP: %v", err)
	}
	c.udpConn = conn
	return nil
}

// Receive messages from TCP server
func (c *ChatClient) receiveTCPMessages() {
	reader := bufio.NewReader(c.tcpConn)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("\n*** Lost connection to server ***")
			os.Exit(0)
		}
		// Print messages from server (without prompt)
		message = strings.TrimRight(message, "\n")
		if message != "" {
			fmt.Println(message)
			// Print prompt after receiving message (unless it's a system message)
			if !strings.HasPrefix(message, "***") && !strings.HasPrefix(message, "Enter") {
				fmt.Print("> ")
			}
		}
	}
}

// Receive status updates from UDP server
func (c *ChatClient) receiveUDPStatus() {
	buffer := make([]byte, 1024)
	for {
		n, err := c.udpConn.Read(buffer)
		if err != nil {
			continue
		}

		data := string(buffer[:n])
		parts := strings.Split(data, ":")

		if len(parts) >= 3 && parts[0] == "STATUS" {
			username := parts[1]
			status := parts[2]
			fmt.Printf("\nStatus Update: %s is %s\n", username, status)
			fmt.Print("> ")
		}
	}
}

// Send status via UDP
func (c *ChatClient) sendStatus(status string) {
	statusMsg := fmt.Sprintf("STATUS:%s:%s", c.username, status)
	c.udpConn.Write([]byte(statusMsg))
}

// Heartbeat - periodically send online status
func (c *ChatClient) startHeartbeat() {
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for range ticker.C {
			c.sendStatus("online")
		}
	}()
}

// Send message via TCP
func (c *ChatClient) sendMessage(message string) {
	c.tcpConn.Write([]byte(message + "\n"))
}

func (c *ChatClient) Run() {
	// Connect TCP
	if err := c.connectTCP(); err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer c.tcpConn.Close()

	// Connect UDP
	if err := c.connectUDP(); err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer c.udpConn.Close()

	fmt.Println("Connected to chat server (TCP: 9000, UDP: 9001)")

	// Read prompt from server and username from stdin
	reader := bufio.NewReader(c.tcpConn)
	stdinReader := bufio.NewReader(os.Stdin)

	// Read "Enter username: " prompt from server
	prompt, err := reader.ReadString(':')
	if err != nil {
		fmt.Println("Error reading prompt:", err)
		return
	}
	// Read whitespace after colon
	reader.ReadByte()

	fmt.Print(prompt + " ")

	// Read username from stdin
	username, err := stdinReader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading username:", err)
		return
	}
	c.username = strings.TrimSpace(username)

	// Send username to server
	c.tcpConn.Write([]byte(c.username + "\n"))

	// Read join message from server
	joinMsg, _ := reader.ReadString('\n')
	fmt.Print(joinMsg)

	// Send initial online status
	c.sendStatus("online")

	// Start heartbeat
	c.startHeartbeat()

	// Start receiving TCP messages
	go c.receiveTCPMessages()

	// Start receiving UDP status updates
	go c.receiveUDPStatus()

	// Wait a bit to receive message history
	time.Sleep(200 * time.Millisecond)

	// Message sending loop
	var lastTypingTime time.Time
	typingNotified := false

	fmt.Print("> ")
	for {
		input, err := stdinReader.ReadString('\n')
		if err != nil {
			break
		}

		input = strings.TrimSpace(input)

		if input == "exit" {
			c.sendStatus("away")
			fmt.Println("Exited chat")
			break
		}

		if input == "" {
			fmt.Print("> ")
			continue
		}

		// If user is typing but hasn't sent a message
		if time.Since(lastTypingTime) > 2*time.Second {
			c.sendStatus("typing")
			typingNotified = true
		}
		lastTypingTime = time.Now()

		// Send message
		c.sendMessage(input)

		// Reset status to online after sending
		if typingNotified {
			time.Sleep(100 * time.Millisecond)
			c.sendStatus("online")
			typingNotified = false
		}
	}
}

func main() {
	client := NewChatClient()
	client.Run()
}
