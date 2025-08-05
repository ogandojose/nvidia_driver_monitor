package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Config holds all configuration for the application
type Config struct {
	Server       ServerConfig       `json:"server"`
	Cache        CacheConfig        `json:"cache"`
	RateLimit    RateLimitConfig    `json:"rate_limit"`
	RequestLimit RequestLimitConfig `json:"request_limit"`
	URLs         URLConfig          `json:"urls"`
	HTTP         HTTPConfig         `json:"http"`
	Testing      TestingConfig      `json:"testing"`
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port        int  `json:"port"`
	HTTPSPort   int  `json:"https_port"`
	EnableHTTPS bool `json:"enable_https"`
}

// CacheConfig holds cache-related configuration
type CacheConfig struct {
	RefreshInterval string `json:"refresh_interval"` // Duration string like "15m"
	Enabled         bool   `json:"enabled"`
}

// GetRefreshInterval parses and returns the refresh interval as time.Duration
func (c *CacheConfig) GetRefreshInterval() time.Duration {
	if c.RefreshInterval == "" {
		return 15 * time.Minute // default
	}

	duration, err := time.ParseDuration(c.RefreshInterval)
	if err != nil {
		return 15 * time.Minute // fallback to default
	}

	return duration
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	RequestsPerMinute int  `json:"requests_per_minute"`
	Enabled           bool `json:"enabled"`
}

// RequestLimitConfig holds request limiting configuration
type RequestLimitConfig struct {
	MaxBodySize    int64  `json:"max_body_size"`    // Maximum request body size in bytes
	ReadTimeout    string `json:"read_timeout"`     // Server read timeout duration
	WriteTimeout   string `json:"write_timeout"`    // Server write timeout duration  
	IdleTimeout    string `json:"idle_timeout"`     // Server idle timeout duration
	RequestTimeout string `json:"request_timeout"`  // Per-request timeout duration
	MaxHeaderBytes int    `json:"max_header_bytes"` // Maximum request header size in bytes
}

// GetReadTimeout parses and returns the read timeout as time.Duration
func (r *RequestLimitConfig) GetReadTimeout() time.Duration {
	if r.ReadTimeout == "" {
		return 15 * time.Second // default
	}
	
	duration, err := time.ParseDuration(r.ReadTimeout)
	if err != nil {
		return 15 * time.Second // fallback to default
	}
	
	return duration
}

// GetWriteTimeout parses and returns the write timeout as time.Duration
func (r *RequestLimitConfig) GetWriteTimeout() time.Duration {
	if r.WriteTimeout == "" {
		return 15 * time.Second // default
	}
	
	duration, err := time.ParseDuration(r.WriteTimeout)
	if err != nil {
		return 15 * time.Second // fallback to default
	}
	
	return duration
}

// GetIdleTimeout parses and returns the idle timeout as time.Duration
func (r *RequestLimitConfig) GetIdleTimeout() time.Duration {
	if r.IdleTimeout == "" {
		return 60 * time.Second // default
	}
	
	duration, err := time.ParseDuration(r.IdleTimeout)
	if err != nil {
		return 60 * time.Second // fallback to default
	}
	
	return duration
}

// GetRequestTimeout parses and returns the request timeout as time.Duration
func (r *RequestLimitConfig) GetRequestTimeout() time.Duration {
	if r.RequestTimeout == "" {
		return 30 * time.Second // default
	}
	
	duration, err := time.ParseDuration(r.RequestTimeout)
	if err != nil {
		return 30 * time.Second // fallback to default
	}
	
	return duration
}

// ValidateRequestLimits validates the request limits configuration
func (r *RequestLimitConfig) ValidateRequestLimits() error {
	if r.MaxBodySize < 0 {
		return fmt.Errorf("max_body_size cannot be negative")
	}
	
	if r.MaxHeaderBytes < 0 {
		return fmt.Errorf("max_header_bytes cannot be negative")
	}
	
	// Validate timeout formats by parsing them
	if r.ReadTimeout != "" {
		if _, err := time.ParseDuration(r.ReadTimeout); err != nil {
			return fmt.Errorf("invalid read_timeout format: %v", err)
		}
	}
	
	if r.WriteTimeout != "" {
		if _, err := time.ParseDuration(r.WriteTimeout); err != nil {
			return fmt.Errorf("invalid write_timeout format: %v", err)
		}
	}
	
	if r.IdleTimeout != "" {
		if _, err := time.ParseDuration(r.IdleTimeout); err != nil {
			return fmt.Errorf("invalid idle_timeout format: %v", err)
		}
	}
	
	if r.RequestTimeout != "" {
		if _, err := time.ParseDuration(r.RequestTimeout); err != nil {
			return fmt.Errorf("invalid request_timeout format: %v", err)
		}
	}
	
	return nil
}

