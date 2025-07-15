package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type SupportedRelease struct {
	BranchName             string            `json:"branch_name"`
	IsServer               bool              `json:"is_server"`
	IsSupported            map[string]bool   `json:"is_supported"`
	CurrentUpstreamVersion string            `json:"current_upstream_version"`
	DatePublished          string            `json:"date_published"`
	SourceVersionUpdates   map[string]string `json:"source_version_updates,omitempty"`  // Optional field for source version updates
	SourceVersionProposed  map[string]string `json:"source_version_proposed,omitempty"` // Optional field for source version proposed
}

// ReadSupportedReleases reads the JSON file and returns an array of SupportedRelease
func ReadSupportedReleases(filename string) ([]SupportedRelease, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var releases []SupportedRelease
	if err := json.Unmarshal(bytes, &releases); err != nil {
		return nil, err
	}
	return releases, nil
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
			r.BranchName, r.IsServer, supportedStr, r.CurrentUpstreamVersion, r.DatePublished)
	}
}

// LogSupportedReleases logs the array of SupportedRelease as a table using the standard logger
func LogSupportedReleases(releases []SupportedRelease) {
	fmt.Printf("%-20s %-8s %-80s %-25s %-15s\n", "Branch Name", "Server", "Supported", "Current Upstream Version", "Date Published")
	fmt.Println("-------------------------------------------------------------------------------------------------------------------------------------------------------------")
	for _, r := range releases {
		// Format IsSupported map as key:value pairs
		supportedStr := ""
		for k, v := range r.IsSupported {
			supportedStr += fmt.Sprintf("%s:%t ", k, v)
		}
		fmt.Printf("%-20s %-8t %-80s %-25s %-15s\n",
			r.BranchName, r.IsServer, supportedStr, r.CurrentUpstreamVersion, r.DatePublished)
	}
}
