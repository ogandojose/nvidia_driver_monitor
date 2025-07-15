package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings" // CHANGED: import strconv for string to int conversion

	version "github.com/knqyf263/go-deb-version"
)

// Structs for decoding JSON response for source packages
type SourceAPIResponse struct {
	Start     int                `json:"start"`
	TotalSize int                `json:"total_size"`
	Entries   []SourcePubHistory `json:"entries"`
}

type SourcePubHistory struct {
	DisplayName          string `json:"display_name"`
	SourcePackageName    string `json:"source_package_name"`
	SourcePackageVersion string `json:"source_package_version"`
	DistroSeriesLink     string `json:"distro_series_link"`
	DatePublished        string `json:"date_published"`
	Pocket               string `json:"pocket"`
	Status               string `json:"status"`
	ComponentName        string `json:"component_name"`
	SectionName          string `json:"section_name"`
}

// Helper to extract series from distro_series_link
func SeriesFromDistroSeriesLink(s string) string {
	parts := strings.Split(strings.TrimRight(s, "/"), "/")
	if len(parts) < 1 {
		return ""
	}
	return parts[len(parts)-1]
}

// Holds the latest version per series/pocket for a source package
type SourceVersionPerPocket struct {
	UpdatesSecurity version.Version // CHANGED: now version.Version
	Proposed        version.Version // CHANGED: now version.Version
}

type SourceVersionPerSeries struct {
	PackageName string
	VersionMap  map[string]*SourceVersionPerPocket
}

// Query Launchpad for published sources for a given package
func getMaxSourceVersionsArchive(sourceName string) (maxVersionPerSeries SourceVersionPerSeries, retErr error) {
	var result SourceAPIResponse
	url := fmt.Sprintf("https://api.launchpad.net/devel/ubuntu/+archive/primary/?ws.op=getPublishedSources&source_name=%s&created_since_date=2025-01-10&order_by_date=true&exact_match=true", sourceName)

	fmt.Println("Query:", url)

	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("HTTP request failed: %v", err)
		retErr = err
		return maxVersionPerSeries, retErr
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Unexpected status code: %d", resp.StatusCode)
		retErr = err
		return maxVersionPerSeries, retErr
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
		retErr = err
		return maxVersionPerSeries, retErr
	}

	log.Printf("📦 Found %d source publications:\n\n", result.TotalSize)

	maxVersionPerSeries = SourceVersionPerSeries{
		PackageName: sourceName,
		VersionMap:  make(map[string]*SourceVersionPerPocket),
	}

	for _, entry := range result.Entries {
		log.Printf("📦 %s\n", entry.DisplayName)
		log.Printf("  → Version:     %s\n", entry.SourcePackageVersion)
		log.Printf("  → Series:      %s\n", entry.DistroSeriesLink)
		log.Printf("  → Published:   %s\n", entry.DatePublished)
		log.Printf("  → Pocket:      %s | Status: %s\n", entry.Pocket, entry.Status)
		log.Printf("  → Component:   %s | Section: %s\n", entry.ComponentName, entry.SectionName)
		log.Println()

		currSeries := SeriesFromDistroSeriesLink(entry.DistroSeriesLink)
		currVersion, err := version.NewVersion(entry.SourcePackageVersion) // CHANGED: parse version
		if err != nil {
			log.Printf("Error parsing version %s: %v", entry.SourcePackageVersion, err)
			continue
		}

		// Ensure the map entry exists
		if _, exists := maxVersionPerSeries.VersionMap[currSeries]; !exists {
			maxVersionPerSeries.VersionMap[currSeries] = &SourceVersionPerPocket{}
			maxVersionPerSeries.VersionMap[currSeries].UpdatesSecurity = currVersion
			maxVersionPerSeries.VersionMap[currSeries].Proposed = currVersion
		}

		switch entry.Pocket {
		case "Proposed":
			if currVersion.GreaterThan(maxVersionPerSeries.VersionMap[currSeries].Proposed) { // CHANGED: use GreaterThan
				maxVersionPerSeries.VersionMap[currSeries].Proposed = currVersion
			}
		case "Updates", "Security":
			if currVersion.GreaterThan(maxVersionPerSeries.VersionMap[currSeries].UpdatesSecurity) { // CHANGED: use GreaterThan
				maxVersionPerSeries.VersionMap[currSeries].UpdatesSecurity = currVersion
			}
		default:
			// ignore
		}
	}

	return maxVersionPerSeries, nil
}

