package main

import (
	"fmt"
	"sync"
	"time"
)

func findEvens(numbers []int, result chan []int, wg *sync.WaitGroup) { //wg *sync.WaitGroup: WaitGroup to synchronize goroutines
	defer wg.Done() // Notify when done

	var evens []int
	for _, num := range numbers {
		if num%2 == 0 {
			evens = append(evens, num)
		}
		time.Sleep(50 * time.Millisecond)
	}
	result <- evens // Send result to channel
}

func findOdds(numbers []int, result chan []int, wg *sync.WaitGroup) {
	defer wg.Done() // Notify when done

	var odds []int
	for _, num := range numbers {
		if num%2 != 0 {
			odds = append(odds, num)
		}
		time.Sleep(50 * time.Millisecond)
	}
	result <- odds // Send result to channel
}

func findSquares(numbers []int, result chan []int, wg *sync.WaitGroup) {
	defer wg.Done() // Notify when done

	var squares []int
	for _, num := range numbers { 
		squares = append(squares, num*num)
		time.Sleep(50 * time.Millisecond)
	}
	result <- squares // Send result to channel
}

func main() {
	fmt.Println("=== Number Processor ===")
	numbers := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10} 
	fmt.Printf("Numbers: %v\n", numbers)

	evenChan := make(chan []int) // Channel for even numbers
	oddChan := make(chan []int)   // Channel for odd numbers
	squareChan := make(chan []int) // Channel for square numbers

	var wg sync.WaitGroup // WaitGroup to wait for all goroutines
	wg.Add(3) // We have 3 goroutines to wait for

	start := time.Now()

	go findEvens(numbers, evenChan, &wg) // Start goroutine to find evens
	go findOdds(numbers, oddChan, &wg)
	go findSquares(numbers, squareChan, &wg)

	go func() {
		wg.Wait() // Wait for all goroutines to finish
		close(evenChan) // Close channels
		close(oddChan)
		close(squareChan)
	}()

	evens := <-evenChan // Receive results from channels
	odds := <-oddChan
	squares := <-squareChan

	fmt.Printf("Evens: %v\n", evens)
	fmt.Printf("Odds: %v\n", odds)
	fmt.Printf("Squares: %v\n", squares)

	fmt.Printf("\nTime: %s\n", time.Since(start))
}
