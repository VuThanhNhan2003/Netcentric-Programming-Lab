package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// File Transfer Server
func fileTransferServer() {
	// Listen on port 8081
	listener, err := net.Listen("tcp", ":8081")
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		return
	}
	defer listener.Close()

	fmt.Println("File Transfer Server listening on :8081")

	// Create received directory if it doesn't exist
	os.MkdirAll("./received", 0755) // permissions rwxr-xr-x

	for {
		// Accept incoming connection
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Error accepting connection: %v\n", err)
			continue
		}

		// Handle each file transfer in a goroutine
		go handleFileTransfer(conn)
	}
}

func handleFileTransfer(conn net.Conn) {
	defer conn.Close()

	fmt.Println("Client connected")

	// Read file metadata: "FILENAME:filesize"
	reader := bufio.NewReader(conn)
	metadata, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Error reading metadata: %v\n", err)
		return
	}

	// Parse metadata
	metadata = strings.TrimSpace(metadata)
	parts := strings.Split(metadata, ":")
	if len(parts) != 2 {
		fmt.Println("Invalid metadata format")
		return
	}

	filename := parts[0]
	filesize, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		fmt.Printf("Error parsing file size: %v\n", err)
		return
	}

	fmt.Printf("Receiving file: %s (%d bytes)\n", filename, filesize)

	// Create file in received directory
	savePath := filepath.Join("./received", filename)
	file, err := os.Create(savePath)
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer file.Close()

	// Receive file data
	written, err := io.CopyN(file, reader, filesize)
	if err != nil {
		fmt.Printf("Error receiving file: %v\n", err)
		return
	}

	fmt.Printf("File saved successfully to %s (%d bytes written)\n", savePath, written)

	// Send confirmation to client
	confirmation := "✓ File transfer complete!\n"
	conn.Write([]byte(confirmation))
}

func main() {
	fileTransferServer()
}
