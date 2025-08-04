# Scripts Directory

This directory contains utility scripts for the NVIDIA Driver Monitor project.

## Real Data Scripts (`real-data/`)

Scripts for capturing and using real API data instead of synthetic test data:

- **`run-web-with-real-data.sh`** - Complete setup to run the web server with real mock data
- **`setup-real-mock-data.sh`** - Sets up real captured API responses as mock data
- **`organize-real-mock-data.sh`** - Organizes captured API responses into proper directory structure
- **`validate-real-data-complete.sh`** - Validates that the system is using 100% real data
- **`test-real-mock-data.sh`** - Tests mock server endpoints with real data
- **`capture-real-api-data.sh`** - Advanced script for capturing API responses
- **`capture-real-data-simple.sh`** - Simple script for capturing API responses

## Testing Scripts (`testing/`)

Scripts for testing and validation:

- **`ultimate-coverage-test.sh`** - Comprehensive testing suite
- **`verify-comprehensive-coverage.sh`** - Verifies test coverage
- **`test-integration.sh`** - Integration testing

## Service Scripts (`service/`)

Scripts for system service management:

- **`install-service.sh`** - Install NVIDIA Driver Monitor as a system service
- **`uninstall-service.sh`** - Remove the system service

## Quick Start

To run the web server with real data:
```bash
cd scripts/real-data
./run-web-with-real-data.sh
```

To run comprehensive tests:
```bash
cd scripts/testing
./ultimate-coverage-test.sh
```
