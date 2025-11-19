package main

import (
	"fmt"
	"time"
)

func fetchWebsite(name string, delayMs int) {
 // TODO: Implement fetch simulation
 fmt.Printf("Fetching %s...\n", name)
 time.Sleep(time.Duration(delayMs) * time.Millisecond)
 fmt.Printf("✓ Got data from %s\n", name)
}

func main() {
 fmt.Println("=== Fetching Websites ===")
 start := time.Now()

 // TODO: Fetch 4 websites concurrently
 go fetchWebsite("Google.com", 200)
 go fetchWebsite("Facebook.com", 400)
 go fetchWebsite("Amazon.com", 300)
 go fetchWebsite("Twitter.com", 150)

  time.Sleep(500 * time.Millisecond)
 fmt.Printf("\nCompleted in: %s\n", time.Since(start))
}