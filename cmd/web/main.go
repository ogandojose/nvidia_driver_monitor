package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"nvidia_driver_monitor/internal/config"
	"nvidia_driver_monitor/internal/web"
)

func main() {
	var addr = flag.String("addr", ":8080", "Server address")
	var enableHTTPS = flag.Bool("https", false, "Enable HTTPS with self-signed certificate")
	var certFile = flag.String("cert", "server.crt", "Certificate file path (for HTTPS)")
	var keyFile = flag.String("key", "server.key", "Private key file path (for HTTPS)")
	var configFile = flag.String("config", "config.json", "Configuration file path")
	var rateLimit = flag.Int("rate-limit", 0, "Rate limit (requests per minute, 0 to use config)")
	var templateDir = flag.String("templates", "templates", "Templates directory path")
	flag.Parse()

	fmt.Printf("Starting NVIDIA Driver Package Status Web Server...\n")

	// Load configuration
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Override config with command line flags
	if *rateLimit > 0 {
		cfg.RateLimit.RequestsPerMinute = *rateLimit
	}

	// Create template path
	templatePath, err := filepath.Abs(*templateDir)
	if err != nil {
		log.Fatalf("Failed to resolve template directory: %v", err)
	}

	// Create and start web service with configuration
	webService, err := web.NewWebServiceWithConfig(cfg, templatePath)
	if err != nil {
		log.Fatalf("Failed to create web service: %v", err)
	}

	// Configure HTTPS if requested
	if *enableHTTPS || cfg.Server.EnableHTTPS {
		webService.EnableHTTPS = true
		webService.CertFile = *certFile
		webService.KeyFile = *keyFile
		fmt.Printf("HTTPS mode enabled\n")
		if cfg.Server.EnableHTTPS {
			*addr = fmt.Sprintf(":%d", cfg.Server.HTTPSPort)
		}
		fmt.Printf("Server will be available at https://localhost%s\n", *addr)
	} else {
		if *addr == ":8080" && cfg.Server.Port != 8080 {
			*addr = fmt.Sprintf(":%d", cfg.Server.Port)
		}
		fmt.Printf("HTTP mode (use -https flag for HTTPS)\n")
		fmt.Printf("Server will be available at http://localhost%s\n", *addr)
	}

	fmt.Printf("Configuration loaded: Rate limit: %d req/min, Cache refresh: %v\n", 
		cfg.RateLimit.RequestsPerMinute, cfg.Cache.RefreshInterval)
	fmt.Printf("Initializing data... This may take a moment...\n")

	if err := webService.Start(*addr); err != nil {
		log.Fatalf("Failed to start web server: %v", err)
	}
}
