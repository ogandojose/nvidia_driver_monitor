package web

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"nvidia_driver_monitor/internal/lrm"
)

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

	// Fetch LRM data - use cached version to avoid refetching if less than 5 minutes old
	lrmData, err := lrm.GetCachedLRMData()
	if err != nil {
		http.Error(w, `{"error": "Failed to fetch LRM data"}`, http.StatusInternalServerError)
		return
	}

	// Debug logging
	log.Printf("API Handler - Debug function returned %d kernels, TotalKernels: %d, SupportedLRM: %d",
		len(lrmData.KernelResults), lrmData.TotalKernels, lrmData.SupportedLRM)

	// Apply filters
	filteredResults := lrmData.KernelResults
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
