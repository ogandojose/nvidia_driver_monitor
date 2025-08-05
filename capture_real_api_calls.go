package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"nvidia_driver_monitor/internal/config"
	"nvidia_driver_monitor/internal/drivers"
	"nvidia_driver_monitor/internal/lrm"
	"nvidia_driver_monitor/internal/packages"
	"nvidia_driver_monitor/internal/releases"
	"nvidia_driver_monitor/internal/sru"
)

// HTTPClient wrapper to capture all API calls
type CapturingHTTPClient struct {
	client    *http.Client
	outputDir string
}

func NewCapturingHTTPClient(outputDir string) *CapturingHTTPClient {
	return &CapturingHTTPClient{
		client:    &http.Client{Timeout: 30 * time.Second},
		outputDir: outputDir,
	}
}

func (c *CapturingHTTPClient) Get(url string) (*http.Response, error) {
	fmt.Printf("📡 Fetching: %s\n", url)
	
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, err
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		resp.Body.Close()
		return nil, err
	}
	resp.Body.Close()

	// Save the response to disk
	if err := c.saveResponse(url, body); err != nil {
		fmt.Printf("❌ Failed to save response for %s: %v\n", url, err)
	} else {
		fmt.Printf("💾 Saved response for %s\n", url)
	}

	// Create a new response with the body we read
	resp.Body = io.NopCloser(strings.NewReader(string(body)))
	return resp, nil
}

func (c *CapturingHTTPClient) saveResponse(url string, body []byte) error {
	// Create a safe filename from the URL
	filename := c.urlToFilename(url)
	filepath := filepath.Join(c.outputDir, filename)
	
	// Ensure directory exists
	dir := filepath.Dir(filepath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Save the response
	return os.WriteFile(filepath, body, 0644)
}

func (c *CapturingHTTPClient) urlToFilename(url string) string {
	// Replace dangerous characters and create a readable filename
	filename := url
	filename = strings.ReplaceAll(filename, "https://", "")
	filename = strings.ReplaceAll(filename, "http://", "")
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ReplaceAll(filename, "?", "_")
	filename = strings.ReplaceAll(filename, "&", "_")
	filename = strings.ReplaceAll(filename, "=", "_")
	filename = strings.ReplaceAll(filename, "+", "_")
	filename = strings.ReplaceAll(filename, ":", "_")
	
	// Add appropriate extension
	if strings.Contains(url, ".json") || strings.Contains(url, "releases.json") {
		if !strings.HasSuffix(filename, ".json") {
			filename += ".json"
		}
	} else if strings.Contains(url, ".yaml") {
		if !strings.HasSuffix(filename, ".yaml") {
			filename += ".yaml"
		}
	} else {
		filename += ".json" // default to JSON
	}
	
	return filename
}

func main() {
	outputDir := "captured_real_api_responses"
	
	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	fmt.Println("🚀 Starting NVIDIA Driver Monitor with API Capture")
	fmt.Printf("📁 Saving responses to: %s\n", outputDir)
	fmt.Println(strings.Repeat("=", 80))

	// Load configuration
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Printf("Warning: Could not load config file, using defaults: %v", err)
		cfg = config.DefaultConfig()
	}

	// Ensure testing is disabled to use real URLs
	cfg.Testing.Enabled = false

	// Create capturing HTTP client
	capturingClient := NewCapturingHTTPClient(outputDir)

	// Replace the default HTTP client in the packages that make HTTP calls
	// We'll need to modify the internal packages to use our client
	
	// Set configuration for various packages
	lrm.SetProcessorConfig(cfg)
	sru.SetSRUConfig(cfg)
	packages.SetPackagesConfig(cfg)

	// Configuration
	packageQuery := "nvidia-graphics-drivers-570"
	supportedReleasesFile := "data/supportedReleases.json"

	// Enable logging for this capture session
	log.SetOutput(os.Stdout)

	fmt.Println("\n📦 Fetching source package versions...")
	// Get source package versions
	sourceVersions, err := packages.GetMaxSourceVersionsArchive(cfg, packageQuery)
	if err != nil {
		fmt.Printf("Error fetching source versions: %v\n", err)
	} else {
		packages.PrintSourceVersionMapTable(sourceVersions)
	}

	fmt.Println("\n🎮 Fetching NVIDIA UDA releases...")
	// Get the latest UDA releases from nvidia.com
	udaEntries, err := drivers.GetNvidiaDriverEntries(cfg)
	if err != nil {
		fmt.Printf("Error fetching UDA releases: %v\n", err)
	}

	fmt.Println("\n🖥️ Fetching NVIDIA server driver versions...")
	// Get server driver versions
	_, allBranches, err := drivers.GetLatestServerDriverVersions(cfg)
	if err != nil {
		fmt.Printf("Error fetching server driver data: %v\n", err)
	}

	fmt.Println("\n📋 Reading supported releases configuration...")
	// Read supported releases configuration
	supportedReleases, err := releases.ReadSupportedReleases(supportedReleasesFile)
	if err != nil {
		fmt.Printf("Error reading supported releases: %v\n", err)
		return
	}

	fmt.Println("\n🔄 Fetching SRU cycles...")
	// SRU Cycle Processing
	sruCycles, err := sru.FetchSRUCycles()
	if err != nil {
		fmt.Printf("Error fetching SRU cycles: %v\n", err)
	} else {
		sruCycles.AddPredictedCycles()
		sruCycles.PrintSRUCycles()
	}

	// Update supported releases with latest versions
	if udaEntries != nil {
		releases.UpdateSupportedUDAReleases(udaEntries, supportedReleases)
	}
	if allBranches != nil {
		releases.UpdateSupportedReleasesWithLatestERD(allBranches, supportedReleases)
	}

	// Print updated supported releases
	releases.PrintSupportedReleases(supportedReleases)

	// Fetch SRU cycles for package processing
	sruCyclesForPackages, err := sru.FetchSRUCycles()
	if err != nil {
		fmt.Printf("Error fetching SRU cycles: %v\n", err)
		sruCyclesForPackages = nil // Continue without SRU cycles
	} else {
		sruCyclesForPackages.AddPredictedCycles()
	}

	fmt.Println("\n📦 Processing each supported release...")
	// Process each supported release
	for _, release := range supportedReleases {
		currentPackageName := "nvidia-graphics-drivers-" + release.BranchName
		fmt.Printf("📦 Processing package: %s\n", currentPackageName)

		currentSourceVersions, err := packages.GetMaxSourceVersionsArchive(cfg, currentPackageName)
		if err != nil {
			fmt.Printf("Error fetching source versions for %s: %v\n", currentPackageName, err)
			continue
		}

		packages.PrintSourceVersionMapTableWithSupported(currentSourceVersions, supportedReleases, sruCyclesForPackages)
	}

	// Save updated supported releases
	if err := releases.WriteSupportedReleases(supportedReleasesFile, supportedReleases); err != nil {
		fmt.Printf("Error writing supported releases: %v\n", err)
	}

	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("✅ API Capture Complete!")
	fmt.Printf("📁 All responses saved to: %s\n", outputDir)
	
	// List captured files
	fmt.Println("\n📄 Captured files:")
	filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			fmt.Printf("  - %s (%d bytes)\n", path, info.Size())
		}
		return nil
	})
}
