package main

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// PingResult stores the ping result for each server
type PingResult struct {
	ServerAddr string        // Server address (e.g., localhost:9001)
	RTT        time.Duration // Round-Trip Time
	Success    bool          // Ping succeeded or timed out
}

// pingOnce - Ping a server once and measure RTT
func pingOnce(serverAddr string, timeout time.Duration) PingResult {
	result := PingResult{
		ServerAddr: serverAddr,
		Success:    false,
	}

	// Resolve server address
	addr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		return result
	}

	// Create UDP connection
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return result
	}
	defer conn.Close()

	// Start measuring time
	startTime := time.Now()

	// Send PING message
	_, err = conn.Write([]byte("PING"))
	if err != nil {
		return result
	}

	// Set read timeout for response
	conn.SetReadDeadline(time.Now().Add(timeout))

	// Wait for PONG response
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		// Timeout or other error
		return result
	}

	// Calculate RTT (Round-Trip Time)
	rtt := time.Since(startTime)

	// Check if response starts with "PONG"
	response := string(buffer[:n])
	if len(response) >= 4 && response[:4] == "PONG" {
		result.RTT = rtt
		result.Success = true
	}

	return result
}

// pingMonitor - Ping all servers and display results
func pingMonitor(servers []string) {
	timeout := 1 * time.Second

	for {
		fmt.Println("\nPinging servers...")

		// Channel to receive results from goroutines
		resultsChan := make(chan PingResult, len(servers))
		var wg sync.WaitGroup

		// Ping all servers concurrently
		for _, serverAddr := range servers {
			wg.Add(1)
			go func(addr string) {
				defer wg.Done()
				result := pingOnce(addr, timeout)
				resultsChan <- result
			}(serverAddr)
		}

		// Wait for all goroutines to finish
		wg.Wait()
		close(resultsChan)

		// Collect results
		var results []PingResult
		for result := range resultsChan {
			results = append(results, result)
		}

		// Display results table
		fmt.Println("\nServer                  Status         RTT")
		fmt.Println("--------------------------------------------------")

		var totalRTT time.Duration
		successCount := 0

		for _, result := range results {
			status := "✗ Timeout"
			rttStr := "-"

			if result.Success {
				status = "✓ Online"
				rttStr = fmt.Sprintf("%.1fms", float64(result.RTT.Microseconds())/1000.0)
				totalRTT += result.RTT
				successCount++
			}

			fmt.Printf("%-23s %-14s %s\n", result.ServerAddr, status, rttStr)
		}

		// Calculate and display statistics
		fmt.Println()
		if successCount > 0 {
			avgRTT := totalRTT / time.Duration(successCount)
			fmt.Printf("Average RTT: %.1fms\n", float64(avgRTT.Microseconds())/1000.0)
		} else {
			fmt.Println("Average RTT: N/A (all servers timeout)")
		}

		successRate := float64(successCount) / float64(len(servers)) * 100
		fmt.Printf("Success Rate: %.2f%% (%d/%d)\n", successRate, successCount, len(servers))

		// Wait 2 seconds before next ping
		time.Sleep(2 * time.Second)
	}
}

func main() {
	fmt.Println("=== UDP Ping Monitor ===\n")

	// Server configuration
	servers := []string{
		"localhost:9001",
		"localhost:9002",
		"localhost:9003",
	}

	// Start monitoring
	pingMonitor(servers)
}
