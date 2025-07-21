package main

import (
	"flag"
	"fmt"
	"log"

	"nvidia_driver_monitor/internal/web"
)

func main() {
	var addr = flag.String("addr", ":8080", "Server address")
	var enableHTTPS = flag.Bool("https", false, "Enable HTTPS with self-signed certificate")
	var certFile = flag.String("cert", "server.crt", "Certificate file path (for HTTPS)")
	var keyFile = flag.String("key", "server.key", "Private key file path (for HTTPS)")
	flag.Parse()

	fmt.Printf("Starting NVIDIA Driver Package Status Web Server...\n")

	// Create and start web service
	webService, err := web.NewWebService()
	if err != nil {
		log.Fatalf("Failed to create web service: %v", err)
	}

	// Configure HTTPS if requested
	if *enableHTTPS {
		webService.EnableHTTPS = true
		webService.CertFile = *certFile
		webService.KeyFile = *keyFile
		fmt.Printf("HTTPS mode enabled\n")
		fmt.Printf("Server will be available at https://localhost%s\n", *addr)
	} else {
		fmt.Printf("HTTP mode (use -https flag for HTTPS)\n")
		fmt.Printf("Server will be available at http://localhost%s\n", *addr)
	}

	fmt.Printf("Initializing data... This may take a moment...\n")

	if err := webService.Start(*addr); err != nil {
		log.Fatalf("Failed to start web server: %v", err)
	}
}
