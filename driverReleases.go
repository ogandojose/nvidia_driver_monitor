package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

type BranchInfo struct {
	BranchName            string    `json:"branch_name"`
	IsServer              bool      `json:"is_server"`
	IsSupported           bool      `json:"is_supported"`
	LatestUpstreamVersion string    `json:"latest_upstream_version"`
	DatePublished         time.Time `json:"date_published"`
}

func readSupportedReleasesJson() ([]BranchInfo, error) {
	// Read the JSON file
	data, err := os.ReadFile("supportedReleases.json")
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	// Unmarshal into slice of BranchInfo
	var releases []BranchInfo
	err = json.Unmarshal(data, &releases)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	return releases, nil
}

func logReleases(releases []BranchInfo) {
	for _, r := range releases {
		log.Printf(
			"Branch: %s | Server: %v | Supported: %v | Version: %s | Date: %s",
			r.BranchName,
			r.IsServer,
			r.IsSupported,
			r.LatestUpstreamVersion,
			r.DatePublished.Format(time.RFC3339),
		)
	}
}

func printReleases(releases []BranchInfo) {
	for _, r := range releases {
		fmt.Printf(
			"Branch: %s | Server: %v | Supported: %v | Version: %s | Date: %s\n",
			r.BranchName,
			r.IsServer,
			r.IsSupported,
			r.LatestUpstreamVersion,
			r.DatePublished.Format(time.RFC3339),
		)
	}
}

// UpdateSupportedERDReleases updates the CurrentUpstreamVersion in supported based on driverInfo.
// It uses the SupportedRelease and DriverInfo structs from ServerReleases.go.
import (
	. "path/to/ServerReleases" // Adjust the import path as needed
)

func UpdateSupportedERDReleases(supported []SupportedRelease, driverInfo []DriverInfo) {
	branchMap := make(map[string]string)
	for _, d := range driverInfo {
		if d.IsServer {
			branchMap[d.BranchName+"-server"] = d.LatestUpstreamVersion
		}
	}

	for i, s := range supported {
		if ver, ok := branchMap[s.BranchName]; ok {
			supported[i].CurrentUpstreamVersion = ver
		}
	}
}
