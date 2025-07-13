package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

// Struct matching the new JSON structure
type SupportedRelease struct {
	BranchName             string `json:"branch_name"`
	IsServer               bool   `json:"is_server"`
	IsSupported            bool   `json:"is_supported"`
	CurrentUpstreamVersion string `json:"current_upstream_version"`
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

// PrintSupportedReleases prints the array of SupportedRelease to stdout
func PrintSupportedReleases(releases []SupportedRelease) {
	fmt.Printf("Loaded %d releases:\n", len(releases))
	for _, r := range releases {
		fmt.Printf("%+v\n", r)
	}
}

// LogSupportedReleases logs the array of SupportedRelease using the standard logger
func LogSupportedReleases(releases []SupportedRelease) {
	fmt.Printf("Loaded %d releases:\n", len(releases))
	for _, r := range releases {
		fmt.Printf("%+v\n", r)
	}
}
