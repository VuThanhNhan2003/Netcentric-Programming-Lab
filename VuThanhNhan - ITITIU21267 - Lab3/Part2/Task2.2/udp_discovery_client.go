package main

import (
	"fmt"
	"net"
	"strings"
	"time"
)

// ServiceInfo stores information about discovered services
type ServiceInfo struct {
	Name    string
	Address string
	Port    string
}

// discoverServices - Sends a broadcast DISCOVER and collects responses
func discoverServices() []ServiceInfo {
	broadcastAddr, err := net.ResolveUDPAddr("udp", "255.255.255.255:8083")
	if err != nil {
		fmt.Println("Error resolving broadcast address:", err)
		return nil
	}

	// Create a UDP socket to send and receive responses
	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		fmt.Println("Error creating UDP connection:", err)
		return nil
	}
	defer conn.Close()

	// Enable broadcast
	conn.SetWriteBuffer(1024)
	conn.SetDeadline(time.Now().Add(3 * time.Second))

	// Send DISCOVER message
	_, err = conn.WriteToUDP([]byte("DISCOVER"), broadcastAddr)
	if err != nil {
		fmt.Println("Error sending broadcast:", err)
		return nil
	}

	fmt.Println("Discovering services...")

	var services []ServiceInfo
	buffer := make([]byte, 1024)

	// Receive responses within 3 seconds
	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			break // timeout or no more responses
		}

		response := strings.TrimSpace(string(buffer[:n]))
		parts := strings.Split(response, ":")
		if len(parts) == 4 && parts[0] == "SERVICE" {
			service := ServiceInfo{
				Name:    parts[1],
				Address: parts[2],
				Port:    parts[3],
			}
			services = append(services, service)
		}
	}

	return services
}

func main() {
	fmt.Println("=== UDP Service Discovery Client ===")

	// Run discovery
	services := discoverServices()

	if len(services) == 0 {
		fmt.Println("No services found.")
		return
	}

	// Display results
	fmt.Printf("Found %d services:\n", len(services))
	fmt.Println("Service Name          Address           Port")
	fmt.Println("----------------------------------------------------")
	for _, s := range services {
		fmt.Printf("%-20s %-16s %s\n", s.Name, s.Address, s.Port)
	}
	fmt.Println("\nDiscovery complete!")
}