// Print the version map table for source packages
func PrintSourceVersionMapTable(vps SourceVersionPerSeries) {
	fmt.Printf("Source Package: %s\n", vps.PackageName)
	fmt.Printf(
		"| %-30s | %-42s | %-42s |\n",
		"Series",
		"updates_security",
		"proposed",
	)
	fmt.Println("|--------------------------------|--------------------------------------------|--------------------------------------------|")

	for series, pocket := range vps.VersionMap {
		updates := "-"
		proposed := "-"
		if pocket != nil {
			if pocket.UpdatesSecurity.String() != "" {
				updates = pocket.UpdatesSecurity.String()
			}
			if pocket.Proposed.String() != "" {
				proposed = pocket.Proposed.String()
			}
		}
		fmt.Printf(
			"| %-30s | %-42s | %-42s |\n",
			series,
			updates,
			proposed,
		)
	}
}

// ANSI color codes
const (
	ColorGreen = "\033[32m"
	ColorRed   = "\033[31m"
	ColorReset = "\033[0m"
)

/*
PrintSourceVersionMapTableWithSupported prints the version map table and highlights matches with SupportedRelease in green, mismatches in red.
It matches the series to SupportedRelease by extracting the branch name (e.g., "550-server" or "550") from the package name.
*/
func PrintSourceVersionMapTableWithSupported(vps SourceVersionPerSeries, releases []SupportedRelease) {
	fmt.Printf("Source Package: %s\n", vps.PackageName)
	fmt.Printf(
		"| %-30s | %-42s | %-42s | %-20s |\n",
		"Series",
		"updates_security",
		"proposed",
		"Upstream Version",
	)
	fmt.Println("|--------------------------------|--------------------------------------------|--------------------------------------------|----------------------|")
	// Build a lookup: branch name -> SupportedRelease
	supportedMap := make(map[string]SupportedRelease)
	// No need for orderedBranches, since we use orderedSeries for output order
	for _, rel := range releases {
		supportedMap[rel.BranchName] = rel
	}

	// Extract branch name from package name (e.g., "nvidia-driver-550-server" -> "550-server", "nvidia-driver-550" -> "550")
	branchName := ""
	parts := strings.Split(vps.PackageName, "-")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == "server" && i > 0 {
			branchName = parts[i-1] + "-server"
			break
		}
		if _, ok := supportedMap[parts[i]]; ok {
			branchName = parts[i]
			break
		}
	}
	// Fallback: try just last digits
	if branchName == "" {
		for i := len(parts) - 1; i >= 0; i-- {
			if _, ok := supportedMap[parts[i]]; ok {
				branchName = parts[i]
				break
			}
		}
	}

	supported, found := supportedMap[branchName]

	orderedSeries := []string{"questing", "plucky", "noble", "jammy", "focal", "bionic"} // Specify the desired order of series

	for _, series := range orderedSeries {
		pocket, exists := vps.VersionMap[series]
		if !exists {
			log.Println("Series not found:", series)
			continue // or handle missing series as needed
		}
		updates := "-"
		proposed := "-"
		updatesColor := ColorReset
		proposedColor := ColorReset
		upstreamVersion := "-"
		if found && supported.CurrentUpstreamVersion != "" {
			upstreamVersion = supported.CurrentUpstreamVersion
		}
		if pocket != nil && pocket.UpdatesSecurity.String() != "" {
			updates = pocket.UpdatesSecurity.String()
			if found && supported.CurrentUpstreamVersion != "" {
				// Check if all characters in supported.CurrentUpstreamVersion are in updates
				allPresent := true
				for _, c := range supported.CurrentUpstreamVersion {
					if !strings.ContainsRune(updates, c) {
						allPresent = false
						break
					}
				}
				if allPresent {
					updatesColor = ColorGreen
				} else {
					updatesColor = ColorRed
				}
			}
		}
		if pocket != nil && pocket.Proposed.String() != "" {
			proposed = pocket.Proposed.String()
			if found && supported.CurrentUpstreamVersion != "" {
				// Check if all characters in supported.CurrentUpstreamVersion are in proposed, starting from the beginning
				allPresent := true
				for _, c := range supported.CurrentUpstreamVersion {
					if !strings.ContainsRune(proposed, c) {
						allPresent = false
						break
					}
				}
				if allPresent {
					proposedColor = ColorGreen
				} else {
					proposedColor = ColorRed
				}
			}
		}

		fmt.Printf(
			"| %-30s | %s%-42s%s | %s%-42s%s | %-20s |\n",
			series,
			updatesColor, updates, ColorReset,
			proposedColor, proposed, ColorReset,
			upstreamVersion,
		)
	}
}
