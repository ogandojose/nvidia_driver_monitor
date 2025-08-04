# NVIDIA Driver Package Manager

A Go application for monitoring NVIDIA driver package status in Ubuntu repositories with SRU cycle awareness.

## Features

- **Console Application**: CLI tool for viewing driver status in terminal
- **Web Server**: Modern web interface with real-time data
- **SRU Cycle Awareness**: Shows next Ubuntu kernel cycle dates for outdated drivers
- **Color-coded Status**: Green for up-to-date, red for outdated drivers
- **JSON API**: Programmatic access to driver status data
- **Centralized Configuration**: All URLs and API endpoints managed via configuration files
- **Real Data Integration**: Uses only real API responses captured from live endpoints

## Quick Start

```bash
# Build all applications
make all

# Run console application
./nvidia-driver-status

# Run web server
./nvidia-web-server

# For development/testing with mock server
./nvidia-driver-status -config=config/config-real-mock.json
./nvidia-web-server -config=config/config-real-mock.json
```

## Directory Structure

```
nvidia_driver_monitor/
├── config/                    # Configuration files
├── data/                      # Data files and statistics
├── scripts/                   # Organized scripts by category
│   ├── real-data/            # Real API data scripts
│   ├── testing/              # Testing scripts
│   └── service/              # Service management
├── docs/                     # Project documentation
├── test/                     # Test files and coverage
├── test-data/                # Mock server data (real API responses)
└── captured_real_api_responses/ # Raw captured API data
```

## Configuration

The application uses a centralized configuration system to manage all external URLs, API endpoints, and application settings.

### Configuration File

By default, the application looks for `config/config.json`. If not found, it uses built-in defaults.

#### Generating Configuration

Use the config management tool to generate a default configuration file:

```bash
# Build the config tool
make config

# Generate default configuration to config/ directory
./config-tool -generate -config config/config.json

# Show current configuration
./config-tool -show -config config/config.json

# Validate configuration file
./config-tool -validate -config config/config.json
```

#### Configuration Structure

The configuration file contains:

- **Server Settings**: Ports, HTTPS configuration
- **Cache Settings**: Data refresh intervals
- **Rate Limiting**: Request throttling settings
- **External URLs**: All API endpoints and CDN resources
  - Ubuntu API endpoints
  - Launchpad API endpoints  
  - NVIDIA driver URLs
  - CDN resources (Bootstrap, Chart.js, etc.)
  - Kernel information URLs
- **HTTP Client**: Timeout and retry settings

#### Example Configuration

```json
{
  "server": {
    "port": 8080,
    "https_port": 8443,
    "enable_https": false
  },
  "urls": {
    "ubuntu": {
      "assets_base_url": "https://assets.ubuntu.com/v1"
    },
    "launchpad": {
      "base_url": "https://api.launchpad.net/devel",
      "published_sources_api": "https://api.launchpad.net/devel/ubuntu/+archive/primary"
    },
    "nvidia": {
      "driver_archive_url": "https://www.nvidia.com/en-us/drivers/unix/linux-amd64-display-archive/"
    },
    "cdn": {
      "bootstrap_css": "https://cdn.jsdelivr.net/npm/bootstrap@5.1.3/dist/css/bootstrap.min.css"
    }
  }
}
```

#### Using Custom Configuration

```bash
# Console application
./nvidia-driver-status -config /path/to/config.json

# Web server
./nvidia-web-server -config /path/to/config.json
```

## Building

Use the provided Makefile to build both applications:

```bash
# Build both console and web applications
make

# Build only console application
make console

# Build only web server application
make web

# Install/update dependencies
make deps
```

## Running

### Development Mode

```bash
# Run console application
make run-console
# OR
./nvidia-driver-status

# Run web server application
make run-web
# OR
./nvidia-web-server

# Run web server on custom port
./nvidia-web-server -addr :9090
```

### Production Mode (Systemd Service)

