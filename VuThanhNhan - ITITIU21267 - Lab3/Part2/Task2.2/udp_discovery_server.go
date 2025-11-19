package main

import (
	"fmt"
	"net"
	"strings"
)

// ServiceInfo stores service information
type ServiceInfo struct {
	Name string
	Port int
}

// discoveryServer - Listens for "DISCOVER" requests and responds with all service information
func discoveryServer(services []ServiceInfo) {
	addr, err := net.ResolveUDPAddr("udp", ":8083")
	if err != nil {
		fmt.Println("Error resolving UDP address:", err)
		return
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println("Error listening on UDP:", err)
		return
	}
	defer conn.Close()

	hostAddr := getLocalIP()
	fmt.Printf("Discovery server listening on port 8083\n")

	buffer := make([]byte, 1024)

	for {
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Error reading UDP:", err)
			continue
		}

		message := strings.TrimSpace(string(buffer[:n]))
		if message == "DISCOVER" {
			for _, svc := range services {
				response := fmt.Sprintf("SERVICE:%s:%s:%d", svc.Name, hostAddr, svc.Port)
				conn.WriteToUDP([]byte(response), clientAddr)
				fmt.Printf("Responded to discovery from %s for service '%s'\n", clientAddr, svc.Name)
			}
		}
	}
}

// getLocalIP - Get the local machine's IP address
func getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80") // use Google's DNS
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

func main() {
	fmt.Println("=== UDP Discovery Server ===")

	// List of services
	services := []ServiceInfo{
		{"Database Service", 5432},
		{"Web Service", 8080},
		{"API Service", 3000},
	}

	// Run only one discovery server
	discoveryServer(services)
}
