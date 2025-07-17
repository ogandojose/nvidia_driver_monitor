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
	arm64_updates_security version.Version
	arm64_proposed         version.Version
	i386_updates_security  version.Version
	i386_proposed          version.Version
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

func getMaxVersionsArchive(packageName string) (maxVersionPerSeries VersionPerSeries, retErr error) {

	var currSeries, currArch string
	var result APIResponse
	/* We */
	url := fmt.Sprintf("https://api.launchpad.net/devel/ubuntu/+archive/primary/?ws.op=getPublishedBinaries&binary_name=%s&created_since_date=2024-01-01&order_by_date=true&exact_match=true", packageName)

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

	log.Printf("📦 Found %d binary publications:\n\n", result.TotalSize)

	/* reserve space for the pointer map */
	maxVersionPerSeries = VersionPerSeries{
		PackageName: packageName,
		VersionMap:  make(map[string]*VersionPerPocket),
	}

	for _, entry := range result.Entries {
		log.Printf("🧱 %s\n", entry.DisplayName)
		log.Printf("  → Version:     %s\n", entry.BinaryPackageVersion)
		log.Printf("  → Series/Arch: %s\n", entry.ArchitectureSeries)
		log.Printf("  → Published:   %s\n", entry.DatePublished)
		log.Printf("  → Pocket:      %s | Status: %s\n", entry.Pocket, entry.Status)
		log.Printf("  → Build Link:  %s\n", entry.BuildLink)
		log.Printf("  → Source:      %s (%s)\n", entry.SourcePackageName, entry.SourcePackageVersion)
		log.Printf("  → Component:   %s | Section: %s\n", entry.ComponentName, entry.SectionName)
		log.Println()

		currSeries, currArch = SeriesArchFromDistroArchSeriesLink(entry.ArchitectureSeries)

		log.Printf("CurrSeries: %s, CurrArch: %s\n", currSeries, currArch)

		currVersion, err := version.NewVersion(entry.BinaryPackageVersion)
		if err != nil {
			log.Printf("Error in incoming BinaryPackageVersion %s\n", err)
			retErr = err
			return maxVersionPerSeries, retErr
		}

		/* Check that the version per pocket exists for the currSeries, if not create it */
		_, valueExists := maxVersionPerSeries.VersionMap[currSeries]
		if !valueExists {
			log.Printf("This series: %s is empty, creating a VersionPerPocket\n", currSeries)
			maxVersionPerSeries.VersionMap[currSeries] = &VersionPerPocket{}
			currVersion, _ = version.NewVersion(entry.BinaryPackageVersion)
			maxVersionPerSeries.VersionMap[currSeries].amd64_updates_security = currVersion
			maxVersionPerSeries.VersionMap[currSeries].amd64_proposed = currVersion
		}

		/* This next section can be refactored in a cleaner way, leaving it for clarity */
		switch entry.Pocket {
		case "Proposed":
			switch currArch {
			case "amd64":
				currMaxVersion := maxVersionPerSeries.VersionMap[currSeries].amd64_proposed
				if currVersion.GreaterThan(currMaxVersion) {
					maxVersionPerSeries.VersionMap[currSeries].amd64_proposed = currVersion
				} else {
					log.Printf("%s is not greater than %s\n", &currVersion, &currMaxVersion)
				}
			case "arm64":
				currMaxVersion := maxVersionPerSeries.VersionMap[currSeries].arm64_proposed
				if currVersion.GreaterThan(currMaxVersion) {
					maxVersionPerSeries.VersionMap[currSeries].arm64_proposed = currVersion
				} else {
					log.Printf("%s is not greater than %s\n", &currVersion, &currMaxVersion)
				}
			case "i386":
				currMaxVersion := maxVersionPerSeries.VersionMap[currSeries].i386_proposed
				if currVersion.GreaterThan(currMaxVersion) {
					maxVersionPerSeries.VersionMap[currSeries].i386_proposed = currVersion
				} else {
					log.Printf("%s is not greater than %s\n", &currVersion, &currMaxVersion)
				}
			default:
				log.Printf("Error , invalid architecture %s\n", currArch)
			}

		case "Updates", "Security":
			currMaxVersion := maxVersionPerSeries.VersionMap[currSeries].amd64_updates_security
			if currVersion.GreaterThan(currMaxVersion) {
				maxVersionPerSeries.VersionMap[currSeries].amd64_updates_security = currVersion
			} else {
				log.Printf("%s is not greater than %s\n", &currVersion, &currMaxVersion)
			}

			switch currArch {
			case "amd64":
				currMaxVersion := maxVersionPerSeries.VersionMap[currSeries].amd64_updates_security
				if currVersion.GreaterThan(currMaxVersion) {
					maxVersionPerSeries.VersionMap[currSeries].amd64_updates_security = currVersion
				} else {
					log.Printf("%s is not greater than %s\n", &currVersion, &currMaxVersion)
				}
			case "arm64":
				currMaxVersion := maxVersionPerSeries.VersionMap[currSeries].arm64_updates_security
				if currVersion.GreaterThan(currMaxVersion) {
					maxVersionPerSeries.VersionMap[currSeries].arm64_updates_security = currVersion
				} else {
					log.Printf("%s is not greater than %s\n", &currVersion, &currMaxVersion)
				}
			case "i386":
				currMaxVersion := maxVersionPerSeries.VersionMap[currSeries].i386_updates_security
				if currVersion.GreaterThan(currMaxVersion) {
					maxVersionPerSeries.VersionMap[currSeries].i386_updates_security = currVersion
				} else {
					log.Printf("%s is not greater than %s\n", &currVersion, &currMaxVersion)
				}
			default:
				log.Printf("Error , invalid architecture %s\n", currArch)
			}
		default:

		}

		// currMaxVersion := maxVersionPerSeries.VersionMap[currSeries].amd64_updates_security

		// if currVersion.GreaterThan(currMaxVersion) {
		// 	maxVersionPerSeries.VersionMap[currSeries].amd64_updates_security = currVersion
		// } else {
		// 	log.Printf("%s is not greater than %s\n", &currVersion, &currMaxVersion)
		// }

		//versionsPerSeries[currSeries] = entry.BinaryPackageVersion
	}

	return maxVersionPerSeries, nil
}

