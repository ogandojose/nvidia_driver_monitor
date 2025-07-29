package lrm

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
	"nvidia_driver_monitor/internal/packages"
)

const (
	KernelSeriesURL = "https://kernel.ubuntu.com/forgejo/kernel/kernel-versions/raw/branch/main/info/kernel-series.yaml"
	LaunchpadAPIURL = "https://api.launchpad.net/devel/ubuntu/+archive/primary/?created_since_date=%s&exact_match=true&order_by_date=true&source_name=%s&ws.op=getPublishedSources"
	DSCCacheDir     = "/tmp/lrm-dsc-cache"
)

// DSCDownloadTask represents a task for downloading a DSC file
type DSCDownloadTask struct {
	URL         string
	Filename    string
	Package     string
	Series      string
	Release     string
	PackageName string
	Version     string
	DSCUrl      string
}

// NvidiaDriverInfo represents a parsed NVIDIA driver dependency
type NvidiaDriverInfo struct {
	DriverName string // e.g., "nvidia-graphics-drivers-470"
	Version    string // e.g., "470.256.02-0ubuntu0.24.04.1"
}

// FetchKernelLRMData fetches and processes kernel L-R-M information
func FetchKernelLRMData(routing string) (*LRMVerifierData, error) {
	log.Printf("Fetching kernel-series.yaml...")

	// Download kernel-series.yaml
	resp, err := http.Get(KernelSeriesURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download kernel-series.yaml: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read kernel-series.yaml: %v", err)
	}

	// Debug: log the first few lines to see what we got
	lines := strings.Split(string(body), "\n")
	log.Printf("Downloaded %d bytes, first few lines:", len(body))
	for i, line := range lines {
		if i >= 5 { // Only show first 5 lines
			break
		}
		log.Printf("Line %d: %s", i+1, line)
	}

	// Parse YAML
	var kernelSeries KernelSeries
	if err := yaml.Unmarshal(body, &kernelSeries); err != nil {
		return nil, fmt.Errorf("failed to parse kernel-series.yaml: %v", err)
	}

	log.Printf("Processing kernel sources...")

	// Process kernel data
	var allKernels []KernelLRMResult
	totalSources := 0
	for series, seriesInfo := range kernelSeries {
		for source, sourceInfo := range seriesInfo.Sources {
			totalSources++
			// Skip sources that don't match routing filter
			if routing != "" && sourceInfo.Routing != routing {
				continue
			}

			// Find L-R-M packages in this source
			var lrmPackages []string
			for pkgName, pkgInfo := range sourceInfo.Packages {
				if pkgInfo.Type == "lrm" {
					lrmPackages = append(lrmPackages, pkgName)
				}
			}

			// Determine final supported/development status
			supported := seriesInfo.Supported
			development := seriesInfo.Development

			if sourceInfo.Supported != nil {
				supported = *sourceInfo.Supported
			}
			if sourceInfo.Development != nil {
				development = *sourceInfo.Development
			}

			result := KernelLRMResult{
				Series:      series,
				Codename:    seriesInfo.Codename,
				Source:      source,
				Routing:     sourceInfo.Routing,
				LRMPackages: lrmPackages,
				HasLRM:      len(lrmPackages) > 0,
				Supported:   supported,
				Development: development,
				LTS:         seriesInfo.LTS,
				ESM:         seriesInfo.ESM,
			}

			allKernels = append(allKernels, result)
		}
	}

	log.Printf("Processed %d total sources, found %d kernels", totalSources, len(allKernels))

	// Filter to only supported kernels with LRM packages
	var supportedLRMKernels []KernelLRMResult
	for _, kernel := range allKernels {
		if kernel.Supported && kernel.HasLRM {
			supportedLRMKernels = append(supportedLRMKernels, kernel)
		}
	}

	log.Printf("Found %d total kernels, %d supported with LRM packages", len(allKernels), len(supportedLRMKernels))

	// Fetch latest versions for supported L-R-M kernels
	if len(supportedLRMKernels) > 0 {
		log.Printf("Querying Launchpad for latest versions...")
		supportedLRMKernels, err = fetchLatestVersions(supportedLRMKernels)
		if err != nil {
			log.Printf("Warning: Failed to fetch some versions: %v", err)
		}
	}

	// Sort by series and source for consistent display
	sort.Slice(supportedLRMKernels, func(i, j int) bool {
		if supportedLRMKernels[i].Series != supportedLRMKernels[j].Series {
			return supportedLRMKernels[i].Series < supportedLRMKernels[j].Series
		}
		return supportedLRMKernels[i].Source < supportedLRMKernels[j].Source
	})

	return &LRMVerifierData{
		KernelResults: supportedLRMKernels,
		LastUpdated:   time.Now(),
		IsInitialized: true,
		TotalKernels:  len(allKernels),
		SupportedLRM:  len(supportedLRMKernels),
	}, nil
}

