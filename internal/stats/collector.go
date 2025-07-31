package stats

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// APIStats represents statistics for API calls
type APIStats struct {
	Domain          string        `json:"domain"`          // e.g., "launchpad.net", "nvidia.com", "kernel.ubuntu.com"
	TotalRequests   int64         `json:"total_requests"`  // Total number of requests
	SuccessfulReqs  int64         `json:"successful_reqs"` // Number of successful requests
	FailedReqs      int64         `json:"failed_reqs"`     // Number of failed requests
	TotalRetries    int64         `json:"total_retries"`   // Total number of retries across all requests
	AverageRespTime float64       `json:"avg_response_ms"` // Average response time in milliseconds
	TotalRespTime   time.Duration `json:"-"`               // Internal: sum of all response times
}

// TimeWindow represents a 10-minute window of statistics
type TimeWindow struct {
	StartTime time.Time            `json:"start_time"`
	EndTime   time.Time            `json:"end_time"`
	Stats     map[string]*APIStats `json:"stats"` // Domain -> APIStats
}

// StatsCollector manages API statistics collection
type StatsCollector struct {
	mu           sync.RWMutex
	windows      []*TimeWindow // Last 100 windows (1000 minutes of data)
	currentWin   *TimeWindow
	maxWindows   int
	persistFile  string // Path to persistence file
	saveInterval time.Duration
}

var (
	globalCollector *StatsCollector
	once            sync.Once
)

// GetStatsCollector returns the global statistics collector instance
func GetStatsCollector() *StatsCollector {
	once.Do(func() {
		persistFile := "statistics_data.json"
		globalCollector = &StatsCollector{
			maxWindows:   100,
			persistFile:  persistFile,
			saveInterval: 5 * time.Minute, // Save every 5 minutes
			windows:      make([]*TimeWindow, 0, 100),
		}

		// Load existing data if available
		if err := globalCollector.loadFromFile(); err != nil {
			log.Printf("Warning: Could not load existing statistics data: %v", err)
		}

		// Start new window if none exists
		if globalCollector.currentWin == nil {
			globalCollector.startNewWindow()
		}

		globalCollector.startWindowRotation()
		globalCollector.startPeriodicSaving()
	})
	return globalCollector
}

// startNewWindow creates a new 10-minute time window
func (sc *StatsCollector) startNewWindow() {
	now := time.Now()
	sc.currentWin = &TimeWindow{
		StartTime: now,
		EndTime:   now.Add(10 * time.Minute),
		Stats:     make(map[string]*APIStats),
	}
}

// startWindowRotation starts a goroutine that rotates windows every 10 minutes
func (sc *StatsCollector) startWindowRotation() {
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			sc.rotateWindow()
		}
	}()
}

// rotateWindow moves current window to history and starts a new one
func (sc *StatsCollector) rotateWindow() {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Add current window to history
	sc.windows = append(sc.windows, sc.currentWin)

	// Keep only the last maxWindows (100)
	if len(sc.windows) > sc.maxWindows {
		sc.windows = sc.windows[1:]
	}

	// Start new window
	sc.startNewWindow()

	// Save to file after rotation
	go func() {
		if err := sc.saveToFile(); err != nil {
			log.Printf("Error saving statistics after window rotation: %v", err)
		}
	}()
}

// extractDomain extracts domain from URL for categorization
func extractDomain(url string) string {
	// Simple domain extraction
	if len(url) < 8 {
		return "unknown"
	}

	// Remove protocol
	start := 0
	if url[:7] == "http://" {
		start = 7
	} else if url[:8] == "https://" {
		start = 8
	}

	// Find end of domain
	end := len(url)
	for i := start; i < len(url); i++ {
		if url[i] == '/' || url[i] == '?' {
			end = i
			break
		}
	}

	domain := url[start:end]

	// Categorize known domains
	if domain == "api.launchpad.net" {
		return "launchpad"
	} else if domain == "docs.nvidia.com" || domain == "www.nvidia.com" {
		return "nvidia"
	} else if domain == "kernel.ubuntu.com" {
		return "ubuntu-kernel"
	}

	return domain
}

// RecordRequest records an API request with its outcome
func (sc *StatsCollector) RecordRequest(url string, duration time.Duration, retries int, success bool) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	domain := extractDomain(url)

	// Get or create stats for this domain in current window
	if sc.currentWin.Stats[domain] == nil {
		sc.currentWin.Stats[domain] = &APIStats{
			Domain: domain,
		}
	}

	stats := sc.currentWin.Stats[domain]
	stats.TotalRequests++
	stats.TotalRetries += int64(retries)
	stats.TotalRespTime += duration

	// Calculate average response time
	if stats.TotalRequests > 0 {
		stats.AverageRespTime = float64(stats.TotalRespTime.Nanoseconds()) / float64(stats.TotalRequests) / 1e6 // Convert to milliseconds
	}

	if success {
		stats.SuccessfulReqs++
	} else {
		stats.FailedReqs++
	}
}

