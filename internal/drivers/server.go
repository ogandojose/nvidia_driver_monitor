package drivers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"time"
)

// AllBranches represents all driver branches
type AllBranches map[string]BranchEntry

// BranchEntry represents a driver branch entry
type BranchEntry struct {
	Type       string       `json:"type"`
	DriverInfo []DriverInfo `json:"driver_info"`
}

// DriverInfo represents driver information
type DriverInfo struct {
	ReleaseVersion string            `json:"release_version"`
	ReleaseDate    string            `json:"release_date"`
	ReleaseNotes   string            `json:"release_notes"`
	Architectures  []string          `json:"architectures"`
	RunfileURL     map[string]string `json:"runfile_url"`
}

// GetLatestServerDriverVersions retrieves the latest server driver versions
func GetLatestServerDriverVersions() (map[string]DriverInfo, AllBranches, error) {
	resp, err := http.Get("https://docs.nvidia.com/datacenter/tesla/drivers/releases.json")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch server driver data: %w", err)
	}
	defer resp.Body.Close()

	var data AllBranches
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	// Sort branch keys in reverse order
	branchKeys := make([]string, 0, len(data))
	for k := range data {
		branchKeys = append(branchKeys, k)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(branchKeys)))

	latestVersions := make(map[string]DriverInfo)

	for _, versionListed := range branchKeys {
		branch := data[versionListed]
		if len(branch.DriverInfo) == 0 {
			continue
		}

		// Find the latest driver info for this branch
		var latest DriverInfo
		var latestDate time.Time

		for _, info := range branch.DriverInfo {
			releaseDate, err := time.Parse("2006-01-02", info.ReleaseDate)
			if err != nil {
				log.Printf("Invalid date format for %s: %v", info.ReleaseDate, err)
				continue
			}

			if latest.ReleaseVersion == "" || releaseDate.After(latestDate) {
				latest = info
				latestDate = releaseDate
			}
		}

		if latest.ReleaseVersion != "" {
			latestVersions[versionListed] = latest
		}
	}

	return latestVersions, data, nil
}

// PrintDriverVersions prints driver versions in a formatted table
func PrintDriverVersions(latestVersions map[string]DriverInfo, allBranches AllBranches) {
	fmt.Println("Latest Server Driver Versions:")
	fmt.Println("Branch\tVersion\tDate\tType")
	fmt.Println("----------------------------------------------------")

	// Sort keys for consistent output
	keys := make([]string, 0, len(latestVersions))
	for k := range latestVersions {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, branch := range keys {
		info := latestVersions[branch]
		branchType := "Unknown"
		if branchEntry, exists := allBranches[branch]; exists {
			branchType = branchEntry.Type
		}

		fmt.Printf("%s\t%s\t%s\t%s\n",
			branch,
			info.ReleaseVersion,
			info.ReleaseDate,
			branchType)
	}

	fmt.Println("----------------------------------------------------")
}
