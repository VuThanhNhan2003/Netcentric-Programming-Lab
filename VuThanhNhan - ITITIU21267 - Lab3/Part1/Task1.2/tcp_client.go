package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
)

func sendFile(filename string, serverAddr string) {
	// Check if file exists
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Printf("Error getting file info: %v\n", err)
		return
	}
	filesize := fileInfo.Size()

	fmt.Printf("Sending file: %s\n", filename)

	// Connect to server
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		fmt.Printf("Error connecting to server: %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Println("Connected to server")

	// Send metadata: "filename:filesize\n"
	metadata := fmt.Sprintf("%s:%d\n", filepath.Base(filename), filesize)
	_, err = conn.Write([]byte(metadata))
	if err != nil {
		fmt.Printf("Error sending metadata: %v\n", err)
		return
	}

	// Send file in chunks
	buffer := make([]byte, 1024) // 1KB chunks
	var totalSent int64 = 0

	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			return
		}

		// Send chunk
		_, err = conn.Write(buffer[:n])
		if err != nil {
			fmt.Printf("Error sending data: %v\n", err)
			return
		}

		totalSent += int64(n)

		// Display progress
		percentage := float64(totalSent) / float64(filesize) * 100
		fmt.Printf("\rSent: %d/%d bytes (%.2f%%)", totalSent, filesize, percentage)
	}

	fmt.Println() // New line after progress

	// Wait for confirmation
	confirmation, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		fmt.Printf("Error reading confirmation: %v\n", err)
		return
	}

	fmt.Print(confirmation)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run client.go <filename>")
		return
	}

	filename := os.Args[1]
	sendFile(filename, "localhost:8081")
}
