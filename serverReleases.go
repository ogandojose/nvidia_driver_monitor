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
	log.Println("These are the published ERD Driver Versions:")
	header := fmt.Sprintf(
		"%-8s  %-18s  %-12s  %-10s  %-12s  %s",
		"BRANCH", "TYPE", "VERSION", "DATE", "ARCH", "RUNFILE_URL",
	)
	log.Println(header)
	for version, info := range data {
		branch := branches[version]
		for arch, url := range info.RunfileURL {
			log.Printf(
				"%-8s  %-18s  %-12s  %-10s  %-12s  %s",
				version, branch.Type, info.ReleaseVersion, info.ReleaseDate, arch, url,
			)
		}
	}
	log.Println("----------------------------------------------------")
}

func printDriverVersions(data map[string]DriverInfo, branches AllBranches) {
	fmt.Println("These are the published ERD Driver versions:")
	fmt.Printf(
		"%-8s  %-18s  %-12s  %-10s  %-12s  %s\n",
		"BRANCH", "TYPE", "VERSION", "DATE", "ARCH", "RUNFILE_URL",
	)
	for version, info := range data {
		branch := branches[version]
		for arch, url := range info.RunfileURL {
			fmt.Printf(
				"%-8s  %-18s  %-12s  %-10s  %-12s  %s\n",
				version, branch.Type, info.ReleaseVersion, info.ReleaseDate, arch, url,
			)
		}
	}
	fmt.Println("----------------------------------------------------")
}

func UpdateSupportedReleasesWithLatestERD(branches map[string]BranchEntry, releases []SupportedRelease) []SupportedRelease {
	for i := range releases {
		rel := &releases[i]
		if len(rel.BranchName) > 7 && rel.BranchName[len(rel.BranchName)-7:] == "-server" {
			// Extract the 3 digit number preceding "-server"
			branchNum := rel.BranchName[:len(rel.BranchName)-7]
			if branch, ok := branches[branchNum]; ok && len(branch.DriverInfo) > 0 {
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
	return releases
}
