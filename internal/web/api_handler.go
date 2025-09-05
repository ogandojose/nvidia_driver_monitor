package web

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"nvidia_driver_monitor/internal/lrm"
	"nvidia_driver_monitor/internal/stats"
)

// tryGetLRMData is a test seam that can be overridden in unit tests.
// By default it points to lrm.TryGetCachedLRMData.
var tryGetLRMData = lrm.TryGetCachedLRMData

// APIHandler handles REST API endpoints
type APIHandler struct{}

// NewAPIHandler creates a new API handler
func NewAPIHandler() *APIHandler {
	return &APIHandler{}
}

// LRMDataHandler returns LRM data as JSON
func (h *APIHandler) LRMDataHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get query parameters for filtering
	series := r.URL.Query().Get("series")
	status := r.URL.Query().Get("status")
	routing := r.URL.Query().Get("routing")
	limit := r.URL.Query().Get("limit")
	offset := r.URL.Query().Get("offset")

	// Fetch LRM data - prefer non-blocking cached response; fall back to placeholder
	lrmData, err := tryGetLRMData()
	if err != nil {
		now := time.Now()
		lrmData = &lrm.LRMVerifierData{
			KernelResults: []lrm.KernelLRMResult{},
			TotalKernels:  0,
			SupportedLRM:  0,
			LastUpdated:   now,
			IsInitialized: false,
		}
	}

	// Debug logging
	log.Printf("API Handler - Debug function returned %d kernels, TotalKernels: %d, SupportedLRM: %d",
		len(lrmData.KernelResults), lrmData.TotalKernels, lrmData.SupportedLRM)

	// Apply filters (ensure non-nil slice so JSON encodes as [] not null)
	filteredResults := lrmData.KernelResults
	if filteredResults == nil {
		filteredResults = make([]lrm.KernelLRMResult, 0)
	}
	if series != "" {
		filteredResults = filterBySeries(filteredResults, series)
	}
	if status != "" {
		filteredResults = filterByStatus(filteredResults, status)
	}
	if routing != "" {
		filteredResults = filterByRouting(filteredResults, routing)
	}

	// Apply pagination
	if limit != "" || offset != "" {
		filteredResults = applyPagination(filteredResults, limit, offset)
	}

	// Create response
	response := APIResponse{
		Data: APILRMData{
			KernelResults: filteredResults,
			TotalKernels:  lrmData.TotalKernels,
			SupportedLRM:  lrmData.SupportedLRM,
			LastUpdated:   lrmData.LastUpdated,
			IsInitialized: lrmData.IsInitialized,
			Progress:      lrm.GetProgress(),
		},
		Meta: APIMeta{
			Total:    len(lrmData.KernelResults),
			Filtered: len(filteredResults),
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, `{"error": "Failed to encode response"}`, http.StatusInternalServerError)
		return
	}
}

// LRMProgressHandler returns just the LRM progress as JSON
func (h *APIHandler) LRMProgressHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	prog := lrm.GetProgress()
	if err := json.NewEncoder(w).Encode(prog); err != nil {
		http.Error(w, `{"error": "Failed to encode response"}`, http.StatusInternalServerError)
		return
	}
}

// HealthHandler returns health status
func (h *APIHandler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	health := map[string]interface{}{
		"status":  "healthy",
		"service": "nvidia-driver-monitor",
	}

	if err := json.NewEncoder(w).Encode(health); err != nil {
		http.Error(w, `{"error": "Failed to encode response"}`, http.StatusInternalServerError)
		return
	}
}

// RoutingsHandler returns available routing values
func (h *APIHandler) RoutingsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get available routings
	routings, err := lrm.GetAvailableRoutings()
	if err != nil {
		http.Error(w, `{"error": "Failed to fetch routing data"}`, http.StatusInternalServerError)
		return
	}

	// Create response
	response := map[string]interface{}{
		"routings": routings,
		"count":    len(routings),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, `{"error": "Failed to encode response"}`, http.StatusInternalServerError)
		return
	}
}

// StatisticsHandler returns API statistics as JSON
func (h *APIHandler) StatisticsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	collector := stats.GetStatsCollector()

	// Prepare response data
	response := map[string]interface{}{
		"current_window":          collector.GetCurrentWindowInfo(),
		"historical_windows":      collector.GetAllWindowsStats(),
		"server_time":             time.Now().Format("2006-01-02 15:04:05 UTC"),
		"window_duration_minutes": 10,
		"max_stored_windows":      collector.GetMaxWindows(),
	}

	// Encode and send response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding statistics response: %v", err)
		http.Error(w, `{"error": "Failed to encode response"}`, http.StatusInternalServerError)
		return
	}
}

// API response types
type APIResponse struct {
	Data APILRMData `json:"data"`
	Meta APIMeta    `json:"meta"`
}

type APILRMData struct {
	KernelResults []lrm.KernelLRMResult `json:"kernel_results"`
	TotalKernels  int                   `json:"total_kernels"`
	SupportedLRM  int                   `json:"supported_lrm"`
	LastUpdated   interface{}           `json:"last_updated"`
	IsInitialized bool                  `json:"is_initialized"`
	Progress      lrm.LRMProgress       `json:"progress"`
}

type APIMeta struct {
	Total    int `json:"total"`
	Filtered int `json:"filtered"`
}

// Filter functions
func filterBySeries(results []lrm.KernelLRMResult, series string) []lrm.KernelLRMResult {
	var filtered []lrm.KernelLRMResult
	for _, result := range results {
		if result.Series == series {
			filtered = append(filtered, result)
		}
	}
	return filtered
}

func filterByStatus(results []lrm.KernelLRMResult, status string) []lrm.KernelLRMResult {
	var filtered []lrm.KernelLRMResult
	for _, result := range results {
		switch strings.ToUpper(status) {
		case "SUPPORTED":
			if result.Supported {
				filtered = append(filtered, result)
			}
		case "LTS":
			if result.LTS {
				filtered = append(filtered, result)
			}
		case "DEV", "DEVELOPMENT":
			if result.Development {
				filtered = append(filtered, result)
			}
		case "ESM":
			if result.ESM {
				filtered = append(filtered, result)
			}
		}
	}
	return filtered
}

func filterByRouting(results []lrm.KernelLRMResult, routing string) []lrm.KernelLRMResult {
	var filtered []lrm.KernelLRMResult
	for _, result := range results {
		if result.Routing == routing {
			filtered = append(filtered, result)
		}
	}
	return filtered
}

func applyPagination(results []lrm.KernelLRMResult, limitStr, offsetStr string) []lrm.KernelLRMResult {
	limit := 50 // default
	offset := 0 // default

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	if offset >= len(results) {
		return []lrm.KernelLRMResult{}
	}

	end := offset + limit
	if end > len(results) {
		end = len(results)
	}

	return results[offset:end]
}

// CacheStatusHandler returns cache status information
func (h *APIHandler) CacheStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get cache status from LRM module
	status := lrm.GetCacheStatus()

	// Add server timestamp
	status["server_time"] = time.Now().Format("2006-01-02 15:04:05 UTC")

	// Encode and send response
	if err := json.NewEncoder(w).Encode(status); err != nil {
		log.Printf("Error encoding cache status response: %v", err)
		http.Error(w, `{"error": "Failed to encode response"}`, http.StatusInternalServerError)
		return
	}
}