// FetchKernelLRMDataDebug is like FetchKernelLRMData but returns all kernels (for debugging)
func FetchKernelLRMDataDebug(routing string) (*LRMVerifierData, error) {
	log.Printf("Fetching kernel-series.yaml...")

	// Download kernel-series.yaml
	resp, err := http.Get(KernelSeriesURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download kernel-series.yaml: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read kernel-series.yaml: %v", err)
	}

	// Parse YAML
	var kernelSeries KernelSeries
	if err := yaml.Unmarshal(body, &kernelSeries); err != nil {
		return nil, fmt.Errorf("failed to parse kernel-series.yaml: %v", err)
	}

	log.Printf("Processing kernel sources...")

	// Process kernel data
	var allKernels []KernelLRMResult
	totalSources := 0
	for series, seriesInfo := range kernelSeries {
		for source, sourceInfo := range seriesInfo.Sources {
			totalSources++
			// Apply routing filter if specified
			if routing != "" && sourceInfo.Routing != routing {
				continue
			}

			// Find L-R-M packages in this source
			var lrmPackages []string
			for pkgName, pkgInfo := range sourceInfo.Packages {
				if strings.HasPrefix(pkgName, "linux-restricted-modules-") && pkgInfo.Type == "main" {
					lrmPackages = append(lrmPackages, pkgName)
				}
			}

			// Determine final supported/development status
			supported := seriesInfo.Supported
			development := seriesInfo.Development

			if sourceInfo.Supported != nil {
				supported = *sourceInfo.Supported
			}
			if sourceInfo.Development != nil {
				development = *sourceInfo.Development
			}

			result := KernelLRMResult{
				Series:      series,
				Codename:    seriesInfo.Codename,
				Source:      source,
				Routing:     sourceInfo.Routing,
				LRMPackages: lrmPackages,
				HasLRM:      len(lrmPackages) > 0,
				Supported:   supported,
				Development: development,
				LTS:         seriesInfo.LTS,
				ESM:         seriesInfo.ESM,
			}

			allKernels = append(allKernels, result)
		}
	}

	log.Printf("Processed %d total sources, found %d kernels", totalSources, len(allKernels))

	// Return ALL kernels (no filtering)
	return &LRMVerifierData{
		KernelResults: allKernels,
		LastUpdated:   time.Now(),
		IsInitialized: true,
		TotalKernels:  len(allKernels),
		SupportedLRM:  len(allKernels), // For debug, just use total count
	}, nil
}

