package stats

import (
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
	mu         sync.RWMutex
	windows    []*TimeWindow // Last 10 windows (100 minutes of data)
	currentWin *TimeWindow
	maxWindows int
}

var (
	globalCollector *StatsCollector
	once            sync.Once
)

// GetStatsCollector returns the global statistics collector instance
func GetStatsCollector() *StatsCollector {
	once.Do(func() {
		globalCollector = &StatsCollector{
			maxWindows: 10,
			windows:    make([]*TimeWindow, 0, 10),
		}
		globalCollector.startNewWindow()
		globalCollector.startWindowRotation()
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

	// Keep only the last maxWindows
	if len(sc.windows) > sc.maxWindows {
		sc.windows = sc.windows[1:]
	}

	// Start new window
	sc.startNewWindow()
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