// URLConfig holds all external URLs and API endpoints
type URLConfig struct {
	Ubuntu    UbuntuURLs    `json:"ubuntu"`
	Launchpad LaunchpadURLs `json:"launchpad"`
	NVIDIA    NVIDIAURLs    `json:"nvidia"`
	CDN       CDNURLs       `json:"cdn"`
	Kernel    KernelURLs    `json:"kernel"`
}

// UbuntuURLs holds Ubuntu-related URLs
type UbuntuURLs struct {
	AssetsBaseURL string `json:"assets_base_url"`
}

// LaunchpadURLs holds Launchpad API endpoints
type LaunchpadURLs struct {
	BaseURL              string `json:"base_url"`
	PublishedSourcesAPI  string `json:"published_sources_api"`
	PublishedBinariesAPI string `json:"published_binaries_api"`
	UbuntuSeriesBaseURL  string `json:"ubuntu_series_base_url"`
	CreatedSinceDate     string `json:"created_since_date"`
}

// GetPublishedSourcesURL constructs the full URL for published sources API
func (l *LaunchpadURLs) GetPublishedSourcesURL(sourceName string) string {
	return fmt.Sprintf("%s/?ws.op=getPublishedSources&source_name=%s&created_since_date=%s&order_by_date=true&exact_match=true",
		l.PublishedSourcesAPI, sourceName, l.CreatedSinceDate)
}

// GetPublishedBinariesURL constructs the full URL for published binaries API
func (l *LaunchpadURLs) GetPublishedBinariesURL(binaryName string) string {
	return fmt.Sprintf("%s?ws.op=getPublishedBinaries&binary_name=%s&exact_match=true",
		l.PublishedBinariesAPI, binaryName)
}

// GetUbuntuSeriesURL constructs the URL for a specific Ubuntu series
func (l *LaunchpadURLs) GetUbuntuSeriesURL(codename string) string {
	return fmt.Sprintf("%s/%s", l.UbuntuSeriesBaseURL, codename)
}

// GetTestingURLs returns URLs modified for local testing if testing is enabled
func (c *Config) GetTestingURLs() URLConfig {
	if !c.Testing.Enabled {
		return c.URLs
	}

	// Create testing URLs that point to local mock server
	mockBase := fmt.Sprintf("http://localhost:%d", c.Testing.MockServerPort)

	return URLConfig{
		Ubuntu: UbuntuURLs{
			AssetsBaseURL: fmt.Sprintf("%s/ubuntu/assets", mockBase),
		},
		Launchpad: LaunchpadURLs{
			BaseURL:              fmt.Sprintf("%s/launchpad", mockBase),
			PublishedSourcesAPI:  fmt.Sprintf("%s/launchpad/ubuntu/+archive/primary", mockBase),
			PublishedBinariesAPI: fmt.Sprintf("%s/launchpad/ubuntu/+archive/primary", mockBase),
			UbuntuSeriesBaseURL:  fmt.Sprintf("%s/launchpad/ubuntu", mockBase),
			CreatedSinceDate:     c.URLs.Launchpad.CreatedSinceDate,
		},
		NVIDIA: NVIDIAURLs{
			DriverArchiveURL: fmt.Sprintf("%s/nvidia/drivers", mockBase),
			ServerDriversAPI: fmt.Sprintf("%s/nvidia/datacenter/releases.json", mockBase),
		},
		CDN: c.URLs.CDN, // Keep CDN URLs as-is for styling
		Kernel: KernelURLs{
			SeriesYAMLURL: fmt.Sprintf("%s/kernel/series.yaml", mockBase),
			SRUCycleURL:   fmt.Sprintf("%s/kernel/sru-cycle.yaml", mockBase),
		},
	}
}

// GetEffectiveURLs returns the URLs that should be used (testing or production)
func (c *Config) GetEffectiveURLs() URLConfig {
	if c.Testing.Enabled {
		return c.GetTestingURLs()
	}
	return c.URLs
}

// NVIDIAURLs holds NVIDIA-related URLs
type NVIDIAURLs struct {
	DriverArchiveURL string `json:"driver_archive_url"`
	ServerDriversAPI string `json:"server_drivers_api"`
}

