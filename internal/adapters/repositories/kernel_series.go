package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"nvidia_driver_monitor/internal/config"
	"nvidia_driver_monitor/internal/domain/entities"

	"gopkg.in/yaml.v2"
)

// KernelSeriesRepository implements the domain repository interface
type KernelSeriesRepository struct {
	httpClient HTTPClient
	config     *config.Config
}

// HTTPClient interface for HTTP operations
type HTTPClient interface {
	Get(ctx context.Context, url string) (*http.Response, error)
}

// NewKernelSeriesRepository creates a new kernel series repository
func NewKernelSeriesRepository(httpClient HTTPClient, cfg *config.Config) *KernelSeriesRepository {
	return &KernelSeriesRepository{
		httpClient: httpClient,
		config:     cfg,
	}
}

// GetSupportedSeries fetches supported Ubuntu kernel series from kernel.ubuntu.com
func (r *KernelSeriesRepository) GetSupportedSeries(ctx context.Context) ([]*entities.KernelSeries, error) {
	effectiveURLs := r.config.GetEffectiveURLs()
	url := effectiveURLs.Kernel.SeriesYAMLURL

	resp, err := r.httpClient.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch kernel series: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var yamlData map[string]struct {
		Codename    string `yaml:"codename"`
		Development bool   `yaml:"development"`
		Supported   bool   `yaml:"supported"`
		LTS         bool   `yaml:"lts"`
		ESM         bool   `yaml:"esm"`
	}

	if err := yaml.NewDecoder(resp.Body).Decode(&yamlData); err != nil {
		return nil, fmt.Errorf("failed to decode YAML response: %w", err)
	}

	var series []*entities.KernelSeries
	for version, entry := range yamlData {
		// Only include supported series or development series
		if entry.Supported || entry.Development {
			series = append(series, &entities.KernelSeries{
				Name:        version,
				Codename:    entry.Codename,
				Development: entry.Development,
				Supported:   entry.Supported,
				LTS:         entry.LTS,
				ESM:         entry.ESM,
				Sources:     make(map[string]entities.KernelSource),
			})
		}
	}

	return series, nil
}

// GetKernelsForSeries fetches available kernels for a specific Ubuntu series
func (r *KernelSeriesRepository) GetKernelsForSeries(ctx context.Context, seriesName string) ([]*entities.Kernel, error) {
	// First, we need to get the codename for this series
	// The seriesName could be either a version (like "24.04") or codename (like "noble")
	codename := seriesName

	// If seriesName looks like a version number, we need to map it to codename
	if isVersionNumber(seriesName) {
		// Get all supported series to find the codename
		allSeries, err := r.GetSupportedSeries(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get series list to resolve codename for %s: %w", seriesName, err)
		}

		// Find the codename for this version
		found := false
		for _, series := range allSeries {
			if series.Name == seriesName {
				codename = series.Codename
				found = true
				break
			}
		}

		if !found {
			return nil, fmt.Errorf("could not find codename for series %s", seriesName)
		}
	}

	// Use the same API format that works for NVIDIA packages
	effectiveURLs := r.config.GetEffectiveURLs()
	url := effectiveURLs.Launchpad.GetPublishedSourcesURL("linux")

	resp, err := r.httpClient.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch kernels for series %s: %w", seriesName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch kernels for series %s: client error: %d %s", seriesName, resp.StatusCode, resp.Status)
	}

	var launchpadResponse struct {
		Entries []struct {
			SourcePackageName    string `json:"source_package_name"`
			SourcePackageVersion string `json:"source_package_version"`
			Status               string `json:"status"`
			DatePublished        string `json:"date_published"`
			DistroSeriesLink     string `json:"distro_series_link"`
		} `json:"entries"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&launchpadResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Build the expected distro series link for filtering
	expectedSeriesLink := effectiveURLs.Launchpad.GetUbuntuSeriesURL(codename)

	var kernels []*entities.Kernel
	for _, entry := range launchpadResponse.Entries {
		if entry.SourcePackageName == "linux" && entry.Status == "Published" && entry.DistroSeriesLink == expectedSeriesLink {
			// Extract kernel version from package version
			version := r.extractKernelVersion(entry.SourcePackageVersion)

			kernel := &entities.Kernel{
				Series:        seriesName,
				Source:        entry.SourcePackageName,
				SourceVersion: version,
				Supported:     true,
				Development:   false,
				LTS:           false, // Would need to determine this from series info
				ESM:           false,
				LRMPackages:   []entities.LRMPackage{},
				NvidiaDrivers: []entities.NvidiaDriver{},
				UpdateStatus:  entry.Status,
			}
			kernels = append(kernels, kernel)
		}
	}

	return kernels, nil
}

// extractKernelVersion extracts the kernel version from the package version string
func (r *KernelSeriesRepository) extractKernelVersion(packageVersion string) string {
	// Package version format is typically like "5.15.0-97.107"
	// We want to extract "5.15.0-97"
	parts := strings.Split(packageVersion, ".")
	if len(parts) >= 3 {
		return strings.Join(parts[:3], ".")
	}
	return packageVersion
}

// isVersionNumber checks if the given string is a Ubuntu version number (like "24.04", "22.04")
func isVersionNumber(s string) bool {
	// Ubuntu version format is typically YY.MM (like "24.04", "22.04")
	return strings.Contains(s, ".") && len(s) <= 6 && strings.Count(s, ".") == 1
}