// fetchLatestVersions queries Launchpad API for latest package versions and NVIDIA drivers
func fetchLatestVersions(kernels []KernelLRMResult) ([]KernelLRMResult, error) {
	const maxConcurrency = 5
	const dateThreshold = "2025-01-10"

	// Step 1: Process each kernel to get LRM versions and NVIDIA driver versions
	semaphore := make(chan bool, maxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i := range kernels {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			semaphore <- true
			defer func() { <-semaphore }()

			kernel := &kernels[index]

			// Query L-R-M package version
			if len(kernel.LRMPackages) > 0 {
				version := queryPackageVersion(kernel.LRMPackages[0], kernel.Codename, dateThreshold)
				mu.Lock()
				kernel.LatestLRMVersion = version
				mu.Unlock()
			}

			// Query source package version
			sourceVersion := queryPackageVersion(kernel.Source, kernel.Codename, dateThreshold)
			mu.Lock()
			kernel.SourceVersion = sourceVersion
			mu.Unlock()

			// Get NVIDIA driver versions for this kernel from DSC files
			if kernel.LatestLRMVersion != "N/A" && kernel.LatestLRMVersion != "ERROR" {
				driverVersions := generateNvidiaDriverVersions(kernel.LRMPackages[0], kernel.LatestLRMVersion, kernel.Codename)
				mu.Lock()
				kernel.NvidiaDriverVersions = driverVersions
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	log.Printf("Completed processing all kernels for LRM and NVIDIA driver versions")

	// Step 2: Collect all unique NVIDIA driver packages that we found in DSC files
	driverPackageSet := make(map[string]bool)
	for _, kernel := range kernels {
		for _, driverStr := range kernel.NvidiaDriverVersions {
			if strings.Contains(driverStr, "=") {
				parts := strings.SplitN(driverStr, "=", 2)
				if len(parts) == 2 {
					driverPackageSet[parts[0]] = true
				}
			}
		}
	}
	log.Printf("Found %d unique NVIDIA driver packages to query DKMS versions for", len(driverPackageSet))

	// Step 3: Query DKMS versions for each unique driver package using the same logic as the main dashboard
	dkmsVersionsMap := make(map[string]map[string]string) // [packageName][series] = version
	var dkmsMu sync.Mutex
	var dkmsWg sync.WaitGroup

	for driverPackage := range driverPackageSet {
		dkmsWg.Add(1)
		go func(packageName string) {
			defer dkmsWg.Done()
			
			// Use the same function as the main dashboard to get DKMS versions
			sourceVersions, err := packages.GetMaxSourceVersionsArchive(packageName)
			if err != nil {
				log.Printf("Warning: Failed to get source versions for %s: %v", packageName, err)
				return
			}

			// Extract Updates/Security versions for each series (same logic as main dashboard)
			seriesList := []string{"questing", "plucky", "noble", "jammy", "focal", "bionic"}
			packageVersions := make(map[string]string)
			
			for _, series := range seriesList {
				if pocket, exists := sourceVersions.VersionMap[series]; exists {
					if pocket.UpdatesSecurity.String() != "" {
						packageVersions[series] = pocket.UpdatesSecurity.String()
					}
				}
			}

			dkmsMu.Lock()
			if len(packageVersions) > 0 {
				dkmsVersionsMap[packageName] = packageVersions
				log.Printf("DKMS versions for %s: %v", packageName, packageVersions)
			}
			dkmsMu.Unlock()
		}(driverPackage)
	}

	dkmsWg.Wait()
	log.Printf("Fetched DKMS versions for %d driver packages", len(dkmsVersionsMap))

	// Step 4: Update each kernel with DKMS versions and generate update status
	for i := range kernels {
		kernel := &kernels[i]
		kernel.DKMSVersions = make(map[string]string)
		
		// For each NVIDIA driver in this kernel, get the corresponding DKMS version
		for _, driverStr := range kernel.NvidiaDriverVersions {
			if strings.Contains(driverStr, "=") {
				parts := strings.SplitN(driverStr, "=", 2)
				if len(parts) == 2 {
					driverPackage := parts[0] // e.g., "nvidia-graphics-drivers-535-server"
					if driverVersions, exists := dkmsVersionsMap[driverPackage]; exists {
						if dkmsVersion, seriesExists := driverVersions[kernel.Codename]; seriesExists {
							kernel.DKMSVersions[driverPackage] = dkmsVersion
							log.Printf("Kernel %s/%s: Found DKMS version for %s: %s", kernel.Series, kernel.Source, driverPackage, dkmsVersion)
						}
					}
				}
			}
		}

		// Generate update status by comparing NVIDIA drivers with DKMS versions
		kernel.UpdateStatus = generateUpdateStatus(kernel.NvidiaDriverVersions, kernel.DKMSVersions)
		kernel.NvidiaDriverStatuses = generateNvidiaDriverStatuses(kernel.NvidiaDriverVersions, kernel.DKMSVersions)
	}

	return kernels, nil
}

// queryPackageVersion queries Launchpad API for the latest version of a package
func queryPackageVersion(packageName, codename, dateThreshold string) string {
	url := fmt.Sprintf(LaunchpadAPIURL, dateThreshold, packageName)

	log.Printf("Querying %s in %s...", packageName, codename)

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error querying %s: %v", packageName, err)
		return "ERROR"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("HTTP error for %s: %d", packageName, resp.StatusCode)
		return "ERROR"
	}

	var apiResp LaunchpadResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		log.Printf("JSON decode error for %s: %v", packageName, err)
		return "ERROR"
	}

	log.Printf("Found %d publications for %s", len(apiResp.Entries), packageName)

	// Find the latest version from the relevant series and pockets
	var latestVersion string
	var latestDate time.Time
	var pocket string

	for _, entry := range apiResp.Entries {
		if entry.Status != "Published" {
			continue
		}

		// Extract series from distro_series_link
		seriesFromLink := extractSeriesFromLink(entry.DistroSeriesLink)
		if seriesFromLink != codename {
			continue
		}

		// Consider release, updates, and security pockets (prioritize security > updates > release)
		if entry.Pocket != "Release" && entry.Pocket != "Updates" && entry.Pocket != "Security" {
			continue
		}

		// Prefer newer dates, but also prefer security/updates over release
		isNewer := entry.DatePublished.After(latestDate)
		isBetterPocket := (pocket == "Release" && (entry.Pocket == "Updates" || entry.Pocket == "Security")) ||
			(pocket == "Updates" && entry.Pocket == "Security")

		if isNewer || isBetterPocket {
			latestVersion = entry.SourcePackageVersion
			latestDate = entry.DatePublished
			pocket = entry.Pocket
			log.Printf("  → %s %s in %s (%s)", packageName, latestVersion, codename, pocket)
		}
	}

	if latestVersion == "" {
		log.Printf("No packages found for %s in %s", packageName, codename)
		return "N/A"
	}

	return fmt.Sprintf("%s (%s)", latestVersion, pocket)
}

// extractSeriesFromLink extracts the series name from a Launchpad distro series link
func extractSeriesFromLink(link string) string {
	// Link format: https://api.launchpad.net/devel/ubuntu/noble
	parts := strings.Split(link, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// generateNvidiaDriverVersions finds NVIDIA driver versions from DSC files
func generateNvidiaDriverVersions(lrmPackage, version, codename string) []string {
	if version == "N/A" || version == "ERROR" || lrmPackage == "" {
		return []string{}
	}

	log.Printf("Fetching NVIDIA driver versions for %s in %s from DSC file", lrmPackage, codename)

	// Try to find and download DSC file for this package
	dscURL, err := findDSCURL(lrmPackage, codename, version)
	if err != nil {
		log.Printf("Failed to find DSC URL for %s: %v", lrmPackage, err)
		return []string{}
	}

	// Create DSC cache directory if it doesn't exist
	err = os.MkdirAll(DSCCacheDir, 0755)
	if err != nil {
		log.Printf("Failed to create DSC cache directory: %v", err)
		return []string{}
	}

	// Generate filename for the DSC file
	filename := fmt.Sprintf("%s-%s.dsc", codename, lrmPackage)
	filePath := fmt.Sprintf("%s/%s", DSCCacheDir, filename)

	// Download DSC file if it doesn't exist
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		err = downloadDSCFile(dscURL, filename)
		if err != nil {
			log.Printf("Failed to download DSC file for %s: %v", lrmPackage, err)
			return []string{}
		}
	}

	// Parse DSC file to extract NVIDIA driver versions
	driverVersions, err := parseDSCFile(filePath)
	if err != nil {
		log.Printf("Failed to parse DSC file %s: %v", filePath, err)
		return []string{}
	}

	log.Printf("Found %d NVIDIA drivers for %s in %s: %v", len(driverVersions), lrmPackage, codename, driverVersions)
	return driverVersions
}

// extractDriverBranch extracts the driver branch from a package name
func extractDriverBranch(packageName string) string {
	prefix := "nvidia-graphics-drivers-"
	if !strings.HasPrefix(packageName, prefix) {
		return ""
	}
	return strings.TrimPrefix(packageName, prefix)
}

// StringPtr returns a pointer to a string (utility function)
func StringPtr(s string) *string {
	return &s
}

// BoolPtr returns a pointer to a bool (utility function)
func BoolPtr(b bool) *bool {
	return &b
}

// FilterKernelData filters kernel data based on criteria
func FilterKernelData(kernels []KernelLRMResult, criteria FilterCriteria) []KernelLRMResult {
	var filtered []KernelLRMResult

	for _, kernel := range kernels {
		// Apply filters
		if criteria.Development != nil && kernel.Development != *criteria.Development {
			continue
		}
		if criteria.Supported != nil && kernel.Supported != *criteria.Supported {
			continue
		}
		if criteria.HasLRM != nil && kernel.HasLRM != *criteria.HasLRM {
			continue
		}
		if criteria.Routing != nil && kernel.Routing != *criteria.Routing {
			continue
		}

		filtered = append(filtered, kernel)
	}

	return filtered
}

// GetLatestDKMSVersions queries Launchpad API for the latest NVIDIA driver packages in a release
func GetLatestDKMSVersions(release string) (map[string]string, error) {
	log.Printf("Fetching latest DKMS versions for %s", release)

	// Common NVIDIA driver packages to check
	driverPackages := []string{
		"nvidia-graphics-drivers-535",
		"nvidia-graphics-drivers-535-server",
		"nvidia-graphics-drivers-550",
		"nvidia-graphics-drivers-550-server",
		"nvidia-graphics-drivers-570",
		"nvidia-graphics-drivers-570-server",
		"nvidia-graphics-drivers-575",
		"nvidia-graphics-drivers-575-server",
		"nvidia-graphics-drivers-470",
		"nvidia-graphics-drivers-470-server",
		"nvidia-graphics-drivers-390",
	}

	dkmsVersions := make(map[string]string)
	const maxConcurrency = 5
	const dateThreshold = "2025-01-10"

	semaphore := make(chan bool, maxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, pkg := range driverPackages {
		wg.Add(1)
		go func(packageName string) {
			defer wg.Done()
			semaphore <- true
			defer func() { <-semaphore }()

			version := queryPackageVersion(packageName, release, dateThreshold)
			if version != "N/A" && version != "ERROR" {
				mu.Lock()
				dkmsVersions[packageName] = version
				mu.Unlock()
				log.Printf("Found %s = %s in %s", packageName, version, release)
			}
		}(pkg)
	}

	wg.Wait()

	log.Printf("Found %d DKMS packages for %s", len(dkmsVersions), release)
	return dkmsVersions, nil
}

// CompareDKMSVersions compares NVIDIA driver version with DKMS version and returns status
func CompareDKMSVersions(nvidiaDriver, dkmsVersion string) string {
	if dkmsVersion == "N/A" || dkmsVersion == "" {
		return "N/A"
	}

	if nvidiaDriver == "N/A" || nvidiaDriver == "" {
		return "N/A"
	}

	// Extract version from NVIDIA driver string
	nvidiaVersion := ""
	if strings.Contains(nvidiaDriver, "=") {
		parts := strings.Split(nvidiaDriver, "=")
		if len(parts) > 1 {
			nvidiaVersion = parts[1]
		}
	} else {
		nvidiaVersion = nvidiaDriver
	}

	// Compare versions
	if nvidiaVersion == dkmsVersion {
		return "✅ Latest"
	}

	// Check if DKMS version is newer
	if strings.Contains(dkmsVersion, "-") && strings.Contains(nvidiaVersion, "-") {
		// Extract base version and Ubuntu revision
		nvidiaParts := strings.Split(nvidiaVersion, "-")
		dkmsParts := strings.Split(dkmsVersion, "-")

		if len(nvidiaParts) >= 2 && len(dkmsParts) >= 2 {
			nvidiaBase := nvidiaParts[0]
			dkmsBase := dkmsParts[0]

			// If base versions are different, show update available
			if nvidiaBase != dkmsBase {
				return fmt.Sprintf("🔄 Update Available (%s)", dkmsVersion)
			}

			// If base versions are same, compare Ubuntu revisions
			nvidiaRev := strings.Join(nvidiaParts[1:], "-")
			dkmsRev := strings.Join(dkmsParts[1:], "-")

			if nvidiaRev != dkmsRev {
				return fmt.Sprintf("🔄 Update Available (%s)", dkmsVersion)
			}
		}
	}

	// Default case - show update available if versions don't match
	return fmt.Sprintf("🔄 Update Available (%s)", dkmsVersion)
}

// SimplifyNvidiaDriverName simplifies NVIDIA driver display names
func SimplifyNvidiaDriverName(fullDriverString string) string {
	if !strings.Contains(fullDriverString, "nvidia-graphics-drivers-") {
		return fullDriverString
	}

	// Split on the equals sign to separate driver name and version
	parts := strings.SplitN(fullDriverString, "=", 2)
	if len(parts) != 2 {
		return fullDriverString
	}

	driverName := parts[0]
	version := parts[1]

	// Extract the driver branch (e.g., "535", "470-server") from the full name
	prefix := "nvidia-graphics-drivers-"
	if strings.HasPrefix(driverName, prefix) {
		branch := driverName[len(prefix):]
		return fmt.Sprintf("%s=%s", branch, version)
	}

	return fullDriverString
}

// Find and download the DSC file for a given LRM package
func findDSCURL(packageName, codename, version string) (string, error) {
	// Query Launchpad API for package information
	createdSince := time.Now().AddDate(0, -6, 0).Format("2006-01-02")
	url := fmt.Sprintf(LaunchpadAPIURL, createdSince, packageName)

	log.Printf("Querying Launchpad API for %s: %s", packageName, url)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to query Launchpad API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("launchpad API returned HTTP %d", resp.StatusCode)
	}

	var launchpadResp LaunchpadResponse
	if err := json.NewDecoder(resp.Body).Decode(&launchpadResp); err != nil {
		return "", fmt.Errorf("failed to decode Launchpad response: %v", err)
	}

	// Find the entry for the specific release
	for _, entry := range launchpadResp.Entries {
		// Extract series name from distro_series_link (e.g., ".../jammy" -> "jammy")
		seriesName := extractSeriesFromLink(entry.DistroSeriesLink)
		if seriesName == codename {
			// Make a separate API call to get source file URLs
			sourceUrls, err := fetchSourceFileUrls(entry.SelfLink)
			if err != nil {
				log.Printf("Failed to fetch source URLs for %s: %v", packageName, err)
				continue
			}

			// Look for DSC files in the source URLs
			for _, fileUrl := range sourceUrls {
				if strings.HasSuffix(fileUrl, ".dsc") {
					log.Printf("Found DSC URL for %s in %s: %s", packageName, codename, fileUrl)
					return fileUrl, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no DSC file found for %s in %s", packageName, codename)
}

// fetchSourceFileUrls queries the Launchpad API to get source file URLs for a package
func fetchSourceFileUrls(selfLink string) ([]string, error) {
	// Construct the sourceFileUrls API URL from the self_link
	sourceFileUrlsURL := selfLink + "?ws.op=sourceFileUrls"

	// Make the HTTP request
	resp, err := http.Get(sourceFileUrlsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch source file URLs: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("source file URLs API returned HTTP %d", resp.StatusCode)
	}

	// Parse the JSON response - it should be an array of strings
	var sourceUrls []string
	err = json.NewDecoder(resp.Body).Decode(&sourceUrls)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source file URLs response: %v", err)
	}

	return sourceUrls, nil
}

// downloadDSCFile downloads a DSC file from a URL and saves it to the DSC cache directory
func downloadDSCFile(url, filename string) error {
	log.Printf("Downloading DSC file: %s", url)

	// Download the file
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download DSC file: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d when downloading DSC file", resp.StatusCode)
	}

	// Create the file
	filePath := fmt.Sprintf("%s/%s", DSCCacheDir, filename)
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", filePath, err)
	}
	defer file.Close()

	// Copy the response body to the file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %v", filePath, err)
	}

	log.Printf("Successfully downloaded DSC file: %s", filename)
	return nil
}

// parseDSCFile reads a DSC file and extracts NVIDIA driver dependencies
func parseDSCFile(filePath string) ([]string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read DSC file %s: %v", filePath, err)
	}

	return parseNvidiaDriverDependencies(string(content)), nil
}

// parseNvidiaDriverDependencies extracts NVIDIA driver versions from DSC content
func parseNvidiaDriverDependencies(content string) []string {
	var driverVersions []string

	// Find the Ubuntu-Nvidia-Dependencies section
	lines := strings.Split(content, "\n")
	inDependenciesSection := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if trimmedLine == "Ubuntu-Nvidia-Dependencies:" {
			inDependenciesSection = true
			continue
		}

		// Stop if we hit an empty line or PGP signature
		if inDependenciesSection && (trimmedLine == "" || strings.HasPrefix(trimmedLine, "-----")) {
			break
		}

		// Parse driver dependency lines
		if inDependenciesSection && strings.Contains(trimmedLine, "nvidia-graphics-drivers-") {
			// Remove leading/trailing whitespace and comma
			trimmedLine = strings.TrimRight(trimmedLine, ",")

			// Extract driver name and version
			// Format: "nvidia-graphics-drivers-470 (= 470.256.02-0ubuntu0.24.04.1),"
			if idx := strings.Index(trimmedLine, " (= "); idx > 0 {
				driverName := trimmedLine[:idx]
				versionPart := trimmedLine[idx+4:]
				if endIdx := strings.Index(versionPart, ")"); endIdx > 0 {
					version := versionPart[:endIdx]
					driverVersions = append(driverVersions, fmt.Sprintf("%s=%s", driverName, version))
				}
			}
		}
	}

	return driverVersions
}

