package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	version "github.com/knqyf263/go-deb-version"
)

type VersionPerPocket struct {
	amd64_updates_security version.Version
	amd64_proposed         version.Version
	//Can add arm later?
}

type VersionPerSeries struct {
	PackageName string
	VersionMap  map[string]*VersionPerPocket
}

// Structs for decoding JSON response
type APIResponse struct {
	Start     int                `json:"start"`
	TotalSize int                `json:"total_size"`
	Entries   []BinaryPubHistory `json:"entries"`
}

type BinaryPubHistory struct {
	DisplayName          string `json:"display_name"`
	BinaryPackageName    string `json:"binary_package_name"`
	BinaryPackageVersion string `json:"binary_package_version"`
	ArchitectureSeries   string `json:"distro_arch_series_link"`
	DatePublished        string `json:"date_published"`
	Pocket               string `json:"pocket"`
	Status               string `json:"status"`
	BuildLink            string `json:"build_link"`
	ComponentName        string `json:"component_name"`
	SectionName          string `json:"section_name"`
	SourcePackageName    string `json:"source_package_name"`
	SourcePackageVersion string `json:"source_package_version"`
}

func SeriesArchFromDistroArchSeriesLink(s string) (string, string) {
	parts := strings.Split(strings.TrimRight(s, "/"), "/")
	if len(parts) < 2 {
		return "", ""
	}
	return parts[len(parts)-2], parts[len(parts)-1]
}

func getMaxVersionsArchive(packageName string) {

	//url := "https://api.launchpad.net/devel/ubuntu/+archive/primary/?ws.op=getPublishedBinaries&binary_name=nvidia-utils-535-server&created_since_date=2024-01-01&order_by_date=true"
	url := fmt.Sprintf("https://api.launchpad.net/devel/ubuntu/+archive/primary/?ws.op=getPublishedBinaries&binary_name=%s&created_since_date=2024-01-01&order_by_date=true", packageName)

	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Unexpected status code: %d", resp.StatusCode)
	}

	var result APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
	}

	fmt.Printf("📦 Found %d binary publications:\n\n", result.TotalSize)

	//versionsPerSeries := make(map[string]string)

	var maxVersionPerSeries VersionPerSeries

	maxVersionPerSeries.PackageName = packageName

	var currSeries, currArch string
	for _, entry := range result.Entries {
		fmt.Printf("🧱 %s\n", entry.DisplayName)
		fmt.Printf("  → Version:     %s\n", entry.BinaryPackageVersion)
		fmt.Printf("  → Series/Arch: %s\n", entry.ArchitectureSeries)
		fmt.Printf("  → Published:   %s\n", entry.DatePublished)
		fmt.Printf("  → Pocket:      %s | Status: %s\n", entry.Pocket, entry.Status)
		fmt.Printf("  → Build Link:  %s\n", entry.BuildLink)
		fmt.Printf("  → Source:      %s (%s)\n", entry.SourcePackageName, entry.SourcePackageVersion)
		fmt.Printf("  → Component:   %s | Section: %s\n", entry.ComponentName, entry.SectionName)
		fmt.Println()

		currSeries, currArch = SeriesArchFromDistroArchSeriesLink(entry.ArchitectureSeries)

		fmt.Printf("CurrSeries: %s, CurrArch: %s\n", currSeries, currArch)

		currVersion, err := version.NewVersion(entry.BinaryPackageVersion)
		if err != nil {
			fmt.Printf("Error in incoming BinaryPackageVersion %s\n", err)
		}

		_, valueExists := maxVersionPerSeries.VersionMap[currSeries]
		//_, valueExists := versionsPerSeries[currSeries]
		if !valueExists {
			fmt.Printf("This series: %s is empty so assigning as max value\n", currSeries)
			//versionsPerSeries[currSeries] = entry.BinaryPackageVersion
			maxVersionPerSeries.VersionMap[currSeries] = &VersionPerPocket{}
			currVersion, _ = version.NewVersion(entry.BinaryPackageVersion)
			maxVersionPerSeries.VersionMap[currSeries].amd64_updates_security = currVersion
			maxVersionPerSeries.VersionMap[currSeries].amd64_proposed = currVersion
		}

		currMaxVersion := maxVersionPerSeries.VersionMap[currSeries].amd64_updates_security

		if err != nil {
			fmt.Printf("Error %s", err)
		}

		if currVersion.GreaterThan(currMaxVersion) {
			maxVersionPerSeries.VersionMap[currSeries].amd64_updates_security = currVersion
		} else {
			fmt.Printf("%s is not greater than %s\n", &currVersion, &currMaxVersion)
		}

		//versionsPerSeries[currSeries] = entry.BinaryPackageVersion
	}

	fmt.Printf("Function done\n")
}

func main() {
	url := "https://api.launchpad.net/devel/ubuntu/+archive/primary/?ws.op=getPublishedBinaries&binary_name=nvidia-utils-535-server&created_since_date=2024-01-01&order_by_date=true"

	versionsPerSeries := make(map[string]string)

	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Unexpected status code: %d", resp.StatusCode)
	}

	var result APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
	}

	fmt.Printf("📦 Found %d binary publications:\n\n", result.TotalSize)

	var currSeries, currArch string
	for _, entry := range result.Entries {
		fmt.Printf("🧱 %s\n", entry.DisplayName)
		fmt.Printf("  → Version:     %s\n", entry.BinaryPackageVersion)
		fmt.Printf("  → Series/Arch: %s\n", entry.ArchitectureSeries)
		fmt.Printf("  → Published:   %s\n", entry.DatePublished)
		fmt.Printf("  → Pocket:      %s | Status: %s\n", entry.Pocket, entry.Status)
		fmt.Printf("  → Build Link:  %s\n", entry.BuildLink)
		fmt.Printf("  → Source:      %s (%s)\n", entry.SourcePackageName, entry.SourcePackageVersion)
		fmt.Printf("  → Component:   %s | Section: %s\n", entry.ComponentName, entry.SectionName)
		fmt.Println()

		currSeries, currArch = SeriesArchFromDistroArchSeriesLink(entry.ArchitectureSeries)

		fmt.Printf("CurrSeries: %s, CurrArch: %s\n", currSeries, currArch)

		currVersion, err := version.NewVersion(entry.BinaryPackageVersion)
		if err != nil {
			fmt.Printf("Error in incoming BinaryPackageVersion %s\n", err)
		}

		_, valueExists := versionsPerSeries[currSeries]
		if !valueExists {
			fmt.Printf("This series: %s is empty so assigning as max value\n", currSeries)
			versionsPerSeries[currSeries] = entry.BinaryPackageVersion
		}

		currMaxVersion, err := version.NewVersion(versionsPerSeries[currSeries])
		if err != nil {
			fmt.Printf("Error %s", err)
		}

		if currVersion.GreaterThan(currMaxVersion) {
			versionsPerSeries[currSeries] = entry.BinaryPackageVersion
		} else {
			fmt.Printf("%s is not greater than %s\n", entry.BinaryPackageVersion, versionsPerSeries[currSeries])
		}

		//versionsPerSeries[currSeries] = entry.BinaryPackageVersion
	}

	for index, version := range versionsPerSeries {
		fmt.Printf("Series %s Version %s\n", index, version)
	}
}
