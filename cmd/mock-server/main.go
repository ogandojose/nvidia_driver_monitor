package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"nvidia_driver_monitor/internal/config"
)

// MockServer provides mock responses for external APIs
type MockServer struct {
	dataDir string
	port    int
}

// NewMockServer creates a new mock server instance
func NewMockServer(dataDir string, port int) *MockServer {
	return &MockServer{
		dataDir: dataDir,
		port:    port,
	}
}

// Start starts the mock server
func (ms *MockServer) Start() error {
	http.HandleFunc("/", ms.handleRequest)

	addr := fmt.Sprintf(":%d", ms.port)
	log.Printf("🚀 Mock Server starting on http://localhost%s", addr)
	log.Printf("📂 Serving mock data from: %s", ms.dataDir)
	log.Printf("📋 Available endpoints:")
	log.Printf("   • Launchpad API: http://localhost%s/launchpad/*", addr)
	log.Printf("   • NVIDIA APIs: http://localhost%s/nvidia/*", addr)
	log.Printf("   • Kernel APIs: http://localhost%s/kernel/*", addr)
	log.Printf("   • Ubuntu APIs: http://localhost%s/ubuntu/*", addr)

	return http.ListenAndServe(addr, nil)
}

// handleRequest routes requests to appropriate mock handlers
func (ms *MockServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	log.Printf("📥 Mock request: %s %s", r.Method, r.URL.Path)

	// Add CORS headers for browser requests
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	path := r.URL.Path

	switch {
	case strings.HasPrefix(path, "/launchpad/"):
		ms.handleLaunchpadAPI(w, r)
	case strings.HasPrefix(path, "/nvidia/"):
		ms.handleNVIDIAAPI(w, r)
	case strings.HasPrefix(path, "/kernel/"):
		ms.handleKernelAPI(w, r)
	case strings.HasPrefix(path, "/ubuntu/"):
		ms.handleUbuntuAPI(w, r)
	default:
		ms.handleNotFound(w, r)
	}
}

// handleLaunchpadAPI handles Launchpad API mock responses with parameter awareness
func (ms *MockServer) handleLaunchpadAPI(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	query := r.URL.Query()

	// Handle published sources API
	if strings.Contains(path, "+archive/primary") && query.Get("ws.op") == "getPublishedSources" {
		sourceName := query.Get("source_name")
		if sourceName == "" {
			http.Error(w, "Missing source_name parameter", http.StatusBadRequest)
			return
		}

		// Check for series-specific requests
		var seriesPrefix string
		if strings.Contains(path, "/ubuntu/") && !strings.Contains(path, "/ubuntu/+archive/") {
			// Extract series from path like /launchpad/ubuntu/noble/+archive/primary
			parts := strings.Split(path, "/")
			for i, part := range parts {
				if part == "ubuntu" && i+1 < len(parts) && parts[i+1] != "+archive" {
					seriesPrefix = fmt.Sprintf("%s-", parts[i+1])
					break
				}
			}
		}

		// Try to serve series-specific file first, then fall back to generic
		var filename string
		if seriesPrefix != "" {
			filename = fmt.Sprintf("launchpad/sources/%s%s.json", seriesPrefix, sourceName)
			if _, err := os.Stat(filepath.Join(ms.dataDir, filename)); os.IsNotExist(err) {
				filename = fmt.Sprintf("launchpad/sources/%s.json", sourceName)
			}
		} else {
			filename = fmt.Sprintf("launchpad/sources/%s.json", sourceName)
		}

		// Log parameter analysis for debugging
		params := []string{}
		if query.Get("created_since_date") != "" {
			params = append(params, fmt.Sprintf("date=%s", query.Get("created_since_date")))
		}
		if query.Get("exact_match") == "true" {
			params = append(params, "exact_match=true")
		}
		if query.Get("order_by_date") == "true" {
			params = append(params, "order_by_date=true")
		}

		paramStr := ""
		if len(params) > 0 {
			paramStr = fmt.Sprintf(" [%s]", strings.Join(params, ", "))
		}

		log.Printf("📦 Source query: %s%s%s", sourceName,
			func() string {
				if seriesPrefix != "" {
					return fmt.Sprintf(" [series=%s]", strings.TrimSuffix(seriesPrefix, "-"))
				}
				return ""
			}(),
			paramStr)
		ms.serveFile(w, filename, "application/json")
		return
	}

	// Handle published binaries API
	if strings.Contains(path, "+archive/primary") && query.Get("ws.op") == "getPublishedBinaries" {
		binaryName := query.Get("binary_name")
		if binaryName == "" {
			http.Error(w, "Missing binary_name parameter", http.StatusBadRequest)
			return
		}

		// Check for series-specific requests
		var seriesPrefix string
		if strings.Contains(path, "/ubuntu/") && !strings.Contains(path, "/ubuntu/+archive/") {
			parts := strings.Split(path, "/")
			for i, part := range parts {
				if part == "ubuntu" && i+1 < len(parts) && parts[i+1] != "+archive" {
					seriesPrefix = fmt.Sprintf("%s-", parts[i+1])
					break
				}
			}
		}

		// Try series-specific file first, then fall back to generic
		var filename string
		if seriesPrefix != "" {
			filename = fmt.Sprintf("launchpad/binaries/%s%s.json", seriesPrefix, binaryName)
			if _, err := os.Stat(filepath.Join(ms.dataDir, filename)); os.IsNotExist(err) {
				filename = fmt.Sprintf("launchpad/binaries/%s.json", binaryName)
			}
		} else {
			filename = fmt.Sprintf("launchpad/binaries/%s.json", binaryName)
		}

		exactMatch := ""
		if query.Get("exact_match") == "true" {
			exactMatch = " [exact_match=true]"
		}

		log.Printf("📦 Binary query: %s%s%s", binaryName,
			func() string {
				if seriesPrefix != "" {
					return fmt.Sprintf(" [series=%s]", strings.TrimSuffix(seriesPrefix, "-"))
				}
				return ""
			}(),
			exactMatch)
		ms.serveFile(w, filename, "application/json")
		return
	}

	// Handle Ubuntu series API
	if strings.HasPrefix(path, "/launchpad/ubuntu/") {
		series := strings.TrimPrefix(path, "/launchpad/ubuntu/")
		// Remove any trailing path components
		if idx := strings.Index(series, "/"); idx != -1 {
			series = series[:idx]
		}

		if series != "" {
			log.Printf("🐧 Series info: %s", series)
			ms.serveFile(w, fmt.Sprintf("launchpad/series/%s.json", series), "application/json")
			return
		}
	}

	ms.handleNotFound(w, r)
}

