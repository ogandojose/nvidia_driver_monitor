package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig   `json:"server"`
	Cache    CacheConfig    `json:"cache"`
	RateLimit RateLimitConfig `json:"rate_limit"`
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port     int  `json:"port"`
	HTTPSPort int `json:"https_port"`
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
