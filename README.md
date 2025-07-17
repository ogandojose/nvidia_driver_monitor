# NVIDIA Driver Package Manager

## Project Structure

This project has been refactored into a more maintainable structure following Go best practices:

```
nvidia_example_550/
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
