package lrm

import "time"

// KernelSeries represents the top-level structure of the kernel-series.yaml file
type KernelSeries map[string]SeriesInfo

// KernelLRMResult represents a kernel source with L-R-M package information
type KernelLRMResult struct {
	Series               string
	Codename             string
	Source               string
	Routing              string
	LRMPackages          []string
	HasLRM               bool
	Supported            bool
	Development          bool
	LTS                  bool
	ESM                  bool
	LatestLRMVersion     string
	SourceVersion        string
	NvidiaDriverVersions []string
	NvidiaDriversFromDSC []string          // New field to store actual driver versions from DSC files
	DKMSVersions         map[string]string // DKMS package versions for this kernel's series
	UpdateStatus         string
	NvidiaDriverStatuses []NvidiaDriverStatus // Individual driver statuses with detailed info
}

// LRMVerifierData holds all the cached L-R-M data
type LRMVerifierData struct {
	KernelResults []KernelLRMResult
	LastUpdated   time.Time
	IsInitialized bool
	TotalKernels  int
	SupportedLRM  int
}

// SeriesInfo represents information about a kernel series from kernel-series.yaml
type SeriesInfo struct {
	Codename    string                `yaml:"codename"`
	Development bool                  `yaml:"development"`
	Supported   bool                  `yaml:"supported"`
	LTS         bool                  `yaml:"lts"`
	ESM         bool                  `yaml:"esm"`
	Sources     map[string]SourceInfo `yaml:"sources"`
}

// SourceInfo represents information about a kernel source
type SourceInfo struct {
	Routing          string                 `yaml:"routing"`
	Versions         []string               `yaml:"versions"`
	Variants         []string               `yaml:"variants"`
	PackageRelations string                 `yaml:"package-relations"`
	Packages         map[string]PackageInfo `yaml:"packages"`
	// These can override series-level settings
	Supported   *bool `yaml:"supported,omitempty"`
	Development *bool `yaml:"development,omitempty"`
}

// PackageInfo represents information about a package
type PackageInfo struct {
	Type string   `yaml:"type"`
	Repo []string `yaml:"repo"`
}

// FilterCriteria represents the filtering options for L-R-M data
type FilterCriteria struct {
	Development *bool   // nil = no filter, true = only development, false = only non-development
	Supported   *bool   // nil = no filter, true = only supported, false = only non-supported
	HasLRM      *bool   // nil = no filter, true = only with LRM, false = only without LRM
	Routing     *string // nil = no filter, non-nil = filter by specific routing
}

// LaunchpadPackageEntry represents a package entry from Launchpad API
type LaunchpadPackageEntry struct {
	SelfLink             string          `json:"self_link"`
	DisplayName          string          `json:"display_name"`
	SourcePackageName    string          `json:"source_package_name"`
	SourcePackageVersion string          `json:"source_package_version"`
	Status               string          `json:"status"`
	Pocket               string          `json:"pocket"`
	DatePublished        time.Time       `json:"date_published"`
	DistroSeriesLink     string          `json:"distro_series_link"`
	SourcePackageLink    string          `json:"source_package_link"`
	BuildLink            string          `json:"build_link"`
	Files                []LaunchpadFile `json:"files"`
}

// LaunchpadFile represents a file entry from Launchpad API
type LaunchpadFile struct {
	SelfLink string `json:"self_link"`
	FileLink string `json:"file_link"`
	FileType string `json:"file_type"`
}

// LaunchpadResponse represents the response from Launchpad API
type LaunchpadResponse struct {
	Entries []LaunchpadPackageEntry `json:"entries"`
}

// NvidiaDriverStatus represents the status of an individual NVIDIA driver
type NvidiaDriverStatus struct {
	DriverName  string // e.g., "nvidia-graphics-drivers-535"
	DSCVersion  string // Version from DSC file
	DKMSVersion string // Version from DKMS/Updates-Security
	Status      string // "✅ Up to date", "🔄 Update available", "⚠️ Unknown"
	FullString  string // Full driver string with version for display
}