// func PrintVersionMapTable(vps VersionPerSeries) {
// 	fmt.Printf("Package: %s\n", vps.PackageName)
// 	fmt.Printf("| %-30s | %-40s | %-40s |\n", "Series", "amd64_updates_security", "amd64_proposed")
// 	fmt.Println("|--------------------------------|------------------------------------------|------------------------------------------|")

// 	for series, pocket := range vps.VersionMap {
// 		updates := "-"
// 		proposed := "-"

// 		if pocket != nil {
// 			if pocket.amd64_updates_security.String() != "" {
// 				updates = pocket.amd64_updates_security.String()
// 			}
// 			if pocket.amd64_proposed.String() != "" {
// 				proposed = pocket.amd64_proposed.String()
// 			}
// 		}

// 		fmt.Printf("| %-30s | %-40s | %-40s |\n", series, updates, proposed)
// 	}
// }

func PrintVersionMapTable(vps VersionPerSeries) {
	fmt.Printf("Package: %s\n", vps.PackageName)
	fmt.Printf(
		"| %-30s | %-42s | %-42s | %-42s | %-42s | %-42s | %-42s |\n",
		"Series",
		"amd64_updates_security",
		"amd64_proposed",
		"arm64_updates_security",
		"arm64_proposed",
		"i386_updates_security",
		"i386_proposed",
	)

	fmt.Println("|--------------------------------|--------------------------------------------|--------------------------------------------|--------------------------------------------|--------------------------------------------|--------------------------------------------|--------------------------------------------|")

	for series, pocket := range vps.VersionMap {
		amd64Upd := "-"
		amd64Prop := "-"
		arm64Upd := "-"
		arm64Prop := "-"
		i386Upd := "-"
		i386Prop := "-"

		if pocket != nil {
			if s := pocket.amd64_updates_security.String(); s != "" {
				amd64Upd = s
			}
			if s := pocket.amd64_proposed.String(); s != "" {
				amd64Prop = s
			}
			if s := pocket.arm64_updates_security.String(); s != "" {
				arm64Upd = s
			}
			if s := pocket.arm64_proposed.String(); s != "" {
				arm64Prop = s
			}
			if s := pocket.i386_updates_security.String(); s != "" {
				i386Upd = s
			}
			if s := pocket.i386_proposed.String(); s != "" {
				i386Prop = s
			}
		}

		fmt.Printf(
			"| %-30s | %-42s | %-42s | %-42s | %-42s | %-42s | %-42s |\n",
			series,
			amd64Upd,
			amd64Prop,
			arm64Upd,
			arm64Prop,
			i386Upd,
			i386Prop,
		)
	}
}
