#!/bin/bash

# Simple Real API Data Capture
# Uses the application's existing HTTP client with logging enabled

set -e

echo "🔍 REAL API DATA CAPTURE (Simple Method)"
echo "========================================"
echo ""

cd "$(dirname "$0")"

# Create capture directory
CAPTURE_DIR="captured-real-data"
rm -rf "$CAPTURE_DIR" 2>/dev/null || true
mkdir -p "$CAPTURE_DIR"/{launchpad/{sources,binaries,series},nvidia,kernel}

echo "📂 Created capture directory: $CAPTURE_DIR"

# Kill any existing servers
pkill -f nvidia-web-server 2>/dev/null || true
pkill -f nvidia-mock-server 2>/dev/null || true
sleep 2

# Build the application if needed
if [ ! -f "../nvidia-web-server" ]; then
    echo "🔨 Building application..."
    cd .. && make web && cd test-data
fi

echo "🎯 Making direct API calls to capture real data..."
echo ""

# Function to capture API response
capture_api() {
    local url="$1"
    local filename="$2"
    local description="$3"
    
    echo "📡 Capturing: $description"
    echo "   URL: $url"
    echo "   File: $filename"
    
    # Create directory if needed
    mkdir -p "$(dirname "$CAPTURE_DIR/$filename")"
    
    # Make the API call and save response
    if curl -s -H "User-Agent: nvidia-driver-monitor-capture/1.0" "$url" > "$CAPTURE_DIR/$filename" 2>/dev/null; then
        size=$(stat -c%s "$CAPTURE_DIR/$filename" 2>/dev/null || echo "0")
        if [ "$size" -gt 10 ]; then
            echo "   ✅ Success (${size} bytes)"
        else
            echo "   ⚠️ Small response (${size} bytes)"
        fi
    else
        echo "   ❌ Failed"
        rm -f "$CAPTURE_DIR/$filename"
    fi
    echo ""
}

# Capture NVIDIA driver source packages
echo "📦 Capturing NVIDIA driver source packages..."
NVIDIA_PACKAGES=(
    "nvidia-graphics-drivers-535"
    "nvidia-graphics-drivers-535-server"
    "nvidia-graphics-drivers-550"
    "nvidia-graphics-drivers-550-server"
    "nvidia-graphics-drivers-570"
    "nvidia-graphics-drivers-570-server"
    "nvidia-graphics-drivers-575"
    "nvidia-graphics-drivers-575-server"
    "nvidia-graphics-drivers-470"
    "nvidia-graphics-drivers-470-server"
    "nvidia-graphics-drivers-390"
    "nvidia-graphics-drivers-460"
    "nvidia-graphics-drivers-450"
    "nvidia-graphics-drivers-465"
)

for package in "${NVIDIA_PACKAGES[@]}"; do
    url="https://api.launchpad.net/devel/ubuntu/+archive/primary?ws.op=getPublishedSources&source_name=${package}&created_since_date=2024-01-01&order_by_date=true&exact_match=true"
    capture_api "$url" "launchpad/sources/${package}.json" "NVIDIA $package sources"
    sleep 1  # Be nice to the API
done

# Capture generic NVIDIA query
url="https://api.launchpad.net/devel/ubuntu/+archive/primary?ws.op=getPublishedSources&source_name=nvidia-graphics-drivers&created_since_date=2024-01-01&order_by_date=true"
capture_api "$url" "launchpad/sources/nvidia-graphics-drivers.json" "Generic NVIDIA drivers"
sleep 1

# Capture LRM packages
echo "📦 Capturing LRM packages..."
LRM_PACKAGES=(
    "linux-restricted-modules"
    "linux-restricted-modules-aws"
    "linux-restricted-modules-azure"
    "linux-restricted-modules-gcp"
    "linux-restricted-modules-gke"
    "linux-restricted-modules-oem"
    "linux-restricted-modules-raspi"
    "linux"
)

for package in "${LRM_PACKAGES[@]}"; do
    url="https://api.launchpad.net/devel/ubuntu/+archive/primary?ws.op=getPublishedSources&source_name=${package}&created_since_date=2024-01-01&order_by_date=true&exact_match=true"
    capture_api "$url" "launchpad/sources/${package}.json" "LRM $package sources"
    sleep 1
done

# Capture binary packages
echo "📦 Capturing binary packages..."
BINARY_PACKAGES=(
    "nvidia-driver-535"
    "nvidia-driver-550"
    "nvidia-driver-570"
    "nvidia-driver-575"
    "nvidia-driver-470"
    "nvidia-driver-390"
    "nvidia-driver-460"
    "nvidia-driver-450"
    "nvidia-driver-465"
)

for package in "${BINARY_PACKAGES[@]}"; do
    url="https://api.launchpad.net/devel/ubuntu/+archive/primary?ws.op=getPublishedBinaries&binary_name=${package}&exact_match=true"
    capture_api "$url" "launchpad/binaries/${package}.json" "Binary $package"
    sleep 1
