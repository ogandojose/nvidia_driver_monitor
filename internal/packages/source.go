package packages

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"nvidia_example_550/internal/releases"
	"nvidia_example_550/internal/sru"

	version "github.com/knqyf263/go-deb-version"
)

// SourceAPIResponse represents the JSON response for source packages
type SourceAPIResponse struct {
	Start     int                `json:"start"`
	TotalSize int                `json:"total_size"`
	Entries   []SourcePubHistory `json:"entries"`
}

// SourcePubHistory represents a source package publication history entry
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

// SourceVersionPerPocket holds the latest version per pocket for a source package
type SourceVersionPerPocket struct {
	UpdatesSecurity version.Version
	Proposed        version.Version
}

// SourceVersionPerSeries holds package versions per series
type SourceVersionPerSeries struct {
	PackageName string
	VersionMap  map[string]*SourceVersionPerPocket
}

// SeriesFromDistroSeriesLink extracts series from distro_series_link
func SeriesFromDistroSeriesLink(s string) string {
	parts := strings.Split(strings.TrimRight(s, "/"), "/")
	if len(parts) < 1 {
		return ""
	}
	return parts[len(parts)-1]
}

// GetMaxSourceVersionsArchive retrieves the maximum source package versions from archive
func GetMaxSourceVersionsArchive(packageName string) (*SourceVersionPerSeries, error) {
	if packageName == "" {
		return nil, fmt.Errorf("package name cannot be empty")
	}

	url := fmt.Sprintf("https://api.launchpad.net/devel/ubuntu/+archive/primary/?ws.op=getPublishedSources&source_name=%s&created_since_date=2025-01-10&order_by_date=true&exact_match=true", packageName)

	fmt.Println("Query:", url)

	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("HTTP request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Unexpected status code: %d", resp.StatusCode)
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var apiResp SourceAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	log.Printf("📦 Found %d source publications:\n\n", apiResp.TotalSize)

	versionMap := make(map[string]*SourceVersionPerPocket)

	for _, entry := range apiResp.Entries {
		if entry.Status != "Published" {
			continue
		}

		log.Printf("📦 %s\n", entry.DisplayName)
		log.Printf("  → Version:     %s\n", entry.SourcePackageVersion)
		log.Printf("  → Series:      %s\n", entry.DistroSeriesLink)
		log.Printf("  → Published:   %s\n", entry.DatePublished)
		log.Printf("  → Pocket:      %s | Status: %s\n", entry.Pocket, entry.Status)
		log.Printf("  → Component:   %s | Section: %s\n", entry.ComponentName, entry.SectionName)
		log.Println()

		series := SeriesFromDistroSeriesLink(entry.DistroSeriesLink)
		if series == "" {
			continue
		}

		ver, err := version.NewVersion(entry.SourcePackageVersion)
		if err != nil {
			log.Printf("Error parsing version %s: %v", entry.SourcePackageVersion, err)
			continue
		}

		// Ensure the map entry exists
		if _, exists := versionMap[series]; !exists {
			versionMap[series] = &SourceVersionPerPocket{}
			versionMap[series].UpdatesSecurity = ver
			versionMap[series].Proposed = ver
		}

		switch entry.Pocket {
		case "Proposed":
			if ver.GreaterThan(versionMap[series].Proposed) {
				versionMap[series].Proposed = ver
			}
		case "Updates", "Security":
			if ver.GreaterThan(versionMap[series].UpdatesSecurity) {
				versionMap[series].UpdatesSecurity = ver
			}
		default:
			// ignore
		}
	}

	return &SourceVersionPerSeries{
		PackageName: packageName,
		VersionMap:  versionMap,
	}, nil
}

// getMaxSourceVersionsArchive is a wrapper function for backward compatibility
func getMaxSourceVersionsArchive(packageName string) (*SourceVersionPerSeries, error) {
	return GetMaxSourceVersionsArchive(packageName)
}

// PrintSourceVersionMapTable prints the source version map in table format
func PrintSourceVersionMapTable(vps *SourceVersionPerSeries) {
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

// PrintSourceVersionMapTableWithSupported prints source version map with supported releases and SRU cycles
func PrintSourceVersionMapTableWithSupported(vps *SourceVersionPerSeries, supportedReleases []releases.SupportedRelease, sruCycles *sru.SRUCycles) {
	fmt.Printf("Source Package: %s\n", vps.PackageName)
	fmt.Printf(
		"| %-30s | %-42s | %-42s | %-20s | %-15s | %-15s |\n",
		"Series",
		"updates_security",
		"proposed",
		"Upstream Version",
		"Release Date",
		"SRU Cycle",
	)
	fmt.Println("|--------------------------------|--------------------------------------------|--------------------------------------------|----------------------|-----------------|-----------------|")

	// Build a lookup: branch name -> SupportedRelease
	supportedMap := make(map[string]releases.SupportedRelease)
	for _, rel := range supportedReleases {
		supportedMap[rel.BranchName] = rel
	}

	// Extract branch name from package name (e.g., "nvidia-graphics-drivers-550-server" -> "550-server", "nvidia-graphics-drivers-550" -> "550")
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
			continue // Skip series that don't exist in the version map
		}
		updates := "-"
		proposed := "-"
		updatesColor := ColorReset
		proposedColor := ColorReset
		upstreamVersion := "-"
		releaseDate := "-"
		sruCycleDate := "-"

		if found && supported.CurrentUpstreamVersion != "" {
			upstreamVersion = supported.CurrentUpstreamVersion
			if supported.DatePublished != "" {
				releaseDate = supported.DatePublished
			}
		}

		if pocket != nil && pocket.UpdatesSecurity.String() != "" {
			updates = pocket.UpdatesSecurity.String()
			if found && supported.CurrentUpstreamVersion != "" {
				// Check if the upstream version is contained in the package version
				if strings.Contains(updates, supported.CurrentUpstreamVersion) {
					updatesColor = ColorGreen
				} else {
					updatesColor = ColorRed
					// If version is red (upstream is greater), find SRU cycle
					if sruCycles != nil && supported.DatePublished != "" {
						if sruCycle := sruCycles.GetMinimumCutoffAfterDate(supported.DatePublished); sruCycle != nil {
							sruCycleDate = sruCycle.ReleaseDate
						}
					}
				}
			}
		}

		if pocket != nil && pocket.Proposed.String() != "" {
			proposed = pocket.Proposed.String()
			if found && supported.CurrentUpstreamVersion != "" {
				// Check if the upstream version is contained in the package version
				if strings.Contains(proposed, supported.CurrentUpstreamVersion) {
					proposedColor = ColorGreen
				} else {
					proposedColor = ColorRed
					// If version is red (upstream is greater), find SRU cycle (only if not already set)
					if sruCycles != nil && supported.DatePublished != "" && sruCycleDate == "-" {
						if sruCycle := sruCycles.GetMinimumCutoffAfterDate(supported.DatePublished); sruCycle != nil {
							sruCycleDate = sruCycle.ReleaseDate
						}
					}
				}
			}
		}

		fmt.Printf(
			"| %-30s | %s%-42s%s | %s%-42s%s | %-20s | %-15s | %-15s |\n",
			series,
			updatesColor, updates, ColorReset,
			proposedColor, proposed, ColorReset,
			upstreamVersion,
			releaseDate,
			sruCycleDate,
		)
	}
}

// ANSI color codes for console output
const (
	ColorGreen = "\033[32m"
	ColorRed   = "\033[31m"
	ColorReset = "\033[0m"
)
