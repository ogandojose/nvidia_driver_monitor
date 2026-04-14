package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"nvidia_driver_monitor/internal/config"
)

func main() {
	var (
		configFile = flag.String("config", "config.json", "Path to configuration file")
		generate   = flag.Bool("generate", false, "Generate default configuration file")
		testing    = flag.Bool("testing", false, "Generate configuration with testing mode enabled")
		validate   = flag.Bool("validate", false, "Validate configuration file")
		show       = flag.Bool("show", false, "Show current configuration")
	)
	flag.Parse()

	if *generate {
		generateConfig(*configFile, *testing)
		return
	}

	if *validate {
		validateConfig(*configFile)
		return
	}

	if *show {
		showConfig(*configFile)
		return
	}

	flag.Usage()
}

func generateConfig(configFile string, testing bool) {
	cfg := config.DefaultConfig()

	if testing {
		cfg.Testing.Enabled = true
		cfg.Testing.MockServerPort = 9999
		cfg.Testing.DataDir = "test-data"
		cfg.Server.Port = 8080
		cfg.Server.HTTPSPort = 8443
		cfg.Cache.Enabled = true
		cfg.Cache.RefreshInterval = "10s"
		cfg.HTTP.Timeout = "5s"
		cfg.RateLimit.Enabled = true
		cfg.RateLimit.RequestsPerMinute = 100
		cfg.HTTP.Retries = 3
		cfg.HTTP.UserAgent = "NVIDIA-Driver-Monitor/Testing"
		cfg.Processing.MaxConcurrency = 5
	}

	if err := config.SaveConfig(cfg, configFile); err != nil {
		log.Fatalf("Failed to generate config file: %v", err)
	}

	fmt.Printf("✅ Generated default configuration file: %s\n", configFile)
	if testing {
		fmt.Println("\n🧪 Testing mode enabled!")
		fmt.Printf("   - Mock server port: %d\n", cfg.Testing.MockServerPort)
		fmt.Printf("   - Test data directory: %s\n", cfg.Testing.DataDir)
		fmt.Println("   - All external APIs will use local mock server")
		fmt.Println("\n💡 To use:")
		fmt.Println("   1. Start mock server: make run-mock")
		fmt.Println("   2. Start web server: ./nvidia-web-server -config " + configFile)
	} else {
		fmt.Println("\nConfiguration includes:")
		fmt.Println("  - Server settings (ports, HTTPS)")
		fmt.Println("  - Cache and rate limiting settings")
		fmt.Println("  - All external URLs and API endpoints")
		fmt.Println("  - HTTP client configuration")
	}
	fmt.Printf("\nEdit %s to customize settings for your environment.\n", configFile)
}

func validateConfig(configFile string) {
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("❌ Configuration validation failed: %v", err)
	}

	// Basic validation checks
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		log.Fatalf("❌ Invalid server port: %d", cfg.Server.Port)
	}

	if cfg.Server.HTTPSPort <= 0 || cfg.Server.HTTPSPort > 65535 {
		log.Fatalf("❌ Invalid HTTPS port: %d", cfg.Server.HTTPSPort)
	}

	if cfg.Cache.RefreshInterval == "" {
		log.Fatalf("❌ Cache refresh interval cannot be empty")
	}

	if cfg.HTTP.Timeout == "" {
		log.Fatalf("❌ HTTP timeout cannot be empty")
	}

	if cfg.Processing.MaxConcurrency < 0 || cfg.Processing.MaxConcurrency > 50 {
		log.Fatalf("❌ Processing max concurrency must be 0 for default or between 1 and 50")
	}

	// Validate request limits
	if err := cfg.RequestLimit.ValidateRequestLimits(); err != nil {
		log.Fatalf("❌ Request limits validation failed: %v", err)
	}

	// Validate duration parsing
	cfg.Cache.GetRefreshInterval()    // Just call it to test
	cfg.HTTP.GetTimeout()             // Just call it to test
	cfg.RequestLimit.GetReadTimeout() // Test request limit timeouts
	cfg.RequestLimit.GetWriteTimeout()
	cfg.RequestLimit.GetIdleTimeout()
	cfg.RequestLimit.GetRequestTimeout()

	fmt.Printf("✅ Configuration file %s is valid\n", configFile)
}