// GetCurrentWindowStats returns statistics for the current 10-minute window
func (sc *StatsCollector) GetCurrentWindowStats() map[string]*APIStats {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	// Create a copy to avoid race conditions
	result := make(map[string]*APIStats)
	for domain, stats := range sc.currentWin.Stats {
		result[domain] = &APIStats{
			Domain:          stats.Domain,
			TotalRequests:   stats.TotalRequests,
			SuccessfulReqs:  stats.SuccessfulReqs,
			FailedReqs:      stats.FailedReqs,
			TotalRetries:    stats.TotalRetries,
			AverageRespTime: stats.AverageRespTime,
		}
	}

	return result
}

// GetAllWindowsStats returns statistics for all stored windows
func (sc *StatsCollector) GetAllWindowsStats() []*TimeWindow {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	// Create a copy to avoid race conditions
	result := make([]*TimeWindow, len(sc.windows))
	for i, window := range sc.windows {
		result[i] = &TimeWindow{
			StartTime: window.StartTime,
			EndTime:   window.EndTime,
			Stats:     make(map[string]*APIStats),
		}

		// Copy stats
		for domain, stats := range window.Stats {
			result[i].Stats[domain] = &APIStats{
				Domain:          stats.Domain,
				TotalRequests:   stats.TotalRequests,
				SuccessfulReqs:  stats.SuccessfulReqs,
				FailedReqs:      stats.FailedReqs,
				TotalRetries:    stats.TotalRetries,
				AverageRespTime: stats.AverageRespTime,
			}
		}
	}

	return result
}

// GetCurrentWindowInfo returns information about the current window
func (sc *StatsCollector) GetCurrentWindowInfo() *TimeWindow {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	return &TimeWindow{
		StartTime: sc.currentWin.StartTime,
		EndTime:   sc.currentWin.EndTime,
		Stats:     sc.GetCurrentWindowStats(),
	}
}

// PersistentData represents the data structure for JSON persistence
type PersistentData struct {
	Windows    []*TimeWindow `json:"windows"`
	CurrentWin *TimeWindow   `json:"current_window"`
	SavedAt    time.Time     `json:"saved_at"`
}

// saveToFile saves current statistics to a JSON file
func (sc *StatsCollector) saveToFile() error {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	data := &PersistentData{
		Windows:    sc.windows,
		CurrentWin: sc.currentWin,
		SavedAt:    time.Now(),
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(sc.persistFile)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal statistics: %w", err)
	}

	// Write to temporary file first
	tempFile := sc.persistFile + ".tmp"
	if err := os.WriteFile(tempFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempFile, sc.persistFile); err != nil {
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}

// loadFromFile loads statistics from a JSON file
func (sc *StatsCollector) loadFromFile() error {
	// Check if file exists
	if _, err := os.Stat(sc.persistFile); os.IsNotExist(err) {
		return nil // No existing data, start fresh
	}

	// Read file
	jsonData, err := os.ReadFile(sc.persistFile)
	if err != nil {
		return fmt.Errorf("failed to read statistics file: %w", err)
	}

	// Parse JSON
	var data PersistentData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return fmt.Errorf("failed to parse statistics JSON: %w", err)
	}

	// Validate data age (don't load data older than 24 hours)
	if time.Since(data.SavedAt) > 24*time.Hour {
		log.Printf("Statistics data is older than 24 hours, starting fresh")
		return nil
	}

	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Load windows
	sc.windows = data.Windows
	if sc.windows == nil {
		sc.windows = make([]*TimeWindow, 0, sc.maxWindows)
	}

	// Load current window if it's still valid (not expired)
	if data.CurrentWin != nil && time.Now().Before(data.CurrentWin.EndTime) {
		sc.currentWin = data.CurrentWin
	}

	log.Printf("Loaded %d historical windows from %s", len(sc.windows), sc.persistFile)
	return nil
}

// startPeriodicSaving starts a goroutine that periodically saves statistics
func (sc *StatsCollector) startPeriodicSaving() {
	go func() {
		ticker := time.NewTicker(sc.saveInterval)
		defer ticker.Stop()

		for range ticker.C {
			if err := sc.saveToFile(); err != nil {
				log.Printf("Error during periodic save: %v", err)
			}
		}
	}()
}

// GetMaxWindows returns the maximum number of windows stored
func (sc *StatsCollector) GetMaxWindows() int {
	return sc.maxWindows
}
