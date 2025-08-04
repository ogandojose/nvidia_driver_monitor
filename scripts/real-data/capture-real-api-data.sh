#!/bin/bash

# Real API Data Capture Script
# Runs the application normally and captures all real API responses

set -e

echo "🔍 REAL API DATA CAPTURE"
echo "========================"
echo "This script will run the application normally and capture real API responses"
echo ""

cd "$(dirname "$0")"

# Create capture directory
CAPTURE_DIR="captured-real-data"
mkdir -p "$CAPTURE_DIR"/{launchpad/{sources,binaries,series},nvidia,kernel,ubuntu}

echo "📂 Created capture directory: $CAPTURE_DIR"

# Kill any existing servers
pkill -f nvidia-web-server 2>/dev/null || true
pkill -f nvidia-mock-server 2>/dev/null || true
sleep 2

# Build the application
echo "🔨 Building application..."
make web > /dev/null 2>&1

# Create a special config that logs all HTTP requests
cat > config-capture.json << 'EOF'
{
  "urls": {
    "launchpad": {
      "base": "https://api.launchpad.net/devel",
      "archive_primary": "https://api.launchpad.net/devel/ubuntu/+archive/primary"
    },
    "nvidia": {
      "datacenter_releases": "https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2204/x86_64",
      "driver_archive": "https://www.nvidia.com/drivers"
    },
    "kernel": {
      "series": "https://kernel.ubuntu.com/api/kernel/series.yaml",
      "sru_cycle": "https://kernel.ubuntu.com/api/sru-cycle.yaml"
    },
    "ubuntu": {
      "base": "https://api.launchpad.net/devel/ubuntu"
    }
  },
  "testing": {
    "enabled": false
  },
  "cache_dir": "./cache-capture",
  "server": {
    "addr": ":8080",
    "read_timeout": "30s",
    "write_timeout": "30s"
  },
  "http": {
    "timeout": "30s",
    "max_retries": 3,
    "retry_delay": "2s",
    "user_agent": "nvidia-driver-monitor-capture/1.0"
  }
}
EOF

echo "⚙️ Created capture configuration"

# Start the web server in capture mode
echo "🚀 Starting web server to capture real API calls..."
echo "   This will make real API calls to gather actual data"
echo ""

# Create a simple HTTP proxy/logger to capture requests
cat > http-capture-proxy.go << 'EOF'
package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func main() {
	// Create capture directory
	captureDir := "captured-real-data"
	
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Parse the target URL from query parameter
		targetURL := r.URL.Query().Get("url")
		if targetURL == "" {
			http.Error(w, "Missing url parameter", http.StatusBadRequest)
			return
		}
		
		log.Printf("📥 Capturing: %s", targetURL)
		
		// Make the real request
		resp, err := http.Get(targetURL)
		if err != nil {
			log.Printf("❌ Error fetching %s: %v", targetURL, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()
		
		// Read response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("❌ Error reading response: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		// Save the response to file
		filename := generateFilename(targetURL)
		filepath := filepath.Join(captureDir, filename)
		
		// Create directory if needed
		dir := filepath.Dir(filepath)
		os.MkdirAll(dir, 0755)
		
		err = os.WriteFile(filepath, body, 0644)
		if err != nil {
			log.Printf("⚠️ Error saving file %s: %v", filepath, err)
		} else {
			log.Printf("💾 Saved: %s", filepath)
		}
		
		// Forward the response
		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
	})
	
	log.Println("🔍 HTTP Capture Proxy starting on :9998")
	log.Fatal(http.ListenAndServe(":9998", nil))
}

func generateFilename(targetURL string) string {
	u, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Sprintf("unknown-%d.txt", time.Now().Unix())
	}
	
	// Determine the type and generate appropriate filename
	if strings.Contains(u.Host, "launchpad.net") {
		if strings.Contains(u.Path, "+archive/primary") {
			query := u.Query()
			op := query.Get("ws.op")
			
			if op == "getPublishedSources" {
				sourceName := query.Get("source_name")
				if sourceName != "" {
					return fmt.Sprintf("launchpad/sources/%s.json", sourceName)
				}
			} else if op == "getPublishedBinaries" {
				binaryName := query.Get("binary_name")
				if binaryName != "" {
					return fmt.Sprintf("launchpad/binaries/%s.json", binaryName)
				}
			}
		} else if strings.Contains(u.Path, "/ubuntu/") && !strings.Contains(u.Path, "+archive") {
			// Ubuntu series info
			re := regexp.MustCompile(`/ubuntu/([^/]+)/?$`)
			matches := re.FindStringSubmatch(u.Path)
			if len(matches) > 1 {
				return fmt.Sprintf("launchpad/series/%s.json", matches[1])
			}
		}
	} else if strings.Contains(u.Host, "nvidia.com") || strings.Contains(u.Host, "developer.download.nvidia.com") {
		if strings.Contains(u.Path, "releases.json") {
			return "nvidia/server-drivers.json"
		}
		return "nvidia/driver-archive.html"
	} else if strings.Contains(u.Host, "kernel.ubuntu.com") {
		if strings.Contains(u.Path, "series.yaml") {
			return "kernel/series.yaml"
		} else if strings.Contains(u.Path, "sru-cycle.yaml") {
			return "kernel/sru-cycle.yaml"
		}
	}
	
	// Fallback filename
	clean := strings.ReplaceAll(u.Host+u.Path, "/", "_")
	clean = strings.ReplaceAll(clean, "?", "_")
	clean = strings.ReplaceAll(clean, "&", "_")
	clean = strings.ReplaceAll(clean, "=", "_")
	return fmt.Sprintf("misc/%s.txt", clean)
}
EOF