```bash
# Install as systemd service
make install-service

# Start the service
make service-start

# Check service status
make service-status

# View service logs
make service-logs

# Stop the service
make service-stop
```

For detailed service management, see [SERVICE.md](SERVICE.md).

## Web Interface

Once the web server is running, access:
- **Main Dashboard**: http://localhost:8080/
- **Individual Package**: http://localhost:8080/package?package=nvidia-graphics-drivers-550
- **JSON API**: http://localhost:8080/api

## Development

```bash
# Format code
make fmt

# Run tests
make test

# Kill processes on port 8080
make kill-web

# Clean build artifacts (keeps mod cache)
make clean-dev

# Full clean (removes everything)
make clean

# Show project status
make status

# Show all available targets
make help
```

## Project Structure

This project has been refactored into a more maintainable structure following Go best practices:

```
nvidia_driver_monitor/
├── main.go                          # Main application entry point
├── go.mod                           # Go module definition
├── go.sum                           # Go module dependencies
├── supportedReleases.json           # Configuration file for supported releases
├── internal/                        # Internal packages (not importable by external projects)
│   ├── packages/                    # Package-related functionality
│   │   ├── source.go               # Source package operations
│   │   └── binary.go               # Binary package operations
│   ├── drivers/                     # Driver-related functionality
│   │   ├── uda.go                  # UDA (Unified Driver Architecture) driver handling
│   │   └── server.go               # Server driver handling
│   ├── releases/                    # Release management
│   │   └── supported.go            # Supported releases configuration
│   └── utils/                       # Common utilities
│       └── common.go               # Shared utility functions
└── old_files/                       # Backup of original files
```

## Package Organization

### `/internal/packages/`
- **source.go**: Handles source package queries and version management from Launchpad
- **binary.go**: Handles binary package queries and version management from Launchpad

### `/internal/drivers/`
- **uda.go**: Fetches and processes UDA driver information from NVIDIA's website
- **server.go**: Fetches and processes server driver information from NVIDIA's datacenter documentation

### `/internal/releases/`
- **supported.go**: Manages supported release configurations, updates, and persistence

### `/internal/utils/`
- **common.go**: Contains shared utility functions used across packages

## Key Improvements

1. **Modular Design**: Code is organized into logical packages based on functionality
2. **Separation of Concerns**: Each package has a specific responsibility
3. **Improved Maintainability**: Changes to one area don't affect others
4. **Better Testing**: Each package can be tested independently
5. **Clear Dependencies**: Import relationships are explicit and well-defined
6. **Go Best Practices**: Follows standard Go project layout conventions

## Usage

```bash
# Build the application
go build -o nvidia_example .

# Run the application
./nvidia_example
```

## Migration Notes

- All original functionality is preserved
- Function signatures remain the same for backward compatibility
- The main application flow is unchanged
- Configuration files remain in the same location

## Dependencies

- `github.com/knqyf263/go-deb-version`: For Debian version comparison
- `golang.org/x/net/html`: For HTML parsing

## Original Files

The original files have been moved to `old_files/` directory for reference and can be removed once the refactoring is verified to work correctly.

## Web Service

A web service is also available that displays the same information as the command-line tool in a user-friendly web interface.

### Quick Start

```bash
# Build and start the web server
./start-web-server.sh

# Or manually:
go build -o web-server ./cmd/web/
./web-server -addr :8080
```

Then open your browser to `http://localhost:8080`

### Features

- **Interactive Web Interface**: Clean, responsive HTML tables showing package status
- **Color Coding**: Green/red indicators for version matching
- **JSON API**: REST endpoints for programmatic access
- **Real-time Data**: Live data from Launchpad and NVIDIA sources

### API Endpoints

- `GET /` - Web interface showing all packages
- `GET /package?package=<name>` - Web interface for specific package  
- `GET /api` - JSON data for all packages
- `GET /api?package=<name>` - JSON data for specific package

See [WEB_SERVICE.md](WEB_SERVICE.md) for detailed documentation.

## Usage
