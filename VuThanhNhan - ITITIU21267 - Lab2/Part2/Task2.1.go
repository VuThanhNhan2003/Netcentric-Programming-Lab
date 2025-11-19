package main

import (
	"fmt"
	"time"
)
func calculateSum(numbers []int, result chan int) {
	// TODO: Calculate sum and send to channel
	sum := 0
	for _, num := range numbers {
		sum += num
		time.Sleep(100 * time.Millisecond) 
	}
	result <- sum // Send sum to channel
}

func calculateAverage(numbers []int, result chan float64) {
	// TODO: Calculate average and send to channel
	sum := 0
	for _, num := range numbers {
		sum += num
		time.Sleep(100 * time.Millisecond) 
	}
	average := float64(sum) / float64(len(numbers))
	result <- average // Send average to channel
}

func main() {
	fmt.Println("=== Concurrent Calculator ===")
	numbers := []int{10, 20, 30, 40, 50}

	// TODO: Create channels, run calculations, receive results
	fmt.Printf("Numbers: %v\n", numbers)

	sumChan := make(chan int) // Channel for sum
	avgChan := make(chan float64)

	start := time.Now()

	go calculateSum(numbers, sumChan) // Start sum calculation
	go calculateAverage(numbers, avgChan)

	sum := <-sumChan // Receive sum from channel
	average := <-avgChan

	fmt.Printf("Sum: %d\n", sum)
	fmt.Printf("Average: %.1f\n", average)

	fmt.Printf("Time: %s\n", time.Since(start))
}
