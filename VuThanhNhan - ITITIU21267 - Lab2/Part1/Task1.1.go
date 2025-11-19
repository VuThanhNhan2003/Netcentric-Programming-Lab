package main
import (
 "fmt"
 "time"
)

func counter(name string, max int) {
	for i := 1; i <= max; i++ {
		fmt.Printf("Counter %s: %d\n", name, i)
		time.Sleep(200 * time.Millisecond)
	}
	fmt.Printf("Counter %s finished!\n", name)
}
func main() {
 fmt.Println("=== Three Counters ===")

	// Start 3 counters concurrently
	go counter("A", 3)
	go counter("B", 4)
	go counter("C", 5)

 time.Sleep(2 * time.Second)
 fmt.Println("All done!")
}