// generateUpdateStatus compares NVIDIA driver versions with DKMS versions and returns status
func generateUpdateStatus(nvidiaDrivers []string, dkmsVersions map[string]string) string {
	if len(nvidiaDrivers) == 0 {
		return "N/A"
	}

	upToDateCount := 0
	updateAvailableCount := 0

	for _, driverStr := range nvidiaDrivers {
		// Parse the driver string format: "nvidia-graphics-drivers-535-server=535.247.01-0ubuntu0.22.04.1"
		if !strings.Contains(driverStr, "=") {
			continue
		}

		parts := strings.SplitN(driverStr, "=", 2)
		if len(parts) != 2 {
			continue
		}

		// The package name is already the full DKMS package name (e.g., "nvidia-graphics-drivers-535-server")
		dkmsPackageName := parts[0]
		currentVersion := parts[1]

		// Find the corresponding DKMS version
		dkmsVersion, exists := dkmsVersions[dkmsPackageName]
		if !exists {
			continue
		}

		// Extract just the version part from DKMS (remove pocket info)
		dkmsVersionParts := strings.Fields(dkmsVersion)
		if len(dkmsVersionParts) > 0 {
			dkmsVersionClean := dkmsVersionParts[0]
			
			// Compare versions
			if currentVersion == dkmsVersionClean {
				upToDateCount++
			} else {
				updateAvailableCount++
			}
		}
	}

	// Summarize the overall status
	if upToDateCount > 0 && updateAvailableCount == 0 {
		return fmt.Sprintf("✅ All up to date (%d/%d)", upToDateCount, len(nvidiaDrivers))
	} else if updateAvailableCount > 0 && upToDateCount == 0 {
		return fmt.Sprintf("🔄 Updates available (%d/%d)", updateAvailableCount, len(nvidiaDrivers))
	} else if upToDateCount > 0 && updateAvailableCount > 0 {
		return fmt.Sprintf("⚠️ Mixed (%d✅/%d🔄)", upToDateCount, updateAvailableCount)
	}

	return "N/A"
}

