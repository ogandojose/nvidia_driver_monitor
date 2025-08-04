#!/bin/bash

# Organize Real Mock Data
# Reorganizes captured real API responses into the mock server's expected structure

set -e

echo "🔄 ORGANIZING REAL MOCK DATA"
echo "=========================="
echo ""

cd "$(dirname "$0")"

TEST_DATA_DIR="test-data"

if [ ! -d "$TEST_DATA_DIR" ]; then
    echo "❌ ERROR: test-data directory not found!"
    exit 1
fi

cd "$TEST_DATA_DIR"

echo "📁 Creating directory structure..."

# Create directory structure
mkdir -p launchpad/sources
mkdir -p launchpad/binaries
mkdir -p launchpad/series
mkdir -p nvidia
mkdir -p kernel

echo "📋 Organizing files by API type..."

# Process Launchpad source files
for file in api_launchpad_net_devel_ubuntu__archive_primary__ws_op_getPublishedSources_source_name_*.json; do
    if [ -f "$file" ]; then
        # Extract source name from filename
        source_name=$(echo "$file" | sed -n 's/.*source_name_\([^_]*\(_[^_]*\)*\)_created_since_date.*/\1/p')
        
        if [ -n "$source_name" ]; then
            echo "  📦 $source_name -> launchpad/sources/$source_name.json"
            mv "$file" "launchpad/sources/$source_name.json"
        fi
    fi
done

# Process NVIDIA files
if [ -f "docs_nvidia_com_datacenter_tesla_drivers_releases_json.json" ]; then
    echo "  🎯 NVIDIA datacenter drivers -> nvidia/server-drivers.json"
    mv "docs_nvidia_com_datacenter_tesla_drivers_releases_json.json" "nvidia/server-drivers.json"
fi

if [ -f "www_nvidia_com_en_us_drivers_unix_linux_amd64_display_archive_.json" ]; then
    echo "  🎯 NVIDIA display drivers -> nvidia/display-drivers.json"
    mv "www_nvidia_com_en_us_drivers_unix_linux_amd64_display_archive_.json" "nvidia/display-drivers.json"
fi

# Process Kernel files
if [ -f "kernel_ubuntu_com_forgejo_kernel_kernel_versions_raw_branch_main_info_sru_cycle_yaml.yaml" ]; then
    echo "  🐧 Kernel SRU cycles -> kernel/sru-cycle.yaml"
    mv "kernel_ubuntu_com_forgejo_kernel_kernel_versions_raw_branch_main_info_sru_cycle_yaml.yaml" "kernel/sru-cycle.yaml"
fi

echo ""
echo "📊 ORGANIZATION COMPLETE"
echo "======================="
echo ""

# Count organized files
launchpad_sources=$(find launchpad/sources -name "*.json" 2>/dev/null | wc -l)
launchpad_binaries=$(find launchpad/binaries -name "*.json" 2>/dev/null | wc -l)
launchpad_series=$(find launchpad/series -name "*.json" 2>/dev/null | wc -l)
nvidia_files=$(find nvidia -name "*.json" 2>/dev/null | wc -l)
kernel_files=$(find kernel -name "*.yaml" 2>/dev/null | wc -l)

echo "📁 Organized Structure:"
echo "  📦 Launchpad sources: $launchpad_sources files"
echo "  📦 Launchpad binaries: $launchpad_binaries files"
echo "  📦 Launchpad series: $launchpad_series files"
echo "  🎯 NVIDIA APIs: $nvidia_files files"
echo "  🐧 Kernel APIs: $kernel_files files"
echo ""

total_files=$((launchpad_sources + launchpad_binaries + launchpad_series + nvidia_files + kernel_files))
echo "📊 Total organized files: $total_files"

if [ "$total_files" -gt 0 ]; then
    echo ""
    echo "✅ Real mock data successfully organized!"
    echo "🚀 You can now test with: make mock-server"
else
    echo ""
    echo "⚠️ WARNING: No files were organized"
fi
