package sru

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"gopkg.in/yaml.v2"
)

// SRUCycle represents a single SRU cycle entry
type SRUCycle struct {
	Name           string    `yaml:"-"` // The cycle name (extracted from map key)
	StartDate      string    `yaml:"start-date,omitempty"`
	ReleaseDate    string    `yaml:"release-date"`
	NotesLink      string    `yaml:"notes-link,omitempty"`
	Stream         int       `yaml:"stream,omitempty"`
	Owner          string    `yaml:"owner,omitempty"`
	Complete       bool      `yaml:"complete,omitempty"`
	Hold           bool      `yaml:"hold,omitempty"`
	Current        bool      `yaml:"current,omitempty"`
	CutoffDate     string    `yaml:"cutoff-date,omitempty"`
	ParsedDate     time.Time `yaml:"-"` // Parsed release date for sorting
	PredictedCycle bool      `yaml:"predicted-cycle,omitempty"`
}

// SRUCycles holds a collection of SRU cycles
type SRUCycles struct {
	Cycles []SRUCycle
}

// FetchSRUCycles fetches and parses SRU cycles from the Ubuntu kernel repository
func FetchSRUCycles() (*SRUCycles, error) {
	url := "https://kernel.ubuntu.com/forgejo/kernel/kernel-versions/raw/branch/main/info/sru-cycle.yaml"

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch SRU cycles: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse YAML into a map
	var cycleMap map[string]SRUCycle
	if err := yaml.Unmarshal(body, &cycleMap); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Convert map to slice and add cycle names
	var cycles []SRUCycle
	for name, cycle := range cycleMap {
		cycle.Name = name

		// Parse release date for sorting
		if cycle.ReleaseDate != "" {
			if parsedDate, err := time.Parse("2006-01-02", cycle.ReleaseDate); err == nil {
				cycle.ParsedDate = parsedDate
			}
		}

		// Set default stream if not specified
		if cycle.Stream == 0 {
			cycle.Stream = 1
		}

		// If CutoffDate is empty, calculate as Name minus 5 days (Name format: YYYY.MM.DD)
		if cycle.CutoffDate == "" {
			// Try to parse the date from the Name
			if len(name) >= 10 {
				if t, err := time.Parse("2006.01.02", name[:10]); err == nil {
					cutoff := t.AddDate(0, 0, -5)
					cycle.CutoffDate = cutoff.Format("2006-01-02")
				}
			}
		}

		cycles = append(cycles, cycle)
	}

	// Sort by release date (newest first)
	sort.Slice(cycles, func(i, j int) bool {
		return cycles[i].ParsedDate.After(cycles[j].ParsedDate)
	})

	return &SRUCycles{Cycles: cycles}, nil
}

// PrintSRUCycles prints all SRU cycles in a formatted table
func (sru *SRUCycles) PrintSRUCycles() {
	fmt.Printf("SRU Cycles (sorted by release date, newest first):\n")
	fmt.Printf("| %-15s | %-12s | %-12s | %-12s | %-8s | %-15s | %-8s | %-8s |\n",
		"Name", "Release Date", "Start Date", "Cutoff Date", "Stream", "Owner", "Complete", "Current")
	fmt.Println("|-----------------|--------------|--------------|--------------|----------|-----------------|----------|----------|")

	for _, cycle := range sru.Cycles {
		startDate := "-"
		if cycle.StartDate != "" {
			startDate = cycle.StartDate
		}

		cutoffDate := "-"
		if cycle.CutoffDate != "" {
			cutoffDate = cycle.CutoffDate
		}

		owner := "-"
		if cycle.Owner != "" {
			owner = cycle.Owner
		}

		complete := "false"
		if cycle.Complete {
			complete = "true"
		}

		current := "false"
		if cycle.Current {
			current = "true"
		}

		fmt.Printf("| %-15s | %-12s | %-12s | %-12s | %-8d | %-15s | %-8s | %-8s |\n",
			cycle.Name,
			cycle.ReleaseDate,
			startDate,
			cutoffDate,
			cycle.Stream,
			owner,
			complete,
			current)
	}

	fmt.Printf("\nTotal SRU cycles: %d\n", len(sru.Cycles))
}

