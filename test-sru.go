package main

import (
	"fmt"
	"log"

	"nvidia_example_550/internal/sru"
)

func main() {
	// Test SRU cycle functionality
	sruCycles, err := sru.FetchSRUCycles()
	if err != nil {
		log.Fatalf("Error fetching SRU cycles: %v", err)
	}

	fmt.Println("=== SRU Cycles Test ===")
	fmt.Printf("Total cycles: %d\n", len(sruCycles.Cycles))

	// Test current cycle
	current := sruCycles.GetCurrentCycle()
	if current != nil {
		fmt.Printf("Current cycle: %s (release: %s)\n", current.Name, current.ReleaseDate)
	} else {
		fmt.Println("No current cycle found")
	}

	// Test stream filtering
	stream1 := sruCycles.GetCyclesByStream(1)
	stream2 := sruCycles.GetCyclesByStream(2)
	fmt.Printf("Stream 1 cycles: %d\n", len(stream1))
	fmt.Printf("Stream 2 cycles: %d\n", len(stream2))

	// Test active cycles
	active := sruCycles.GetActiveCycles()
	fmt.Printf("Active cycles: %d\n", len(active))

	if len(active) > 0 {
		fmt.Println("Active cycles:")
		for _, cycle := range active {
			fmt.Printf("  - %s (release: %s, stream: %d)\n", cycle.Name, cycle.ReleaseDate, cycle.Stream)
		}
	}
}
