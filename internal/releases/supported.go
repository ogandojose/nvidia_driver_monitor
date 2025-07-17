package releases

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"nvidia_example_550/internal/drivers"
)

// SupportedRelease represents a supported release configuration
type SupportedRelease struct {
	BranchName             string            `json:"branch_name"`
	IsServer               bool              `json:"is_server"`
	IsSupported            map[string]bool   `json:"is_supported"`
	CurrentUpstreamVersion string            `json:"current_upstream_version"`
	DatePublished          string            `json:"date_published"`
	SourceVersionUpdates   map[string]string `json:"source_version_updates,omitempty"`
	SourceVersionProposed  map[string]string `json:"source_version_proposed,omitempty"`
}

// ReadSupportedReleases reads the JSON file and returns an array of SupportedRelease
func ReadSupportedReleases(filename string) ([]SupportedRelease, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	var releases []SupportedRelease
	if err := json.Unmarshal(bytes, &releases); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return releases, nil
}

// WriteSupportedReleases writes the supported releases to a JSON file
func WriteSupportedReleases(filename string, releases []SupportedRelease) error {
	data, err := json.MarshalIndent(releases, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filename, err)
	}

	return nil
}

// PrintSupportedReleases prints the array of SupportedRelease as a table to stdout
func PrintSupportedReleases(releases []SupportedRelease) {
	fmt.Printf("%-20s %-8s %-80s %-25s %-15s\n", "Branch Name", "Server", "Supported", "Current Upstream Version", "Date Published")
	fmt.Println("-------------------------------------------------------------------------------------------------------------------------------------------------------------")

	for _, r := range releases {
		// Format IsSupported map as key:value pairs
		supportedStr := ""
		for k, v := range r.IsSupported {
			supportedStr += fmt.Sprintf("%s:%t ", k, v)
		}

		fmt.Printf("%-20s %-8t %-80s %-25s %-15s\n",
			r.BranchName,
			r.IsServer,
			supportedStr,
			r.CurrentUpstreamVersion,
			r.DatePublished)
	}

	fmt.Println("-------------------------------------------------------------------------------------------------------------------------------------------------------------")
}

// UpdateSupportedUDAReleases updates supported releases with UDA release information
func UpdateSupportedUDAReleases(udaEntries []drivers.DriverEntry, supportedReleases []SupportedRelease) {
	// Build a map: major version -> latest non-beta DriverEntry
	latestByMajor := make(map[string]drivers.DriverEntry)
	for _, entry := range udaEntries {
		if entry.IsBeta {
			continue
		}
		major := strings.SplitN(entry.Version, ".", 2)[0]
		if prev, ok := latestByMajor[major]; !ok || entry.Date.After(prev.Date) {
			latestByMajor[major] = entry
		}
	}

	// Update releases
	for i, rel := range supportedReleases {
		major := rel.BranchName
		if entry, ok := latestByMajor[major]; ok {
			supportedReleases[i].CurrentUpstreamVersion = entry.Version
			supportedReleases[i].DatePublished = entry.Date.Format("2006-01-02")
		}
	}
}

// UpdateSupportedReleasesWithLatestERD updates supported releases with latest Enterprise Ready Driver versions
func UpdateSupportedReleasesWithLatestERD(allBranches drivers.AllBranches, supportedReleases []SupportedRelease) {
	for i := range supportedReleases {
		rel := &supportedReleases[i]
		if len(rel.BranchName) > 7 && rel.BranchName[len(rel.BranchName)-7:] == "-server" {
			// Extract the 3 digit number preceding "-server"
			branchNum := rel.BranchName[:len(rel.BranchName)-7]
			if branch, ok := allBranches[branchNum]; ok && len(branch.DriverInfo) > 0 {
				// Find the latest DriverInfo by ReleaseDate
				latest := branch.DriverInfo[0]
				for _, info := range branch.DriverInfo[1:] {
					d1, err1 := time.Parse("2006-01-02", latest.ReleaseDate)
					d2, err2 := time.Parse("2006-01-02", info.ReleaseDate)
					if err1 == nil && err2 == nil && d2.After(d1) {
						latest = info
					}
				}
				rel.CurrentUpstreamVersion = latest.ReleaseVersion
				rel.DatePublished = latest.ReleaseDate
			}
		}
	}
}