// CDNURLs holds CDN and external library URLs
type CDNURLs struct {
	BootstrapCSS string `json:"bootstrap_css"`
	BootstrapJS  string `json:"bootstrap_js"`
	ChartJS      string `json:"chart_js"`
	VanillaCSS   string `json:"vanilla_css"`
}

// KernelURLs holds kernel-related URLs
type KernelURLs struct {
	SeriesYAMLURL string `json:"series_yaml_url"`
	SRUCycleURL   string `json:"sru_cycle_url"`
}

// HTTPConfig holds HTTP client configuration
type HTTPConfig struct {
	Timeout   string `json:"timeout"`        // Duration string like "10s"
	Retries   int    `json:"retries"`
	UserAgent string `json:"user_agent"`
}

// TestingConfig holds testing/mock service configuration
type TestingConfig struct {
	Enabled        bool   `json:"enabled"`
	MockServerPort int    `json:"mock_server_port"`
	DataDir        string `json:"data_dir"`
}

// GetTimeout parses and returns the timeout as time.Duration
func (h *HTTPConfig) GetTimeout() time.Duration {
	if h.Timeout == "" {
		return 10 * time.Second // default
	}

	duration, err := time.ParseDuration(h.Timeout)
	if err != nil {
		return 10 * time.Second // fallback to default
	}

	return duration
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:        8080,
			HTTPSPort:   8443,
			EnableHTTPS: false,
		},
		Cache: CacheConfig{
			RefreshInterval: "15m",
			Enabled:         true,
		},
		RateLimit: RateLimitConfig{
			RequestsPerMinute: 60,
			Enabled:           true,
		},
		RequestLimit: RequestLimitConfig{
			MaxBodySize:    1048576, // 1MB
			ReadTimeout:    "15s",
			WriteTimeout:   "15s",
			IdleTimeout:    "60s",
			RequestTimeout: "30s",
			MaxHeaderBytes: 1048576, // 1MB
		},
		URLs: URLConfig{
			Ubuntu: UbuntuURLs{
				AssetsBaseURL: "https://assets.ubuntu.com/v1",
			},
			Launchpad: LaunchpadURLs{
				BaseURL:              "https://api.launchpad.net/devel",
				PublishedSourcesAPI:  "https://api.launchpad.net/devel/ubuntu/+archive/primary",
				PublishedBinariesAPI: "https://api.launchpad.net/devel/ubuntu/+archive/primary",
				UbuntuSeriesBaseURL:  "https://api.launchpad.net/devel/ubuntu",
				CreatedSinceDate:     "2025-01-10",
			},
			NVIDIA: NVIDIAURLs{
				DriverArchiveURL: "https://www.nvidia.com/en-us/drivers/unix/linux-amd64-display-archive/",
				ServerDriversAPI: "https://docs.nvidia.com/datacenter/tesla/drivers/releases.json",
			},
			CDN: CDNURLs{
				BootstrapCSS: "https://cdn.jsdelivr.net/npm/bootstrap@5.1.3/dist/css/bootstrap.min.css",
				BootstrapJS:  "https://cdn.jsdelivr.net/npm/bootstrap@5.1.3/dist/js/bootstrap.bundle.min.js",
				ChartJS:      "https://cdn.jsdelivr.net/npm/chart.js@3.9.1/dist/chart.min.js",
				VanillaCSS:   "https://assets.ubuntu.com/v1/vanilla-framework-version-4.15.0.min.css",
			},
			Kernel: KernelURLs{
				SeriesYAMLURL: "https://kernel.ubuntu.com/forgejo/kernel/kernel-versions/raw/branch/main/info/kernel-series.yaml",
				SRUCycleURL:   "https://kernel.ubuntu.com/forgejo/kernel/kernel-versions/raw/branch/main/info/sru-cycle.yaml",
			},
		},
		HTTP: HTTPConfig{
			Timeout:   "10s",
			Retries:   5,
			UserAgent: "nvidia-driver-monitor/1.0",
		},
		Testing: TestingConfig{
			Enabled:        false,
			MockServerPort: 9999,
			DataDir:        "test-data",
		},
	}
}

// LoadConfig loads configuration from a file
func LoadConfig(configPath string) (*Config, error) {
	config := DefaultConfig()

	if configPath == "" {
		return config, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil // Use defaults if file doesn't exist
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

// SaveConfig saves configuration to a file
func SaveConfig(config *Config, configPath string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
