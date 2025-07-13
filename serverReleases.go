package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"time"
)

type AllBranches map[string]BranchEntry

type BranchEntry struct {
	Type       string       `json:"type"`
	DriverInfo []DriverInfo `json:"driver_info"`
}

type DriverInfo struct {
	ReleaseVersion string            `json:"release_version"`
	ReleaseDate    string            `json:"release_date"`
	ReleaseNotes   string            `json:"release_notes"`
	Architectures  []string          `json:"architectures"`
	RunfileURL     map[string]string `json:"runfile_url"`
}

func getLatestServerDriverVersions() (map[string]DriverInfo, AllBranches, error) {
	resp, err := http.Get("https://docs.nvidia.com/datacenter/tesla/drivers/releases.json")
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	var data AllBranches
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, nil, err
	}

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

		// Sort releases by most recent date first
		sort.Slice(branch.DriverInfo, func(i, j int) bool {
			d1, _ := time.Parse("2006-01-02", branch.DriverInfo[i].ReleaseDate)
			d2, _ := time.Parse("2006-01-02", branch.DriverInfo[j].ReleaseDate)
			return d1.After(d2)
		})

		latestVersions[versionListed] = branch.DriverInfo[0]
	}

	return latestVersions, data, nil
}

func logDriverVersions(data map[string]DriverInfo, branches AllBranches) {
	for version, info := range data {
		branch := branches[version]
		log.Printf("== Branch %s (%s) ==\n", version, branch.Type)
		log.Printf("- Version: %s\n", info.ReleaseVersion)
		log.Printf("- Date:    %s\n", info.ReleaseDate)
		log.Printf("- Notes:   %s\n", info.ReleaseNotes)
		for arch, url := range info.RunfileURL {
			log.Printf("  [%s] %s\n", arch, url)
		}
		log.Println()
	}
}

func printDriverVersionsToStdout(data map[string]DriverInfo, branches AllBranches) {
	for version, info := range data {
		branch := branches[version]
		fmt.Printf("== Branch %s (%s) ==\n", version, branch.Type)
		fmt.Printf("- Version: %s\n", info.ReleaseVersion)
		fmt.Printf("- Date:    %s\n", info.ReleaseDate)
		fmt.Printf("- Notes:   %s\n", info.ReleaseNotes)
		for arch, url := range info.RunfileURL {
			fmt.Printf("  [%s] %s\n", arch, url)
		}
		fmt.Println()
	}
}
