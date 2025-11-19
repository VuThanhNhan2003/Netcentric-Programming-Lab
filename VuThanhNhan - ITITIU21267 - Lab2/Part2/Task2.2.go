package main

import (
	"fmt"
	"time"
)

func sender(name string, messages chan string, count int) { // Send 'count' messages
	for i := 1; i <= count; i++ {
		msg := fmt.Sprintf("Message %d from %s", i, name)
		messages <- msg // Send message to channel
		time.Sleep(150 * time.Millisecond)
	}
}

func main() {
	fmt.Println("=== Message Queue ===")

	messages := make(chan string, 10) // Buffered channel with capacity 10

	// Start 3 senders concurrently
	go sender("Alice", messages, 3)
	go sender("Bob", messages, 2)
	go sender("Charlie", messages, 4)

	for i := 0; i < 9; i++ {
		msg := <-messages // Receive message from channel
		fmt.Println(msg)
	}

	fmt.Println("\nAll messages received!")
}
