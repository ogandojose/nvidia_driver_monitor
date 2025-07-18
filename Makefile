# NVIDIA Driver Package Status Tool
# Makefile for building console and web server applications

# Build configuration
CONSOLE_BINARY = nvidia-driver-status
WEB_BINARY = nvidia-web-server
CONSOLE_SOURCE = main.go
WEB_SOURCE = cmd/web/main.go

# Go build flags
GO_BUILD_FLAGS = -ldflags="-s -w"

# Default target
.PHONY: all
all: console web

# Build console application
.PHONY: console
console:
	@echo "Building console application..."
	go build $(GO_BUILD_FLAGS) -o $(CONSOLE_BINARY) $(CONSOLE_SOURCE)
	@echo "Console application built: $(CONSOLE_BINARY)"

# Build web server application
.PHONY: web
web:
	@echo "Building web server application..."
	go build $(GO_BUILD_FLAGS) -o $(WEB_BINARY) $(WEB_SOURCE)
	@echo "Web server application built: $(WEB_BINARY)"

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download

# Run console application
.PHONY: run-console
run-console:
	@echo "Running console application..."
	go run $(CONSOLE_SOURCE)

# Run web server application
.PHONY: run-web
run-web:
	@echo "Running web server application..."
	go run $(WEB_SOURCE)

# Kill processes running on port 8080
.PHONY: kill-web
kill-web:
	@echo "Killing processes on port 8080..."
	-pkill -f "$(WEB_BINARY)"
	-pkill -f "go run $(WEB_SOURCE)"
	-lsof -ti:8080 | xargs -r kill -9
	@echo "Processes killed."

# Clean build artifacts and temporary files
.PHONY: clean
clean:
	@echo "Cleaning build artifacts and temporary files..."
	# Remove built binaries
	-rm -f $(CONSOLE_BINARY)
	-rm -f $(WEB_BINARY)
	# Remove test files
	-rm -f test-*.go
	# Remove temporary files
	-rm -f *.tmp
	-rm -f *.log
	# Remove Go build cache (optional)
	-go clean -cache
	-go clean -modcache
	@echo "Clean completed."

# Development clean (keeps mod cache)
.PHONY: clean-dev
clean-dev:
	@echo "Cleaning build artifacts (keeping mod cache)..."
	# Remove built binaries
	-rm -f $(CONSOLE_BINARY)
	-rm -f $(WEB_BINARY)
	# Remove test files
	-rm -f test-*.go
	# Remove temporary files
	-rm -f *.tmp
	-rm -f *.log
	@echo "Development clean completed."

# Test the applications
.PHONY: test
test:
	@echo "Running tests..."
	go test ./...

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
.PHONY: lint
lint:
	@echo "Linting code..."
	golint ./...

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build targets:"
	@echo "  all              - Build both console and web applications (default)"
	@echo "  console          - Build console application"
	@echo "  web              - Build web server application"
	@echo "  deps             - Install/update dependencies"
	@echo ""
	@echo "Development targets:"
	@echo "  run-console      - Run console application"
	@echo "  run-web          - Run web server application"
	@echo "  kill-web         - Kill processes running on port 8080"
	@echo "  test             - Run tests"
	@echo "  fmt              - Format code"
	@echo "  lint             - Lint code"
	@echo "  status           - Show project status"
	@echo ""
	@echo "Service management targets:"
	@echo "  install-service  - Install systemd service (requires sudo)"
	@echo "  uninstall-service- Uninstall systemd service (requires sudo)"
	@echo "  service-start    - Start the service"
	@echo "  service-stop     - Stop the service"
	@echo "  service-restart  - Restart the service"
	@echo "  service-status   - Show service status"
	@echo "  service-logs     - Show service logs"
	@echo ""
	@echo "Cleanup targets:"
	@echo "  clean            - Remove all build artifacts and temporary files"
	@echo "  clean-dev        - Remove build artifacts (keep mod cache)"
	@echo ""
	@echo "  help             - Show this help message"

# Service management targets
.PHONY: install-service
install-service: web
	@echo "Installing systemd service..."
	sudo ./install-service.sh

.PHONY: uninstall-service
uninstall-service:
	@echo "Uninstalling systemd service..."
	sudo ./uninstall-service.sh

.PHONY: service-start
service-start:
	@echo "Starting service..."
	sudo ./service-manager.sh start

.PHONY: service-stop
service-stop:
	@echo "Stopping service..."
	sudo ./service-manager.sh stop

.PHONY: service-restart
service-restart:
	@echo "Restarting service..."
	sudo ./service-manager.sh restart

.PHONY: service-status
service-status:
	@echo "Checking service status..."
	sudo ./service-manager.sh status

.PHONY: service-logs
service-logs:
	@echo "Showing service logs..."
	sudo ./service-manager.sh logs

# Show current status
.PHONY: status
status:
	@echo "Project Status:"
	@echo "==============="
	@echo "Console binary: $(CONSOLE_BINARY)"
	@if [ -f "$(CONSOLE_BINARY)" ]; then echo "  Status: Built ($(shell ls -lh $(CONSOLE_BINARY) | awk '{print $$5}'))"; else echo "  Status: Not built"; fi
	@echo "Web binary: $(WEB_BINARY)"
	@if [ -f "$(WEB_BINARY)" ]; then echo "  Status: Built ($(shell ls -lh $(WEB_BINARY) | awk '{print $$5}'))"; else echo "  Status: Not built"; fi
	@echo "Test files: $(shell find . -name 'test-*.go' 2>/dev/null | wc -l) found"
	@echo "Port 8080: $(shell lsof -ti:8080 2>/dev/null | wc -l) process(es) running"
	@echo "Service status:"
	@if systemctl is-active --quiet nvidia-driver-monitor 2>/dev/null; then echo "  Service: Running"; else echo "  Service: Not running"; fi
	@if systemctl is-enabled --quiet nvidia-driver-monitor 2>/dev/null; then echo "  Enabled: Yes"; else echo "  Enabled: No"; fi
