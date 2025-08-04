package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"nvidia_driver_monitor/internal/config"
	"nvidia_driver_monitor/internal/domain/entities"
)

// PackageRepository implements the domain repository interface for packages
type PackageRepository struct {
	httpClient HTTPClient
	config     *config.Config
}

// NewPackageRepository creates a new package repository
func NewPackageRepository(httpClient HTTPClient, cfg *config.Config) *PackageRepository {
	return &PackageRepository{
		httpClient: httpClient,
		config:     cfg,
	}
}

// GetLRMPackages fetches LRM packages for a specific kernel series
func (r *PackageRepository) GetLRMPackages(ctx context.Context, series string, sourcePackage string) ([]*entities.LaunchpadPackage, error) {
	// Query for linux-restricted-modules packages
	effectiveURLs := r.config.GetEffectiveURLs()
	url := fmt.Sprintf("%s/ubuntu/%s/+archive/primary?ws.op=getPublishedSources&source_name=%s",
		effectiveURLs.Launchpad.BaseURL, series, sourcePackage)

	resp, err := r.httpClient.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch LRM packages for series %s: %w", series, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var launchpadResponse struct {
		Entries []entities.LaunchpadPackage `json:"entries"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&launchpadResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Filter for LRM-related packages
	var lrmPackages []*entities.LaunchpadPackage
	for _, pkg := range launchpadResponse.Entries {
		if r.isLRMPackage(pkg.SourcePackageName) {
			lrmPackages = append(lrmPackages, &pkg)
		}
	}

	return lrmPackages, nil
}

// GetNvidiaDriverPackages fetches NVIDIA driver packages for a series
func (r *PackageRepository) GetNvidiaDriverPackages(ctx context.Context, series string) ([]*entities.LaunchpadPackage, error) {
	// Query for nvidia driver packages
	effectiveURLs := r.config.GetEffectiveURLs()
	url := fmt.Sprintf("%s/ubuntu/%s/+archive/primary?ws.op=getPublishedSources&source_name=nvidia-graphics-drivers",
		effectiveURLs.Launchpad.BaseURL, series)

	resp, err := r.httpClient.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch NVIDIA driver packages for series %s: %w", series, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var launchpadResponse struct {
		Entries []entities.LaunchpadPackage `json:"entries"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&launchpadResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Filter for NVIDIA driver packages
	var nvidiaPackages []*entities.LaunchpadPackage
	for _, pkg := range launchpadResponse.Entries {
		if r.isNvidiaDriverPackage(pkg.SourcePackageName) {
			nvidiaPackages = append(nvidiaPackages, &pkg)
		}
	}

	return nvidiaPackages, nil
}

// GetDSCContent fetches DSC content for a specific package
func (r *PackageRepository) GetDSCContent(ctx context.Context, dscURL string) (*entities.DSCContent, error) {
	resp, err := r.httpClient.Get(ctx, dscURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch DSC content from %s: %w", dscURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d for DSC URL %s", resp.StatusCode, dscURL)
	}

	var content strings.Builder
	buf := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			content.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}

	dscContent := &entities.DSCContent{
		Content: content.String(),
	}

	// Parse the DSC content to extract version information
	if err := r.parseDSCContent(dscContent); err != nil {
		return nil, fmt.Errorf("failed to parse DSC content: %w", err)
	}

	return dscContent, nil
}

// isLRMPackage checks if a package name indicates it's an LRM package
func (r *PackageRepository) isLRMPackage(packageName string) bool {
	lrmPrefixes := []string{
		"linux-restricted-modules",
		"linux-modules-nvidia",
		"linux-objects-nvidia",
		"linux-signatures-nvidia",
	}

	for _, prefix := range lrmPrefixes {
		if strings.HasPrefix(packageName, prefix) {
			return true
		}
	}
	return false
}

// isNvidiaDriverPackage checks if a package name indicates it's an NVIDIA driver package
func (r *PackageRepository) isNvidiaDriverPackage(packageName string) bool {
	nvidiaPatterns := []string{
		"nvidia-graphics-drivers",
		"nvidia-driver",
		"nvidia-compute-utils",
	}

	for _, pattern := range nvidiaPatterns {
		if strings.Contains(packageName, pattern) {
			return true
		}
	}
	return false
}

// parseDSCContent extracts version and metadata from DSC file content
func (r *PackageRepository) parseDSCContent(dsc *entities.DSCContent) error {
	lines := strings.Split(dsc.Content, "\n")
	var dependencies []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "Version:") {
			version := strings.TrimSpace(strings.TrimPrefix(line, "Version:"))
			dsc.Version = version
		} else if strings.HasPrefix(line, "Source:") {
			source := strings.TrimSpace(strings.TrimPrefix(line, "Source:"))
			dsc.PackageName = source
		} else if strings.HasPrefix(line, "Build-Depends:") || strings.HasPrefix(line, "Depends:") {
			deps := strings.TrimSpace(strings.TrimPrefix(line, strings.Split(line, ":")[0]+":"))
			dependencies = append(dependencies, deps)
		}
	}

	dsc.Dependencies = dependencies

	// Extract NVIDIA driver dependencies
	r.extractNvidiaDrivers(dsc)

	return nil
}

// extractNvidiaDrivers finds NVIDIA driver dependencies in the DSC content
func (r *PackageRepository) extractNvidiaDrivers(dsc *entities.DSCContent) {
	var nvidiaDrivers []entities.NvidiaDriverDependency

	for _, dep := range dsc.Dependencies {
		// Look for nvidia driver patterns in dependencies
		if strings.Contains(dep, "nvidia-graphics-drivers") || strings.Contains(dep, "nvidia-driver") {
			// Parse dependency string to extract driver name and version
			parts := strings.Split(dep, "(")
			if len(parts) >= 2 {
				name := strings.TrimSpace(parts[0])
				versionPart := strings.TrimSpace(parts[1])
				versionPart = strings.TrimSuffix(versionPart, ")")

				// Extract version from version constraint (e.g., ">= 470.256.02")
				versionFields := strings.Fields(versionPart)
				if len(versionFields) >= 2 {
					version := versionFields[1]
					nvidiaDrivers = append(nvidiaDrivers, entities.NvidiaDriverDependency{
						Name:    name,
						Version: version,
					})
				}
			}
		}
	}

	dsc.NvidiaDrivers = nvidiaDrivers
}
