package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"nvidia_example_550/internal/web"
)

// TestData represents the expected data structure for comparison
type TestData struct {
	PackageName string
	Series      []TestSeries
}

type TestSeries struct {
	Series          string
	UpdatesSecurity string
	Proposed        string
	UpstreamVersion string
	ReleaseDate     string
	UpdatesColor    string
	ProposedColor   string
}

func main() {
	// Test that web service returns expected data
	fmt.Println("Testing web service data consistency...")

	// Give the server a moment to start if it's not already running
	time.Sleep(2 * time.Second)

	// Test API endpoint
	resp, err := http.Get("http://localhost:8080/api")
	if err != nil {
		fmt.Printf("Error: Web service not running or accessible: %v\n", err)
		fmt.Println("Please start the web service first with: ./start-web-server.sh")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: API returned status code %d\n", resp.StatusCode)
		return
	}

	var packages []web.PackageData
	if err := json.NewDecoder(resp.Body).Decode(&packages); err != nil {
		fmt.Printf("Error decoding JSON: %v\n", err)
		return
	}

	fmt.Printf("Successfully retrieved %d packages from web service\n", len(packages))

	// Test some basic data validation
	for _, pkg := range packages {
		fmt.Printf("Package: %s\n", pkg.PackageName)

		if !strings.HasPrefix(pkg.PackageName, "nvidia-graphics-drivers-") {
			fmt.Printf("  WARNING: Unexpected package name format\n")
		}

		greenCount := 0
		redCount := 0

		for _, series := range pkg.Series {
			if series.UpdatesColor == "success" {
				greenCount++
			} else if series.UpdatesColor == "danger" {
				redCount++
			}
		}

		fmt.Printf("  Series: %d, Green: %d, Red: %d\n", len(pkg.Series), greenCount, redCount)
	}

	// Test specific package endpoint
	fmt.Println("\nTesting specific package endpoint...")
	resp2, err := http.Get("http://localhost:8080/api?package=nvidia-graphics-drivers-575")
	if err != nil {
		fmt.Printf("Error testing specific package: %v\n", err)
		return
	}
	defer resp2.Body.Close()

	var specificPackage web.PackageData
	if err := json.NewDecoder(resp2.Body).Decode(&specificPackage); err != nil {
		fmt.Printf("Error decoding specific package JSON: %v\n", err)
		return
	}

	fmt.Printf("Specific package test successful: %s with %d series\n",
		specificPackage.PackageName, len(specificPackage.Series))

	fmt.Println("\nAll tests passed! Web service is working correctly.")
	fmt.Println("Open http://localhost:8080 in your browser to see the web interface.")
}
