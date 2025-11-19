package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type Student struct {
	ID         int
	StudyHours int
}

func student(id int, studyHours int, library chan bool, wg *sync.WaitGroup, 
	mu *sync.Mutex, waitTimes *[]time.Duration, entryTimes map[int]time.Time) { 
		// mu *sync.Mutex: Mutex for synchronizing access to shared data
		// waitTimes *[]time.Duration: Slice to store wait times
		// entryTimes map[int]time.Time: Map to store entry times of students
	defer wg.Done()

	startWait := time.Now()

	// Try to enter the library (waits if full)
	library <- true

	waitDuration := time.Since(startWait) // Calculate wait time

	// Log wait time (1 student per time)
	mu.Lock() // Lock the mutex before accessing shared data
	*waitTimes = append(*waitTimes, waitDuration) // Store wait time
	entryTimes[id] = time.Now() // Record entry time
	mu.Unlock() // Unlock the mutex

	fmt.Printf("Student %d entered library, will study for %d hours\n", id, studyHours)
	time.Sleep(time.Duration(studyHours) * time.Second)
	fmt.Printf("Student %d left library after %d hours\n", id, studyHours)

	<-library // Leave the library
}

func main() {
	fmt.Println("=== Library Simulation ===")
	fmt.Println("Library capacity: 30 students")
	fmt.Println("Total students today: 100")
	fmt.Println("Simulation: 1 second = 1 hour\n")

	const totalStudents = 100
	const libraryCapacity = 30

	library := make(chan bool, libraryCapacity)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var waitTimes []time.Duration
	entryTimes := make(map[int]time.Time) // Map to track entry times

	students := make([]Student, totalStudents) // Slice to hold students
	for i := 0; i < totalStudents; i++ {
		students[i] = Student{
			ID:         i + 1,
			StudyHours: rand.Intn(4) + 1, // 1-4 hours
		}
	}

	start := time.Now()

	// Start all students as goroutines
	for _, s := range students {
		wg.Add(1)
		go student(s.ID, s.StudyHours, library, &wg, &mu, &waitTimes, entryTimes)
	}

	wg.Wait() // Wait for all students to finish
	duration := time.Since(start)

	// Calculate average wait time
	var totalWait time.Duration
	for _, w := range waitTimes {
		totalWait += w
	}
	avgWait := totalWait / time.Duration(len(waitTimes))

	fmt.Println("\n=== Simulation Complete ===")
	fmt.Printf("Total students served: %d\n", totalStudents)
	fmt.Printf("Library was open for: %.0f hours\n", duration.Seconds())
	fmt.Printf("Average wait time: %.2f hours\n", avgWait.Seconds())
	fmt.Printf("Peak occupancy: %d students\n", libraryCapacity)
}