echo "🔧 Created HTTP capture proxy"

# Build and start the capture proxy
echo "🔨 Building capture proxy..."
go build -o http-capture-proxy http-capture-proxy.go

echo "🚀 Starting capture proxy..."
./http-capture-proxy > capture-proxy.log 2>&1 &
PROXY_PID=$!
sleep 2

# Verify proxy is running
if ! ps -p $PROXY_PID > /dev/null; then
    echo "❌ Capture proxy failed to start"
    cat capture-proxy.log
    exit 1
fi

echo "✅ Capture proxy started (PID: $PROXY_PID)"
echo ""

# Now we need to trigger real API calls by using the application
echo "🎯 Triggering real API calls through the application..."
echo ""

# Start the web server normally
./nvidia-web-server -config config-capture.json > web-server-capture.log 2>&1 &
WEB_PID=$!
sleep 3

# Verify web server is running
if ! ps -p $WEB_PID > /dev/null; then
    echo "❌ Web server failed to start"
    cat web-server-capture.log
    kill $PROXY_PID 2>/dev/null || true
    exit 1
fi

echo "✅ Web server started (PID: $WEB_PID)"
echo ""

# Make requests to trigger API calls
echo "📡 Making requests to trigger real API calls..."
echo ""

# Give the server a moment to be ready
sleep 2

# Trigger various endpoints that make API calls
requests=(
    "http://localhost:8080/"                                           # Main page
    "http://localhost:8080/package?package=nvidia-graphics-drivers-570" # Specific driver
    "http://localhost:8080/package?package=nvidia-graphics-drivers-535" # Another driver
    "http://localhost:8080/lrm"                                        # LRM verifier
    "http://localhost:8080/api/packages"                              # API endpoint
)

for url in "${requests[@]}"; do
    echo "📞 Requesting: $url"
    curl -s "$url" > /dev/null 2>&1 || echo "   ⚠️ Request may have failed"
    sleep 3  # Give time for API calls to complete
done

echo ""
echo "⏳ Waiting for all API calls to complete..."
sleep 10

# Stop servers
echo "🛑 Stopping servers..."
kill $WEB_PID 2>/dev/null || true
kill $PROXY_PID 2>/dev/null || true

# Wait for clean shutdown
sleep 3

echo ""
echo "📊 CAPTURE RESULTS"
echo "=================="

if [ -d "$CAPTURE_DIR" ]; then
    file_count=$(find "$CAPTURE_DIR" -type f | wc -l)
    echo "📁 Captured files: $file_count"
    echo ""
    echo "📋 File listing:"
    find "$CAPTURE_DIR" -type f | sort | while read file; do
        size=$(stat -c%s "$file" 2>/dev/null || echo "0")
        echo "  📄 $file (${size} bytes)"
    done
    echo ""
    
    if [ $file_count -gt 0 ]; then
        echo "✅ SUCCESS: Real API data captured!"
        echo ""
        echo "🔧 Next steps:"
        echo "   1. Review captured data in: $CAPTURE_DIR"
        echo "   2. Run: ./setup-real-mock-data.sh"
        echo "   3. Test with: make test-coverage"
    else
        echo "⚠️ WARNING: No files were captured"
        echo "   Check the logs for issues:"
        echo "   - web-server-capture.log"
        echo "   - capture-proxy.log"
    fi
else
    echo "❌ ERROR: Capture directory not created"
fi

# Cleanup
rm -f http-capture-proxy http-capture-proxy.go config-capture.json
rm -f capture-proxy.log web-server-capture.log

echo ""
echo "🎯 Real API data capture completed!"
