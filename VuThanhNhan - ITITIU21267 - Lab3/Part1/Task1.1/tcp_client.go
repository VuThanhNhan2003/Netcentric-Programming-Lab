package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func chatRoomClient(username string) {
	// Connect to server on port 9000
	conn, err := net.Dial("tcp", "localhost:9000")
	if err != nil {
		fmt.Printf("Error connecting to server: %v\n", err)
		return
	}
	defer conn.Close()
	
	// Send username to server
	fmt.Fprintf(conn, "%s\n", username)
	
	// Start goroutine to read messages from server
	go func() {
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			message := scanner.Text()
			fmt.Println(message)
		}
		if err := scanner.Err(); err != nil {
			fmt.Printf("Connection closed: %v\n", err)
		}
		os.Exit(0)
	}()
	
	// Read user input and send to server
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		message := scanner.Text()
		message = strings.TrimSpace(message)
		
		if message == "exit" {
			fmt.Println("Leaving chat...")
			break
		}
		
		if message == "" {
			continue
		}
		
		// Send message to server
		fmt.Fprintf(conn, "%s\n", message)
	}
}

func main() {
	// Prompt for username
	fmt.Print("Enter username: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	username := strings.TrimSpace(scanner.Text())
	
	if username == "" {
		fmt.Println("Username cannot be empty")
		return
	}
	
	chatRoomClient(username)
}