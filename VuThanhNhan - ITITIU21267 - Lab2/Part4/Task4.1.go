package main

import (
	"fmt"
	"time"
)

func searchEngineA(query string, ch chan string) { // ch is channel to send result
	time.Sleep(300 * time.Millisecond) // Simulate search time
	result := fmt.Sprintf("Results from Engine A for '%s'", query)
	ch <- result // Send result to channel
}

func searchEngineB(query string, ch chan string) {
	time.Sleep(200 * time.Millisecond) // Simulate search time
	result := fmt.Sprintf("Results from Engine B for '%s'", query)
	ch <- result // Send result to channel
}

func main() {
	fmt.Println("=== Search Race ===")
	query := "golang concurrency"

	chA := make(chan string) // Channel for Engine A
	chB := make(chan string) // Channel for Engine B

	go searchEngineA(query, chA) // Start search in Engine A
	go searchEngineB(query, chB) // Start search in Engine B

	select {
	case resultA := <-chA: // Wait for result from Engine A
		fmt.Println("Engine A won! (~300ms)")
		fmt.Println(resultA)
	case resultB := <-chB: // Wait for result from Engine B
		fmt.Println("Engine B won! (~200ms)")
		fmt.Println(resultB)
	}
}