// GetCurrentCycle returns the current SRU cycle
func (sru *SRUCycles) GetCurrentCycle() *SRUCycle {
	for _, cycle := range sru.Cycles {
		if cycle.Current {
			return &cycle
		}
	}
	return nil
}

// GetCyclesByStream returns cycles filtered by stream number
func (sru *SRUCycles) GetCyclesByStream(stream int) []SRUCycle {
	var filtered []SRUCycle
	for _, cycle := range sru.Cycles {
		if cycle.Stream == stream {
			filtered = append(filtered, cycle)
		}
	}
	return filtered
}

// GetActiveCycles returns cycles that are not complete
func (sru *SRUCycles) GetActiveCycles() []SRUCycle {
	var active []SRUCycle
	for _, cycle := range sru.Cycles {
		if !cycle.Complete {
			active = append(active, cycle)
		}
	}
	return active
}
func (sru *SRUCycles) AddPredictedCycles() {
	const numPredicted = 3
	if len(sru.Cycles) == 0 {
		return
	}

	// Find the newest cycle by release date
	newest := sru.Cycles[0]
	for _, c := range sru.Cycles {
		if c.ParsedDate.After(newest.ParsedDate) {
			newest = c
		}
	}

	// Parse the name date (format: YYYY.MM.DD)
	baseNameDate, err := time.Parse("2006.01.02", newest.Name[:10])
	if err != nil {
		return
	}

	// Parse the release date (format: YYYY-MM-DD)
	baseReleaseDate, err := time.Parse("2006-01-02", newest.ReleaseDate)
	if err != nil {
		return
	}

	// Parse the cutoff date (format: YYYY-MM-DD)
	baseCutoffDate, err := time.Parse("2006-01-02", newest.CutoffDate)
	if err != nil {
		// fallback: 5 days before release date
		baseCutoffDate = baseReleaseDate.AddDate(0, 0, -5)
	}

	nextNameDate := baseNameDate
	nextReleaseDate := baseReleaseDate
	nextCutoffDate := baseCutoffDate

	for i := 1; i <= numPredicted; i++ {
		nextNameDate = nextNameDate.AddDate(0, 0, 28)
		nextReleaseDate = nextReleaseDate.AddDate(0, 0, 28)
		nextCutoffDate = nextCutoffDate.AddDate(0, 0, 28)

		name := nextNameDate.Format("2006.01.02")
		releaseDate := nextReleaseDate.Format("2006-01-02")
		cutoffDate := nextCutoffDate.Format("2006-01-02")

		predicted := SRUCycle{
			Name:           name,
			ReleaseDate:    releaseDate,
			CutoffDate:     cutoffDate,
			Stream:         0, // Use 0 to indicate predicted cycles
			Owner:          "Predicted",
			PredictedCycle: true,
			ParsedDate:     nextReleaseDate,
		}
		// Insert at the beginning
		sru.Cycles = append([]SRUCycle{predicted}, sru.Cycles...)
	}
}

// GetMinimumCutoffAfterDate finds the minimum cutoff date that is after the given date
func (sru *SRUCycles) GetMinimumCutoffAfterDate(driverReleaseDate string) *SRUCycle {
	driverDate, err := time.Parse("2006-01-02", driverReleaseDate)
	if err != nil {
		return nil
	}

	var minCycle *SRUCycle
	var minCutoffDate time.Time

	for i, cycle := range sru.Cycles {
		if cycle.CutoffDate == "" {
			continue
		}

		cutoffDate, err := time.Parse("2006-01-02", cycle.CutoffDate)
		if err != nil {
			continue
		}

		// Check if cutoff date is after driver release date
		if cutoffDate.After(driverDate) {
			if minCycle == nil || cutoffDate.Before(minCutoffDate) {
				minCycle = &sru.Cycles[i] // Use index instead of loop variable address
				minCutoffDate = cutoffDate
			}
		}
	}

	return minCycle
}
