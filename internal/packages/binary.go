package packages

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

	version "github.com/knqyf263/go-deb-version"
)

// BinaryAPIResponse represents the JSON response for binary packages
type BinaryAPIResponse struct {
	Start     int                `json:"start"`
	TotalSize int                `json:"total_size"`
	Entries   []BinaryPubHistory `json:"entries"`
}

// BinaryPubHistory represents a binary package publication history entry
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

// BinaryVersionPerPocket holds binary package versions per pocket and architecture
type BinaryVersionPerPocket struct {
	Amd64UpdatesSecurity version.Version
	Amd64Proposed        version.Version
	Arm64UpdatesSecurity version.Version
	Arm64Proposed        version.Version
	I386UpdatesSecurity  version.Version
	I386Proposed         version.Version
}

// BinaryVersionPerSeries holds binary package versions per series
type BinaryVersionPerSeries struct {
	PackageName string
	VersionMap  map[string]*BinaryVersionPerPocket
}

// SeriesArchFromDistroArchSeriesLink extracts series and architecture from distro_arch_series_link
func SeriesArchFromDistroArchSeriesLink(s string) (string, string) {
	parts := strings.Split(strings.TrimRight(s, "/"), "/")
	if len(parts) < 2 {
		return "", ""
	}
	return parts[len(parts)-2], parts[len(parts)-1]
}

// GetMaxBinaryVersionsArchive retrieves the maximum binary package versions from archive
func GetMaxBinaryVersionsArchive(packageName string) (*BinaryVersionPerSeries, error) {
	if packageName == "" {
		return nil, fmt.Errorf("package name cannot be empty")
	}

	url := fmt.Sprintf("https://api.launchpad.net/devel/ubuntu/+archive/primary?ws.op=getPublishedBinaries&binary_name=%s&exact_match=true", packageName)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}
	defer resp.Body.Close()

	var apiResp BinaryAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	log.Printf("Found %d entries for binary package %s", len(apiResp.Entries), packageName)

	versionMap := make(map[string]*BinaryVersionPerPocket)

	for _, entry := range apiResp.Entries {
		if entry.Status != "Published" {
			continue
		}

		series, arch := SeriesArchFromDistroArchSeriesLink(entry.ArchitectureSeries)
		if series == "" || arch == "" {
			continue
		}

		ver, err := version.NewVersion(entry.BinaryPackageVersion)
		if err != nil {
			log.Printf("Invalid version %s for %s: %v", entry.BinaryPackageVersion, packageName, err)
			continue
		}

		if versionMap[series] == nil {
			versionMap[series] = &BinaryVersionPerPocket{}
		}

		pocket := versionMap[series]

		switch entry.Pocket {
		case "Updates", "Security":
			switch arch {
			case "amd64":
				if pocket.Amd64UpdatesSecurity.String() == "" || ver.GreaterThan(pocket.Amd64UpdatesSecurity) {
					pocket.Amd64UpdatesSecurity = ver
				}
			case "arm64":
				if pocket.Arm64UpdatesSecurity.String() == "" || ver.GreaterThan(pocket.Arm64UpdatesSecurity) {
					pocket.Arm64UpdatesSecurity = ver
				}
			case "i386":
				if pocket.I386UpdatesSecurity.String() == "" || ver.GreaterThan(pocket.I386UpdatesSecurity) {
					pocket.I386UpdatesSecurity = ver
				}
			}
		case "Proposed":
			switch arch {
			case "amd64":
				if pocket.Amd64Proposed.String() == "" || ver.GreaterThan(pocket.Amd64Proposed) {
					pocket.Amd64Proposed = ver
				}
			case "arm64":
				if pocket.Arm64Proposed.String() == "" || ver.GreaterThan(pocket.Arm64Proposed) {
					pocket.Arm64Proposed = ver
				}
			case "i386":
				if pocket.I386Proposed.String() == "" || ver.GreaterThan(pocket.I386Proposed) {
					pocket.I386Proposed = ver
				}
			}
		}
	}

	return &BinaryVersionPerSeries{
		PackageName: packageName,
		VersionMap:  versionMap,
	}, nil
}

// PrintBinaryVersionMapTable prints the binary version map in table format
func PrintBinaryVersionMapTable(bvps *BinaryVersionPerSeries) {
	fmt.Printf("Binary Package: %s\n", bvps.PackageName)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Series\tAMD64 Updates/Security\tAMD64 Proposed\tARM64 Updates/Security\tARM64 Proposed\tI386 Updates/Security\tI386 Proposed")

	for series, pocket := range bvps.VersionMap {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			series,
			pocket.Amd64UpdatesSecurity.String(),
			pocket.Amd64Proposed.String(),
			pocket.Arm64UpdatesSecurity.String(),
			pocket.Arm64Proposed.String(),
			pocket.I386UpdatesSecurity.String(),
			pocket.I386Proposed.String())
	}

	w.Flush()
	fmt.Println("----------------------------------------------------")
}