done

# Capture Ubuntu series information
echo "🐧 Capturing Ubuntu series information..."
UBUNTU_SERIES=(
    "noble"      # 24.04 LTS
    "jammy"      # 22.04 LTS
    "focal"      # 20.04 LTS
    "oracular"   # 24.10
    "mantic"     # 23.10
    "lunar"      # 23.04
    "kinetic"    # 22.10
    "plucky"     # 25.04 (future)
    "questing"   # 25.10 (future)
    "bionic"     # 18.04 LTS
)

for series in "${UBUNTU_SERIES[@]}"; do
    url="https://api.launchpad.net/devel/ubuntu/${series}"
    capture_api "$url" "launchpad/series/${series}.json" "Ubuntu $series info"
    sleep 1
done

# Capture NVIDIA APIs
echo "🎮 Capturing NVIDIA APIs..."
# Note: The actual NVIDIA API might be different, let's try a few variants
url="https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2204/x86_64/releases.json"
capture_api "$url" "nvidia/server-drivers.json" "NVIDIA server drivers"

# Try alternative NVIDIA URLs
url="https://api.nvidia.com/datacenter/releases.json"
capture_api "$url" "nvidia/datacenter-releases.json" "NVIDIA datacenter releases"

# Capture Kernel APIs
echo "🐧 Capturing Kernel APIs..."
url="https://kernel.ubuntu.com/api/kernel/series.yaml"
capture_api "$url" "kernel/series.yaml" "Kernel series"

url="https://kernel.ubuntu.com/api/sru-cycle.yaml"
capture_api "$url" "kernel/sru-cycle.yaml" "SRU cycles"

echo ""
echo "📊 CAPTURE RESULTS"
echo "=================="

file_count=$(find "$CAPTURE_DIR" -type f | wc -l)
total_size=$(find "$CAPTURE_DIR" -type f -exec stat -c%s {} \; | awk '{sum+=$1} END {print sum}')

echo "📁 Total files captured: $file_count"
echo "💾 Total size: $(( total_size / 1024 )) KB"
echo ""

if [ $file_count -gt 0 ]; then
    echo "📋 Captured files by category:"
    echo ""
    
    echo "📦 NVIDIA Source Packages:"
    find "$CAPTURE_DIR/launchpad/sources" -name "nvidia-*.json" 2>/dev/null | sort | while read file; do
        size=$(stat -c%s "$file" 2>/dev/null || echo "0")
        echo "  ✅ $(basename "$file") (${size} bytes)"
    done
    
    echo ""
    echo "📦 LRM Source Packages:"
    find "$CAPTURE_DIR/launchpad/sources" -name "linux*.json" 2>/dev/null | sort | while read file; do
        size=$(stat -c%s "$file" 2>/dev/null || echo "0")
        echo "  ✅ $(basename "$file") (${size} bytes)"
    done
    
    echo ""
    echo "📦 Binary Packages:"
    find "$CAPTURE_DIR/launchpad/binaries" -name "*.json" 2>/dev/null | sort | while read file; do
        size=$(stat -c%s "$file" 2>/dev/null || echo "0")
        echo "  ✅ $(basename "$file") (${size} bytes)"
    done
    
    echo ""
    echo "🐧 Ubuntu Series:"
    find "$CAPTURE_DIR/launchpad/series" -name "*.json" 2>/dev/null | sort | while read file; do
        size=$(stat -c%s "$file" 2>/dev/null || echo "0")
        echo "  ✅ $(basename "$file") (${size} bytes)"
    done
    
    echo ""
    echo "🎮 Other APIs:"
    find "$CAPTURE_DIR/nvidia" "$CAPTURE_DIR/kernel" -name "*" -type f 2>/dev/null | sort | while read file; do
        size=$(stat -c%s "$file" 2>/dev/null || echo "0")
        echo "  ✅ $(basename "$file") (${size} bytes)"
    done
    
    echo ""
    echo "✅ SUCCESS: Real API data captured!"
    echo ""
    echo "🔧 Next steps:"
    echo "   1. Review captured data: ls -la $CAPTURE_DIR/*/  "
    echo "   2. Run setup script: ./setup-real-mock-data.sh"
    echo "   3. Test the system: make test-coverage"
    
else
    echo "❌ ERROR: No files were captured"
    echo ""
    echo "🔍 Possible issues:"
    echo "   • Network connectivity problems"
    echo "   • API endpoints have changed"
    echo "   • Rate limiting or authentication required"
    echo ""
    echo "💡 Try manually testing an API call:"
    echo "   curl 'https://api.launchpad.net/devel/ubuntu/+archive/primary?ws.op=getPublishedSources&source_name=nvidia-graphics-drivers-570'"
fi

echo ""
echo "🎯 Real API data capture completed!"