func showConfig(configFile string) {
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	fmt.Printf("Configuration from: %s\n", configFile)
	fmt.Println("=" + strings.Repeat("=", len(configFile)+19))

	fmt.Printf("\n🔧 Server Configuration:\n")
	fmt.Printf("  HTTP Port:  %d\n", cfg.Server.Port)
	fmt.Printf("  HTTPS Port: %d\n", cfg.Server.HTTPSPort)
	fmt.Printf("  HTTPS Enabled: %t\n", cfg.Server.EnableHTTPS)

	fmt.Printf("\n💾 Cache Configuration:\n")
	fmt.Printf("  Enabled: %t\n", cfg.Cache.Enabled)
	fmt.Printf("  Refresh Interval: %s (%v)\n", cfg.Cache.RefreshInterval, cfg.Cache.GetRefreshInterval())

	fmt.Printf("\n🚦 Rate Limiting:\n")
	fmt.Printf("  Enabled: %t\n", cfg.RateLimit.Enabled)
	fmt.Printf("  Requests per minute: %d\n", cfg.RateLimit.RequestsPerMinute)

	fmt.Printf("\n⏱️ Request Limits:\n")
	fmt.Printf("  Max Body Size: %d bytes (%.1f MB)\n", cfg.RequestLimit.MaxBodySize, float64(cfg.RequestLimit.MaxBodySize)/1048576)
	fmt.Printf("  Read Timeout: %s (%v)\n", cfg.RequestLimit.ReadTimeout, cfg.RequestLimit.GetReadTimeout())
	fmt.Printf("  Write Timeout: %s (%v)\n", cfg.RequestLimit.WriteTimeout, cfg.RequestLimit.GetWriteTimeout())
	fmt.Printf("  Idle Timeout: %s (%v)\n", cfg.RequestLimit.IdleTimeout, cfg.RequestLimit.GetIdleTimeout())
	fmt.Printf("  Request Timeout: %s (%v)\n", cfg.RequestLimit.RequestTimeout, cfg.RequestLimit.GetRequestTimeout())
	fmt.Printf("  Max Header Bytes: %d bytes (%.1f KB)\n", cfg.RequestLimit.MaxHeaderBytes, float64(cfg.RequestLimit.MaxHeaderBytes)/1024)

	fmt.Printf("\n🌐 HTTP Configuration:\n")
	fmt.Printf("  Timeout: %s (%v)\n", cfg.HTTP.Timeout, cfg.HTTP.GetTimeout())
	fmt.Printf("  Retries: %d\n", cfg.HTTP.Retries)
	fmt.Printf("  User Agent: %s\n", cfg.HTTP.UserAgent)

	fmt.Printf("\n⚙️ Processing Configuration:\n")
	fmt.Printf("  Max concurrency: %d\n", cfg.Processing.GetMaxConcurrency())

	fmt.Printf("\n🔗 External URLs:\n")
	fmt.Printf("  Ubuntu Assets: %s\n", cfg.URLs.Ubuntu.AssetsBaseURL)
	fmt.Printf("  Launchpad API: %s\n", cfg.URLs.Launchpad.BaseURL)
	fmt.Printf("  NVIDIA Archive: %s\n", cfg.URLs.NVIDIA.DriverArchiveURL)
	fmt.Printf("  Kernel Series: %s\n", cfg.URLs.Kernel.SeriesYAMLURL)
	fmt.Printf("  SRU Cycles: %s\n", cfg.URLs.Kernel.SRUCycleURL)

	fmt.Printf("\n📚 CDN Libraries:\n")
	fmt.Printf("  Bootstrap CSS: %s\n", cfg.URLs.CDN.BootstrapCSS)
	fmt.Printf("  Bootstrap JS:  %s\n", cfg.URLs.CDN.BootstrapJS)
	fmt.Printf("  Chart.js:      %s\n", cfg.URLs.CDN.ChartJS)
}
