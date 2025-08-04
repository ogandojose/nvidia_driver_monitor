#!/usr/bin/env python3

# Calculate the estimated total number of unique API query combinations
# that the NVIDIA Driver Monitor can make

# Known packages from the code
NVIDIA_DRIVERS = [
    "nvidia-graphics-drivers-535",
    "nvidia-graphics-drivers-535-server",
    "nvidia-graphics-drivers-550", 
    "nvidia-graphics-drivers-550-server",
    "nvidia-graphics-drivers-570",
    "nvidia-graphics-drivers-570-server", 
    "nvidia-graphics-drivers-575",
    "nvidia-graphics-drivers-575-server",
    "nvidia-graphics-drivers-470",
    "nvidia-graphics-drivers-470-server",
    "nvidia-graphics-drivers-390",
    "nvidia-graphics-drivers-460",
    "nvidia-graphics-drivers-450",
    "nvidia-graphics-drivers-465",
    # Generic query
    "nvidia-graphics-drivers"
]

LRM_PACKAGES = [
    "linux-restricted-modules",
    "linux-restricted-modules-aws", 
    "linux-restricted-modules-azure",
    "linux-restricted-modules-gcp",
    "linux-restricted-modules-gke",
    "linux-restricted-modules-oem", 
    "linux-restricted-modules-raspi",
    "linux"
]

BINARY_PACKAGES = [
    "nvidia-driver-535",
    "nvidia-driver-550",
    "nvidia-driver-570",
    "nvidia-driver-575", 
    "nvidia-driver-470",
    "nvidia-driver-390",
    "nvidia-driver-460",
    "nvidia-driver-450", 
    "nvidia-driver-465",
    "libnvidia-gl-535",
    "libnvidia-gl-550",
    "libnvidia-gl-570",
    "nvidia-dkms-535",
    "nvidia-dkms-550",
    "nvidia-dkms-570"
]

UBUNTU_SERIES = [
    "bionic",   # 18.04 LTS
    "focal",    # 20.04 LTS  
    "jammy",    # 22.04 LTS
    "kinetic",  # 22.10
    "lunar",    # 23.04
    "mantic",   # 23.10
    "noble",    # 24.04 LTS
    "oracular", # 24.10
    "plucky",   # 25.04
    "questing"  # 25.10
]

# Query parameter combinations
OPERATIONS = ["getPublishedSources", "getPublishedBinaries"]
DATE_FILTERS = [True, False]  # with/without created_since_date
EXACT_MATCH = [True, False]   # with/without exact_match=true
ORDER_BY_DATE = [True, False] # with/without order_by_date=true

def calculate_combinations():
    total = 0
    
    print("🔢 Calculating API Query Combinations")
    print("="*50)
    
    # 1. Published Sources Queries
    sources_count = 0
    
    # Global queries (all series)
    for package in NVIDIA_DRIVERS + LRM_PACKAGES:
        for date_filter in DATE_FILTERS:
            for exact in EXACT_MATCH:
                for order in ORDER_BY_DATE:
                    sources_count += 1
    
    print(f"📦 Global source queries: {sources_count}")
    
    # Series-specific queries  
    series_sources_count = 0
    for series in UBUNTU_SERIES:
        for package in NVIDIA_DRIVERS + LRM_PACKAGES:
            for date_filter in DATE_FILTERS:
                for exact in EXACT_MATCH:
                    for order in ORDER_BY_DATE:
                        series_sources_count += 1
                        
    print(f"🐧 Series-specific source queries: {series_sources_count}")
    
    # 2. Published Binaries Queries
    binaries_count = 0
    
    # Global binary queries
    for package in BINARY_PACKAGES:
        for exact in EXACT_MATCH:
            binaries_count += 1
            
    print(f"📦 Global binary queries: {binaries_count}")
    
    # Series-specific binary queries
    series_binaries_count = 0  
    for series in UBUNTU_SERIES:
        for package in BINARY_PACKAGES:
            for exact in EXACT_MATCH:
                series_binaries_count += 1
                
    print(f"🐧 Series-specific binary queries: {series_binaries_count}")
    
    # 3. Additional endpoints
    other_endpoints = 0
    
    # Ubuntu series info
    other_endpoints += len(UBUNTU_SERIES)
    
    # NVIDIA server driver API
    other_endpoints += 1
    
    # Kernel APIs  
    other_endpoints += 2  # series.yaml, sru-cycle.yaml
    
    print(f"🔧 Other API endpoints: {other_endpoints}")
    
    # Calculate totals
    launchpad_total = sources_count + series_sources_count + binaries_count + series_binaries_count
    grand_total = launchpad_total + other_endpoints
    
    print("\n📊 SUMMARY")
    print("="*30)
    print(f"📋 Launchpad API queries: {launchpad_total:,}")
    print(f"🔧 Other API endpoints: {other_endpoints}")
    print(f"🎯 TOTAL COMBINATIONS: {grand_total:,}")
    
    # Breakdown
    print(f"\n📝 BREAKDOWN:")
    print(f"   • NVIDIA driver packages: {len(NVIDIA_DRIVERS)}")
    print(f"   • LRM packages: {len(LRM_PACKAGES)}")  
    print(f"   • Binary packages: {len(BINARY_PACKAGES)}")
    print(f"   • Ubuntu series: {len(UBUNTU_SERIES)}")
    print(f"   • Parameter combinations: {2**3} (date, exact, order)")
    
    # Verify this explains "888+"
    if grand_total >= 888:
        print(f"\n✅ This explains the '888+' reference!")
        print(f"   The application can make {grand_total:,} different API queries")
    else:
        print(f"\n❓ Total ({grand_total}) is less than 888")
        print(f"   There may be additional query patterns not accounted for")
        
    return grand_total

if __name__ == "__main__":
    total = calculate_combinations()
