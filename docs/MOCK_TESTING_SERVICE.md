# Mock Testing Service Implementation

## Overview

A local mock server has been implemented to simulate responses from external APIs (Launchpad, NVIDIA, Kernel) for faster development and testing cycles. All application modules automatically use the effective URLs (mock or production) based on the testing configuration.

## Architecture

### Mock Server (`cmd/mock-server/main.go`)
- **Port**: 9999 (configurable)
- **Data Directory**: `test-data/` (configurable)
- **Endpoints**: Mirrors all external API endpoints locally

### Configuration Integration
- **Testing Mode**: Enabled via `config.testing.enabled = true`
- **URL Substitution**: All modules use `config.GetEffectiveURLs()` for automatic routing
- **Transparent Switching**: No code changes needed to switch between mock and real APIs
- **Fallback Support**: Graceful fallback to real APIs when mock data unavailable

## Usage

### 1. Generate Testing Configuration
```bash
# Generate config with testing mode enabled
./nvidia-config -generate -testing -config config-testing.json
```

### 2. Start Mock Server
```bash
# Start with default settings
make run-mock

# Start with custom config
./nvidia-mock-server -config config-testing.json

# Start with custom port/data dir
./nvidia-mock-server -port 8888 -data-dir my-test-data
```

### 3. Run Application with Mock APIs
```bash
# Start web server using mock APIs
./nvidia-web-server -config config-testing.json

# Console application with mock APIs
./nvidia-driver-status -config config-testing.json
```

## Mock Endpoints

### Launchpad API
- **Published Sources**: `/launchpad/ubuntu/+archive/primary/?ws.op=getPublishedSources&...`
- **Published Binaries**: `/launchpad/ubuntu/+archive/primary?ws.op=getPublishedBinaries&...`
- **Ubuntu Series**: `/launchpad/ubuntu/{series}`

### NVIDIA API
- **Server Drivers**: `/nvidia/datacenter/releases.json`
- **Driver Archive**: `/nvidia/drivers` (HTML page)

### Kernel API
- **Series Info**: `/kernel/series.yaml`
- **SRU Cycles**: `/kernel/sru-cycle.yaml`

## Test Data Structure

```
test-data/
├── launchpad/
│   ├── sources/
│   │   └── nvidia-graphics-drivers-570.json
│   ├── binaries/
│   └── series/
├── nvidia/
│   └── server-drivers.json
└── kernel/
    ├── series.yaml
    └── sru-cycle.yaml
```

## Features

### Automatic Fallback
- **Missing Data**: Generates minimal valid responses when test files don't exist
- **Error Handling**: Returns appropriate HTTP status codes
- **Logging**: Detailed request/response logging for debugging

### CORS Support
- **Browser Compatibility**: Full CORS headers for frontend testing
- **Development Ready**: Supports all common HTTP methods

### Real Data Capture
To populate test data with real responses:

```bash
# Capture Launchpad response
curl "https://api.launchpad.net/devel/ubuntu/+archive/primary/?ws.op=getPublishedSources&source_name=nvidia-graphics-drivers-570&created_since_date=2025-01-10&order_by_date=true&exact_match=true" > test-data/launchpad/sources/nvidia-graphics-drivers-570.json

# Capture NVIDIA response  
curl "https://docs.nvidia.com/datacenter/tesla/drivers/releases.json" > test-data/nvidia/server-drivers.json

# Capture Kernel data
curl "https://kernel.ubuntu.com/forgejo/kernel/kernel-versions/raw/branch/main/info/kernel-series.yaml" > test-data/kernel/series.yaml
```

## Benefits

### Development Speed
- **No Network Delays**: Instant API responses
- **Offline Development**: Work without internet connectivity
- **Consistent Data**: Predictable test scenarios

### Testing Capabilities
- **Edge Cases**: Simulate error conditions, malformed responses
- **Load Testing**: Test without hammering external services
- **CI/CD**: Reliable tests in automated environments

### Debugging
- **Request Logging**: See exactly what APIs are being called
- **Response Control**: Modify responses to test specific scenarios
- **Timing Control**: No rate limiting or external dependencies

## Configuration Examples

### Development Config (config-testing.json)
```json
{
  "testing": {
    "enabled": true,
    "mock_server_port": 9999,
    "data_dir": "test-data"
  },
  "cache": {
    "refresh_interval": "10s"
  },
  "http": {
    "timeout": "5s",
    "retries": 3
  }
}
```

### Production Config (config.json)
```json
{
  "testing": {
    "enabled": false
  },
  "cache": {
    "refresh_interval": "15m"
  },
  "http": {
    "timeout": "10s",
    "retries": 5
  }
}
```

## Workflow Examples

### Daily Development
```bash
# Terminal 1: Start mock server
make run-mock

# Terminal 2: Start web server with testing
./nvidia-web-server -config config-testing.json

# Terminal 3: Make changes and test instantly
curl http://localhost:8080/api
```

### Creating Test Scenarios
```bash
# 1. Capture real data
curl "https://api.launchpad.net/..." > test-data/launchpad/sources/package-name.json

# 2. Modify for edge cases
# Edit JSON to test error conditions, missing data, etc.

# 3. Test with modified data
./nvidia-web-server -config config-testing.json
```

### CI/CD Integration
```bash
# In CI pipeline
./nvidia-config -generate -testing -config ci-config.json
./nvidia-mock-server -config ci-config.json &
./nvidia-web-server -config ci-config.json &
# Run tests...
```

## Implementation Details

### URL Substitution
When `testing.enabled = true`, the configuration system automatically substitutes URLs:

- `https://api.launchpad.net/...` → `http://localhost:9999/launchpad/...`
- `https://docs.nvidia.com/...` → `http://localhost:9999/nvidia/...`
- `https://kernel.ubuntu.com/...` → `http://localhost:9999/kernel/...`

### Mock Server Features
- **Smart Routing**: Automatically maps external API patterns to local endpoints
- **File-based Responses**: Serves static JSON/YAML files from test-data directory
- **Dynamic Fallbacks**: Generates valid empty responses when files missing
- **Request Logging**: Detailed logging for debugging API calls

This testing infrastructure provides a robust foundation for development, testing, and debugging without dependency on external services.
