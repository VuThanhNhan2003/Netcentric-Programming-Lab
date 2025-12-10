package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// Notification structure
type Notification struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Target    string    `json:"target"`
}

const serverURL = "http://localhost:8080/api/notify"

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run admin.go <command> [args]")
		fmt.Println("Commands:")
		fmt.Println("  broadcast \"message\"")
		fmt.Println("  room <roomname> \"message\"")
		fmt.Println("  user <username> \"message\"")
		fmt.Println("  announce \"title\" \"message\"")
		os.Exit(1)
	}

	command := os.Args[1]
	var notif Notification

	switch command {
	case "broadcast":
		if len(os.Args) < 3 {
			log.Fatal("Usage: broadcast \"message\"")
		}
		notif = Notification{
			Type:    "info",
			Title:   "Broadcast",
			Message: os.Args[2],
			Target:  "all",
		}

	case "room":
		if len(os.Args) < 4 {
			log.Fatal("Usage: room <roomname> \"message\"")
		}
		notif = Notification{
			Type:    "info",
			Title:   "Room: " + os.Args[2],
			Message: os.Args[3],
			Target:  "room:" + os.Args[2],
		}

	case "user":
		if len(os.Args) < 4 {
			log.Fatal("Usage: user <username> \"message\"")
		}
		notif = Notification{
			Type:    "info",
			Title:   "Private Message",
			Message: os.Args[3],
			Target:  "user:" + os.Args[2],
		}

	case "announce":
		if len(os.Args) < 4 {
			log.Fatal("Usage: announce \"title\" \"message\"")
		}
		notif = Notification{
			Type:    "warning",
			Title:   os.Args[2],
			Message: os.Args[3],
			Target:  "all",
		}

	default:
		log.Fatalf("Unknown command: %s", command)
	}

	// Send notification
	data, _ := json.Marshal(notif)
	resp, err := http.Post(serverURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Fatalf("Failed to send: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		log.Fatalf("Server error: %s", string(body))
	}

	fmt.Println("Notification sent successfully")
}