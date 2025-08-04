#!/bin/bash

# Setup Real Mock Data
# Replaces synthetic test data with real captured API responses

set -e

echo "🔄 SETUP REAL MOCK DATA"
echo "======================="
echo ""

cd "$(dirname "$0")"

CAPTURE_DIR="captured_real_api_responses"
MOCK_DIR="test-data"

# Check if captured data exists
if [ ! -d "$CAPTURE_DIR" ]; then
    echo "❌ ERROR: No captured data found!"
    echo ""
    echo "🔧 Please run the capture script first:"
    echo "   ./capture-real-data-simple.sh"
    exit 1
fi

# Count captured files
captured_files=$(find "$CAPTURE_DIR" -type f 2>/dev/null | wc -l)
if [ "$captured_files" -eq 0 ]; then
    echo "❌ ERROR: Capture directory is empty!"
    echo ""
    echo "🔧 Please run the capture script first:"
    echo "   ./capture-real-data-simple.sh"
    exit 1
fi

echo "📁 Found $captured_files captured files"
echo ""

# Backup existing test data
if [ -d "$MOCK_DIR" ]; then
    BACKUP_DIR="test-data-synthetic-backup-$(date +%Y%m%d-%H%M%S)"
    echo "💾 Backing up existing synthetic data to: $BACKUP_DIR"
    mv "$MOCK_DIR" "$BACKUP_DIR"
    echo "   ✅ Backup created"
fi

# Copy captured real data to test-data directory
echo "📋 Setting up real API data as mock data..."
echo ""

cp -r "$CAPTURE_DIR" "$MOCK_DIR"
echo "   ✅ Copied captured data to $MOCK_DIR"

# Verify the structure
echo ""
echo "📊 REAL MOCK DATA SETUP RESULTS"
echo "==============================="

echo ""
echo "📦 Real Data Inventory:"

# Count files by category
sources_count=$(find "$MOCK_DIR/launchpad/sources" -name "*.json" 2>/dev/null | wc -l)
binaries_count=$(find "$MOCK_DIR/launchpad/binaries" -name "*.json" 2>/dev/null | wc -l)
series_count=$(find "$MOCK_DIR/launchpad/series" -name "*.json" 2>/dev/null | wc -l)
nvidia_count=$(find "$MOCK_DIR/nvidia" -type f 2>/dev/null | wc -l)
kernel_count=$(find "$MOCK_DIR/kernel" -type f 2>/dev/null | wc -l)

echo "  📁 Source packages: $sources_count files"
echo "  📁 Binary packages: $binaries_count files"
echo "  📁 Ubuntu series: $series_count files"
echo "  📁 NVIDIA APIs: $nvidia_count files"
echo "  📁 Kernel APIs: $kernel_count files"

total_files=$(( sources_count + binaries_count + series_count + nvidia_count + kernel_count ))
echo ""
echo "📊 Total real API responses: $total_files files"

# Validate some key files exist and have content
echo ""
echo "🔍 Validating key data files..."

validate_file() {
    local file="$1"
    local description="$2"
    
    if [ -f "$MOCK_DIR/$file" ]; then
        size=$(stat -c%s "$MOCK_DIR/$file" 2>/dev/null || echo "0")
        if [ "$size" -gt 10 ]; then
            echo "  ✅ $description (${size} bytes)"
            return 0
        else
            echo "  ⚠️ $description (${size} bytes - may be empty)"
            return 1
        fi
    else
        echo "  ❌ $description (missing)"
        return 1
    fi
}

# Validate critical files
validation_ok=true

validate_file "launchpad/sources/nvidia-graphics-drivers-570.json" "NVIDIA 570 sources" || validation_ok=false
validate_file "launchpad/sources/nvidia-graphics-drivers-535.json" "NVIDIA 535 sources" || validation_ok=false
validate_file "launchpad/binaries/nvidia-driver-570.json" "NVIDIA 570 binaries" || validation_ok=false
validate_file "launchpad/series/noble.json" "Ubuntu Noble series" || validation_ok=false
validate_file "launchpad/series/jammy.json" "Ubuntu Jammy series" || validation_ok=false

# Check if we have some NVIDIA/kernel data (these might fail due to API availability)
validate_file "kernel/series.yaml" "Kernel series YAML" || echo "  ℹ️ Kernel API may not be available"
validate_file "nvidia/server-drivers.json" "NVIDIA server drivers" || echo "  ℹ️ NVIDIA API may not be available"

echo ""

if [ "$validation_ok" = true ]; then
    echo "✅ SUCCESS: Real API data setup completed!"
    echo ""
    echo "🎯 The mock server now uses REAL API data instead of synthetic data"
    echo ""
    echo "📋 What changed:"
    echo "  • All Launchpad API responses are now real"
    echo "  • Version numbers are actual Ubuntu package versions"
    echo "  • Timestamps reflect real publication dates"
    echo "  • Package relationships are authentic"
    echo ""
    echo "🚀 Test the real data system:"
    echo "  make run-mock              # Start mock server with real data"
    echo "  make test-coverage         # Verify everything works"
    echo "  ./verify-comprehensive-coverage.sh  # Run full verification"
    echo ""
    echo "🔄 To revert to synthetic data:"
    echo "  mv $MOCK_DIR $MOCK_DIR-real-backup"
    echo "  mv test-data-synthetic-backup-* $MOCK_DIR"
    
else
    echo "⚠️ WARNING: Some validation checks failed"
    echo ""
    echo "🔧 The system should still work, but some endpoints may return empty responses"
    echo "   This often happens when certain APIs are unavailable or rate-limited"
    echo ""
    echo "🚀 You can still test the system:"
    echo "  make test-coverage"
fi

echo ""
echo "📋 File Details:"
find "$MOCK_DIR" -type f | sort | while read file; do
    rel_file=${file#$MOCK_DIR/}
    size=$(stat -c%s "$file" 2>/dev/null || echo "0")
    if [ "$size" -gt 0 ]; then
        echo "  📄 $rel_file (${size} bytes)"
    else
        echo "  📄 $rel_file (empty)"
    fi
done

echo ""
echo "🎯 Real mock data setup completed!"