// generateNvidiaDriverStatuses creates individual driver status entries
func generateNvidiaDriverStatuses(nvidiaDrivers []string, dkmsVersions map[string]string) []NvidiaDriverStatus {
	var statuses []NvidiaDriverStatus

	for _, driverStr := range nvidiaDrivers {
		// Parse the driver string format: "nvidia-graphics-drivers-535-server=535.247.01-0ubuntu0.22.04.1"
		if !strings.Contains(driverStr, "=") {
			continue
		}

		parts := strings.SplitN(driverStr, "=", 2)
		if len(parts) != 2 {
			continue
		}

		driverName := parts[0]
		dscVersion := parts[1]
		
		status := NvidiaDriverStatus{
			DriverName:  driverName,
			DSCVersion:  dscVersion,
			FullString:  driverStr,
			Status:      "⚠️ Unknown",
		}

		// Find the corresponding DKMS version
		if dkmsVersion, exists := dkmsVersions[driverName]; exists {
			// Extract just the version part from DKMS (remove pocket info)
			dkmsVersionParts := strings.Fields(dkmsVersion)
			if len(dkmsVersionParts) > 0 {
				dkmsVersionClean := dkmsVersionParts[0]
				status.DKMSVersion = dkmsVersionClean
				
				// Compare versions
				if dscVersion == dkmsVersionClean {
					status.Status = "✅ Up to date"
				} else {
					status.Status = "🔄 Update available"
				}
			}
		}

		statuses = append(statuses, status)
	}

	return statuses
}
