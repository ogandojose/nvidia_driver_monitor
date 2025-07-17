package main

import (
	"flag"
	"fmt"
	"log"

	"nvidia_driver_monitor/internal/web"
)

func main() {
	var addr = flag.String("addr", ":8080", "HTTP server address")
	flag.Parse()

	fmt.Printf("Starting NVIDIA Driver Package Status Web Server...\n")
	fmt.Printf("Server will be available at http://localhost%s\n", *addr)

	// Create and start web service
	webService, err := web.NewWebService()
	if err != nil {
		log.Fatalf("Failed to create web service: %v", err)
	}

	fmt.Printf("Initializing data... This may take a moment...\n")

	if err := webService.Start(*addr); err != nil {
		log.Fatalf("Failed to start web server: %v", err)
	}
}
