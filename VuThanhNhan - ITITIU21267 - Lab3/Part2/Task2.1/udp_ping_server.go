package main

import (
	"fmt"
	"net"
	"time"
)

// pingServer - Start a UDP server to listen for ping requests
func pingServer(port int, serverID string) {
	// Create UDP address
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		fmt.Printf("Error resolving address for port %d: %v\n", port, err)
		return
	}

	// Listen on UDP port
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Printf("Error listening on port %d: %v\n", port, err)
		return
	}
	defer conn.Close()

	fmt.Printf("Ping server [%s] started on port %d\n", serverID, port)

	buffer := make([]byte, 1024)

	// Infinite loop waiting for ping requests
	for {
		// Read data from client
		n, clientAddr, err := conn.ReadFromUDP(buffer) 
		if err != nil {
			fmt.Printf("Error reading on port %d: %v\n", port, err)
			continue
		}

		message := string(buffer[:n]) // "PING"

		// Check if it is a PING request
		if message == "PING" {
			// Create PONG response with timestamp
			pongMsg := fmt.Sprintf("PONG %s %d", serverID, time.Now().Unix())

			// Send PONG back to client
			_, err = conn.WriteToUDP([]byte(pongMsg), clientAddr)
			if err != nil {
				fmt.Printf("Error sending PONG from port %d: %v\n", port, err)
			}
		}
	}
}

func main() {
	fmt.Println("=== UDP Ping Servers ===")

	// Server configuration
	ports := []int{9001, 9002, 9003}

	// Start ping servers in goroutines
	for i, port := range ports {
		serverID := fmt.Sprintf("Server-%d", i+1)
		go pingServer(port, serverID)
	}

	// Keep the program running indefinitely
	select {}
}