// handleNVIDIAAPI handles NVIDIA API mock responses
func (ms *MockServer) handleNVIDIAAPI(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch path {
	case "/nvidia/datacenter/releases.json":
		ms.serveFile(w, "nvidia/server-drivers.json", "application/json")
	case "/nvidia/drivers":
		ms.serveFile(w, "nvidia/driver-archive.html", "text/html")
	default:
		ms.handleNotFound(w, r)
	}
}

// handleKernelAPI handles kernel API mock responses
func (ms *MockServer) handleKernelAPI(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch path {
	case "/kernel/series.yaml":
		ms.serveFile(w, "kernel/series.yaml", "text/yaml")
	case "/kernel/sru-cycle.yaml":
		ms.serveFile(w, "kernel/sru-cycle.yaml", "text/yaml")
	default:
		ms.handleNotFound(w, r)
	}
}

// handleUbuntuAPI handles Ubuntu API mock responses
func (ms *MockServer) handleUbuntuAPI(w http.ResponseWriter, r *http.Request) {
	// For now, just return a simple response
	// This could be expanded to serve Ubuntu assets
	ms.handleNotFound(w, r)
}

// handleNotFound handles 404 responses
func (ms *MockServer) handleNotFound(w http.ResponseWriter, r *http.Request) {
	log.Printf("❌ Mock endpoint not found: %s", r.URL.Path)
	response := map[string]interface{}{
		"error":   "Mock endpoint not found",
		"path":    r.URL.Path,
		"message": "This mock endpoint is not implemented yet",
		"hint":    "Check the mock server configuration or add test data files",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(response)
}

// serveFile serves a file from the test data directory
func (ms *MockServer) serveFile(w http.ResponseWriter, filename, contentType string) {
	fullPath := filepath.Join(ms.dataDir, filename)

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		log.Printf("⚠️  Mock data file not found: %s", fullPath)
		// Generate a minimal response based on the file type
		ms.generateFallbackResponse(w, filename, contentType)
		return
	}

	// Serve the file
	data, err := os.ReadFile(fullPath)
	if err != nil {
		log.Printf("❌ Error reading mock data file %s: %v", fullPath, err)
		http.Error(w, "Error reading mock data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Write(data)
	log.Printf("✅ Served mock data: %s", filename)
}

// generateFallbackResponse generates a minimal response when data files don't exist
func (ms *MockServer) generateFallbackResponse(w http.ResponseWriter, filename, contentType string) {
	w.Header().Set("Content-Type", "application/json")

	// Generate minimal responses based on the API type
	var response interface{}

	switch {
	case strings.Contains(filename, "launchpad/sources/"):
		response = map[string]interface{}{
			"total_size": 0,
			"start":      0,
			"entries":    []interface{}{},
		}
	case strings.Contains(filename, "launchpad/binaries/"):
		response = map[string]interface{}{
			"total_size": 0,
			"start":      0,
			"entries":    []interface{}{},
		}
	case strings.Contains(filename, "nvidia/server-drivers"):
		response = map[string]interface{}{
			"drivers": map[string]interface{}{},
		}
	default:
		response = map[string]interface{}{
			"mock":    true,
			"message": "Fallback response - no test data file found",
			"file":    filename,
		}
	}

	json.NewEncoder(w).Encode(response)
	log.Printf("🔄 Generated fallback response for: %s", filename)
}

func main() {
	var (
		port    = flag.Int("port", 9999, "Port to run the mock server on")
		dataDir = flag.String("data-dir", "test-data", "Directory containing mock data files")
		cfgFile = flag.String("config", "", "Load port and data directory from config file")
	)
	flag.Parse()

	// Load configuration if specified
	if *cfgFile != "" {
		cfg, err := config.LoadConfig(*cfgFile)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}

		if cfg.Testing.MockServerPort > 0 {
			*port = cfg.Testing.MockServerPort
		}
		if cfg.Testing.DataDir != "" {
			*dataDir = cfg.Testing.DataDir
		}
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Create and start mock server
	server := NewMockServer(*dataDir, *port)
	log.Fatal(server.Start())
}
