package main

import (
	"fmt"
	"io"
	"log"
	"strings"

	"nvidia_driver_monitor/internal/config"
	"nvidia_driver_monitor/internal/drivers"
	"nvidia_driver_monitor/internal/lrm"
	"nvidia_driver_monitor/internal/packages"
	"nvidia_driver_monitor/internal/releases"
	"nvidia_driver_monitor/internal/sru"
	"nvidia_driver_monitor/internal/utils"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("config/config.json")
	if err != nil {
		log.Printf("Warning: Could not load config file, using defaults: %v", err)
		cfg = config.DefaultConfig()
	}

	// Set configuration for various packages
	lrm.SetProcessorConfig(cfg)
	sru.SetSRUConfig(cfg)
	packages.SetPackagesConfig(cfg)
	utils.SetHTTPAuthToken(cfg.HTTP.GetForgejoToken())

	// Configuration
	packageQuery := "nvidia-graphics-drivers-570"
	supportedReleasesFile := "data/supportedReleases.json"

	// Read supported releases configuration upfront so we can limit branch traversal
	supportedReleases, err := releases.ReadSupportedReleases(supportedReleasesFile)
	if err != nil {
		fmt.Printf("Error reading supported releases: %v\n", err)
		return
	}

	// Disable logging for cleaner output
	log.SetOutput(io.Discard)

	// Get source package versions
	sourceVersions, err := packages.GetMaxSourceVersionsArchive(cfg, packageQuery)
	if err != nil {
		fmt.Printf("Error fetching source versions: %v\n", err)
		return
	}

	packages.PrintSourceVersionMapTable(sourceVersions)

	branchMajors := releases.GetUniqueBranchMajors(supportedReleases)

	// Get the latest UDA releases from nvidia.com
	udaEntries, err := drivers.GetNvidiaDriverEntries(cfg, branchMajors)
	if err != nil {
		fmt.Printf("Error fetching UDA releases: %v\n", err)
		return
	}

	// Get server driver versions
	_, allBranches, err := drivers.GetLatestServerDriverVersions(cfg)
	if err != nil {
		fmt.Printf("Error fetching server driver data: %v\n", err)
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

}
