package utils

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"nvidia_driver_monitor/internal/stats"
)

// HTTP configuration variables
var (
	HTTPTimeout = 10 * time.Second // Default HTTP timeout
	HTTPRetries = 5                // Default number of retries
	httpClient  = &http.Client{
		Timeout: HTTPTimeout,
	}
)

// SetHTTPConfig sets the HTTP timeout and retry configuration
func SetHTTPConfig(timeout time.Duration, retries int) {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	if retries < 1 {
		retries = 1
	}

	HTTPTimeout = timeout
	HTTPRetries = retries
	httpClient = &http.Client{
		Timeout: HTTPTimeout,
	}

	log.Printf("HTTP configuration updated: timeout=%v, retries=%d", HTTPTimeout, HTTPRetries)
}

// HTTPGetWithRetry performs an HTTP GET request with timeout and retry logic
func HTTPGetWithRetry(url string) (*http.Response, error) {
	startTime := time.Now()
	var lastErr error
	var totalRetries int

	collector := stats.GetStatsCollector()

	for attempt := 1; attempt <= HTTPRetries; attempt++ {
		resp, err := httpClient.Get(url)
		if err == nil {
			// Record successful request
			duration := time.Since(startTime)
			collector.RecordRequest(url, duration, totalRetries, true)
			return resp, nil
		}

		lastErr = err
		totalRetries = attempt - 1 // Don't count the first attempt as a retry

		if attempt < HTTPRetries {
			waitTime := time.Duration(attempt) * time.Second
			log.Printf("HTTP request failed (attempt %d/%d): %v. Retrying in %v...", attempt, HTTPRetries, err, waitTime)
			time.Sleep(waitTime)
		} else {
			log.Printf("HTTP request failed after %d attempts: %v", HTTPRetries, err)
		}
	}

	// Record failed request
	duration := time.Since(startTime)
	collector.RecordRequest(url, duration, HTTPRetries-1, false)

	return nil, fmt.Errorf("all %d HTTP attempts failed, last error: %v", HTTPRetries, lastErr)
}

// ExtractSeriesFromLink extracts series name from a Launchpad distro series link
func ExtractSeriesFromLink(link string) string {
	parts := strings.Split(strings.TrimRight(link, "/"), "/")
	if len(parts) < 1 {
		return ""
	}
	return parts[len(parts)-1]
}

// ExtractSeriesAndArchFromLink extracts series and architecture from a Launchpad distro arch series link
func ExtractSeriesAndArchFromLink(link string) (string, string) {
	parts := strings.Split(strings.TrimRight(link, "/"), "/")
	if len(parts) < 2 {
		return "", ""
	}
	return parts[len(parts)-2], parts[len(parts)-1]
}

// FormatSupportedMap formats a map of supported releases as a string
func FormatSupportedMap(supported map[string]bool) string {
	var parts []string
	for k, v := range supported {
		parts = append(parts, fmt.Sprintf("%s:%t", k, v))
	}
	return strings.Join(parts, " ")
}

// IsValidVersion checks if a version string is valid
func IsValidVersion(version string) bool {
	return version != "" && len(version) > 0
}
