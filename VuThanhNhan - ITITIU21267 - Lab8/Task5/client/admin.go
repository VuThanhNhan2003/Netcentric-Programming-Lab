package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Notification structure matching server
type Notification struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Target    string    `json:"target"`
}

const serverURL = "http://localhost:8080"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "broadcast":
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run admin.go broadcast \"message\"")
			os.Exit(1)
		}
		sendNotification(Notification{
			Type:    "info",
			Title:   "Broadcast",
			Message: os.Args[2],
			Target:  "all",
		})

	case "room":
		if len(os.Args) < 4 {
			fmt.Println("Usage: go run admin.go room <room_name> \"message\"")
			os.Exit(1)
		}
		sendNotification(Notification{
			Type:    "info",
			Title:   "Room Notification",
			Message: os.Args[3],
			Target:  fmt.Sprintf("room:%s", os.Args[2]),
		})

	case "user":
		if len(os.Args) < 4 {
			fmt.Println("Usage: go run admin.go user <username> \"message\"")
			os.Exit(1)
		}
		sendNotification(Notification{
			Type:    "info",
			Title:   "Private Message",
			Message: os.Args[3],
			Target:  fmt.Sprintf("user:%s", os.Args[2]),
		})

	case "announce":
		if len(os.Args) < 4 {
			fmt.Println("Usage: go run admin.go announce \"title\" \"message\"")
			os.Exit(1)
		}
		sendNotification(Notification{
			Type:    "success",
			Title:   os.Args[2],
			Message: os.Args[3],
			Target:  "all",
		})

	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: go run admin.go <command> [arguments]")
	fmt.Println("\nCommands:")
	fmt.Println("  broadcast \"message\"")
	fmt.Println("  room <room_name> \"message\"")
	fmt.Println("  user <username> \"message\"")
	fmt.Println("  announce \"title\" \"message\"")
}

func sendNotification(notif Notification) {
	jsonData, err := json.Marshal(notif)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	resp, err := http.Post(
		serverURL+"/api/notify",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if resp.StatusCode != 200 {
		fmt.Printf("Error: %s\n", string(body))
		return
	}

	var result map[string]interface{}
	json.Unmarshal(body, &result)
	fmt.Printf("Notification sent: %s\n", result["id"])
}