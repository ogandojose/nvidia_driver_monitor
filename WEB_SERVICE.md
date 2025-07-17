# NVIDIA Driver Package Web Service

This web service provides a web interface to display NVIDIA driver package information, showing the same data as the `PrintSourceVersionMapTableWithSupported` function but in a user-friendly web format.

## Features

- **Web Interface**: Clean, responsive HTML interface displaying package information in tables
- **Color Coding**: 
  - Green background indicates package version contains upstream version
  - Red background indicates package version does not contain upstream version
- **JSON API**: REST API endpoints for programmatic access
- **Real-time Data**: Fetches live data from Launchpad API and NVIDIA sources

## Building and Running

### Build the Web Server

```bash
go build -o web-server ./cmd/web/
```

### Run the Web Server

```bash
./web-server -addr :8080
```

The server will start and be available at `http://localhost:8080`

## API Endpoints

### Web Interface

- **`/`** - Main page showing all NVIDIA driver packages
- **`/package?package=<package-name>`** - Details for a specific package

### JSON API

- **`/api`** - Returns all packages data as JSON
- **`/api?package=<package-name>`** - Returns specific package data as JSON

## Examples

### Get All Packages (JSON)
```bash
curl http://localhost:8080/api
```

### Get Specific Package (JSON)
```bash
curl "http://localhost:8080/api?package=nvidia-graphics-drivers-575"
```

### View Specific Package (Web)
```
http://localhost:8080/package?package=nvidia-graphics-drivers-575
```

## Data Structure

The web service processes the following information for each package:

- **Package Name**: The full Ubuntu package name
- **Series**: Ubuntu release series (questing, plucky, noble, jammy, focal, bionic)
- **Updates/Security**: Version available in updates/security pocket
- **Proposed**: Version available in proposed pocket
- **Upstream Version**: Latest version from NVIDIA upstream
- **Color Status**: Visual indicator of version matching

## Command Line Options

- **`-addr`**: HTTP server address (default: `:8080`)

## Dependencies

The web service uses the same internal packages as the command-line tool:
- `internal/packages`: Package version fetching from Launchpad
- `internal/drivers`: Driver version fetching from NVIDIA sources  
- `internal/releases`: Supported releases management

## Browser Compatibility

The web interface is designed to work with all modern browsers and includes:
- Responsive CSS styling
- Clean table layouts
- Color-coded status indicators
- Easy navigation between packages
