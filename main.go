package main

import (
	"fmt"
	"io"
	"log"
	"strings"

	"nvidia_example_550/internal/drivers"
	"nvidia_example_550/internal/packages"
	"nvidia_example_550/internal/releases"
	"nvidia_example_550/internal/sru"
)

func main() {
	// Configuration
	packageQuery := "nvidia-graphics-drivers-570"
	supportedReleasesFile := "supportedReleases.json"

	// Disable logging for cleaner output
	log.SetOutput(io.Discard)

	// Get source package versions
	sourceVersions, err := packages.GetMaxSourceVersionsArchive(packageQuery)
	if err != nil {
		fmt.Printf("Error fetching source versions: %v\n", err)
		return
	}

	packages.PrintSourceVersionMapTable(sourceVersions)

	// Get the latest UDA releases from nvidia.com
	udaEntries, err := drivers.GetNvidiaDriverEntries()
	if err != nil {
		fmt.Printf("Error fetching UDA releases: %v\n", err)
		return
	}

	// Get server driver versions
	_, allBranches, err := drivers.GetLatestServerDriverVersions()
	if err != nil {
		fmt.Printf("Error fetching server driver data: %v\n", err)
		return
	}

	// Read supported releases configuration
	supportedReleases, err := releases.ReadSupportedReleases(supportedReleasesFile)
	if err != nil {
		fmt.Printf("Error reading supported releases: %v\n", err)
		return
	}

	// SRU Cycle Processing
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("SRU CYCLE INFORMATION")
	fmt.Println(strings.Repeat("=", 80))

	sruCycles, err := sru.FetchSRUCycles()
	if err != nil {
		fmt.Printf("Error fetching SRU cycles: %v\n", err)
		return
	}

	sruCycles.AddPredictedCycles()
	sruCycles.PrintSRUCycles()

	// Update supported releases with latest versions
	releases.UpdateSupportedUDAReleases(udaEntries, supportedReleases)
	releases.UpdateSupportedReleasesWithLatestERD(allBranches, supportedReleases)

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

	// Process each supported release
	for _, release := range supportedReleases {
		currentPackageName := "nvidia-graphics-drivers-" + release.BranchName

		currentSourceVersions, err := packages.GetMaxSourceVersionsArchive(currentPackageName)
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

}
