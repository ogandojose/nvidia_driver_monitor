# NVIDIA Driver Package Status Tool
# Makefile for building console and web server applications

# Build configuration
CONSOLE_BINARY = nvidia-driver-status
WEB_BINARY = nvidia-web-server
CONFIG_BINARY = nvidia-config
MOCK_BINARY = nvidia-mock-server
CONSOLE_SOURCE = main.go
WEB_SOURCE = cmd/web/main.go
CONFIG_SOURCE = cmd/config/main.go
MOCK_SOURCE = cmd/mock-server/main.go

# Go build flags
GO_BUILD_FLAGS = -ldflags="-s -w"

# Default target
.PHONY: all
all: console web config mock

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

# Build configuration tool
.PHONY: config
config:
	@echo "Building configuration tool..."
	go build $(GO_BUILD_FLAGS) -o $(CONFIG_BINARY) $(CONFIG_SOURCE)
	@echo "Configuration tool built: $(CONFIG_BINARY)"

# Build mock server
.PHONY: mock
mock:
	@echo "Building mock server..."
	go build $(GO_BUILD_FLAGS) -o $(MOCK_BINARY) $(MOCK_SOURCE)
	@echo "Mock server built: $(MOCK_BINARY)"

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

# Run web server with HTTPS
.PHONY: run-web-https
run-web-https:
	@echo "Running web server application with HTTPS..."
	go run $(WEB_SOURCE) -https

# Run LRM verifier (web server focused on LRM functionality)
.PHONY: run-lrm
run-lrm:
	@echo "Running web server for LRM verifier testing..."
	@echo "LRM Verifier will be available at: http://localhost:8080/l-r-m-verifier"
	@echo "API endpoint available at: http://localhost:8080/api/lrm"
	go run $(WEB_SOURCE)

# Run mock server
.PHONY: run-mock
run-mock:
	@echo "Running mock server..."
	@echo "Mock server will be available at: http://localhost:9999"
	go run $(MOCK_SOURCE)

# Run mock server with config
.PHONY: run-mock-config
run-mock-config:
	@echo "Running mock server with configuration..."
	go run $(MOCK_SOURCE) -config config.json

# Run web server with testing mode (requires mock server to be running)
.PHONY: run-web-testing
run-web-testing:
	@echo "Running web server in testing mode..."
	@echo "Make sure to start mock server first: make run-mock"
	@echo "Web server will use local mock APIs instead of external services"
	./$(CONFIG_BINARY) -generate -config config-testing.json > /dev/null
	@echo '{ "testing": { "enabled": true } }' > config-testing-patch.json
	@# Update config to enable testing mode (simplified approach)
	go run $(WEB_SOURCE) -config config-testing.json

# Generate self-signed certificate
.PHONY: generate-cert
generate-cert:
	@echo "Generating self-signed certificate..."
	@if [ -f "server.crt" ] || [ -f "server.key" ]; then \
		echo "Certificate files already exist. Use 'make clean-cert' to remove them first."; \
	else \
		echo "Certificates will be generated automatically when running HTTPS mode."; \
		echo "Use 'make run-web-https' or './$(WEB_BINARY) -https' to start HTTPS server."; \
	fi

# Clean certificate files
.PHONY: clean-cert
clean-cert:
	@echo "Removing certificate files..."
	rm -f server.crt server.key
	@echo "Certificate files removed."

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
	-rm -f nvidia-driver-monitor
	-rm -f nvidia_driver_monitor
	-rm -f nvidia-monitor
	# Remove test and debug files
	-rm -f test-*.go
	-rm -f test_*.go
	-rm -f *_test_*.go
	-rm -f debug_*.html
	-rm -f test_*.html
	# Remove temporary files
	-rm -f *.tmp
	-rm -f *.temp
	-rm -f *.log
	-rm -f *~
	-rm -f *.bak
	-rm -f *.backup
	-rm -f .*.swp
	-rm -f .*.swo
	# Remove distribution directory
	-rm -rf dist/
	# Note: Certificates are preserved (use 'make clean-cert' to remove them)
	# Remove Go build cache (optional)
	-go clean -cache
	-go clean -modcache
	@echo "Clean completed."

# Development clean (keeps mod cache and certificates)
.PHONY: clean-dev
clean-dev:
	@echo "Cleaning build artifacts (keeping mod cache and certificates)..."
	# Remove built binaries
	-rm -f $(CONSOLE_BINARY)
	-rm -f $(WEB_BINARY)
	-rm -f nvidia-driver-monitor
	-rm -f nvidia_driver_monitor
	-rm -f nvidia-monitor
	# Remove test and debug files
	-rm -f test-*.go
	-rm -f test_*.go
	-rm -f *_test_*.go
	-rm -f debug_*.html
	-rm -f test_*.html
	# Remove temporary files
	-rm -f *.tmp
	-rm -f *.temp
	-rm -f *.log
	-rm -f *~
	-rm -f *.bak
	-rm -f *.backup
	-rm -f .*.swp
	-rm -f .*.swo
	# Remove distribution directory
	-rm -rf dist/
	@echo "Development clean completed."

# Full clean including certificates
.PHONY: clean-all
clean-all: clean clean-cert
	@echo "Full clean completed (including certificates)."

# Test the applications
.PHONY: test
test:
	@echo "Running tests..."
	go test ./...

# Run integration test for mock testing service
.PHONY: test-integration
test-integration:
	@echo "Running integration test for mock testing service..."
	./test-integration.sh

# Testing targets
.PHONY: test-coverage test-ultimate verify-coverage

test-coverage: $(MOCK_BINARY)
	@echo "🔍 Running comprehensive coverage verification..."
	@./verify-comprehensive-coverage.sh

test-ultimate: $(MOCK_BINARY)
	@echo "🎯 Running ultimate coverage test (all 2,367 combinations)..."
	@./ultimate-coverage-test.sh

verify-coverage: test-coverage
	@echo "✅ Coverage verification completed"

# Validate templates
.PHONY: validate-templates
validate-templates:
	@echo "Validating HTML templates..."
	@for template in templates/*.html; do \
		if [ -f "$$template" ]; then \
			echo "Checking $$template..."; \
			if ! grep -q "<!DOCTYPE html>" "$$template"; then \
				echo "Warning: $$template missing DOCTYPE declaration"; \
			fi; \
			if ! grep -q "</html>" "$$template"; then \
				echo "Warning: $$template missing closing </html> tag"; \
			fi; \
		fi; \
	done
	@echo "Template validation completed."

# Run template validation and syntax check
.PHONY: check-templates
check-templates: validate-templates
	@echo "Running template syntax check..."
	@echo "Building web server to validate templates..."
	@if go build -o /tmp/template-check $(WEB_SOURCE) 2>/dev/null; then \
		echo "✅ Templates compiled successfully"; \
		rm -f /tmp/template-check; \
	else \
		echo "❌ Template compilation failed"; \
		exit 1; \
	fi

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
	@echo "  run-web-https    - Run web server application with HTTPS"
	@echo "  run-lrm          - Run web server for LRM verifier testing"
	@echo "  run-mock         - Run mock server"
	@echo "  run-mock-config  - Run mock server with configuration"
	@echo "  run-web-testing   - Run web server in testing mode"
	@echo "  generate-cert    - Generate self-signed certificate"
	@echo "  clean-cert       - Clean certificate files"
	@echo "  kill-web         - Kill processes running on port 8080"
	@echo "  test             - Run tests"
	@echo "  test-integration  - Run integration test for mock testing service"
	@echo "  validate-templates - Validate HTML template structure"
	@echo "  check-templates  - Validate templates and test loading"
	@echo "  fmt              - Format code"
	@echo "  lint             - Lint code"
	@echo "  status           - Show project status"
	@echo ""
	@echo "Service management targets:"
	@echo "  install-service  - Install systemd service (requires sudo)"
	@echo "  check-install-requirements - Verify all installation files are present"
	@echo "  uninstall-service- Uninstall systemd service (requires sudo)"
	@echo "  service-start    - Start the service"
	@echo "  service-stop     - Stop the service"
	@echo "  service-restart  - Restart the service"
	@echo "  service-status   - Show service status"
	@echo "  service-logs     - Show service logs"
	@echo "  troubleshoot-network - Run network troubleshooting"
	@echo "  fix-network      - Fix network connectivity issues"
	@echo ""
	@echo "Distribution targets:"
	@echo "  dist             - Create distribution package with all files"
	@echo ""
	@echo "Cleanup targets:"
	@echo "  clean            - Remove all build artifacts and temporary files"
	@echo "  clean-dev        - Remove build artifacts (keep mod cache)"
	@echo ""
	@echo "  help             - Show this help message"

# Service management targets
.PHONY: install-service
install-service: web check-install-requirements
	@echo "Installing systemd service..."
	sudo ./scripts/service/install-service.sh

# Check installation requirements
.PHONY: check-install-requirements
check-install-requirements:
	@echo "Checking installation requirements..."
	@if [ ! -f "$(WEB_BINARY)" ]; then \
		echo "❌ Web binary $(WEB_BINARY) not found. Run 'make web' first."; \
		exit 1; \
	fi
	@if [ ! -d "templates" ]; then \
		echo "❌ Templates directory not found."; \
		exit 1; \
	fi
	@if [ ! -f "templates/lrm_verifier.html" ]; then \
		echo "❌ Required template templates/lrm_verifier.html not found."; \
		exit 1; \
	fi
	@if [ ! -f "templates/statistics.html" ]; then \
		echo "❌ Required template templates/statistics.html not found."; \
		exit 1; \
	fi
	@if [ ! -f "templates/index.html" ]; then \
		echo "❌ Required template templates/index.html not found."; \
		exit 1; \
	fi
	@if [ ! -d "static" ]; then \
		echo "❌ Static assets directory not found."; \
		exit 1; \
	fi
	@if [ ! -f "static/css/statistics.css" ]; then \
		echo "❌ Required CSS file static/css/statistics.css not found."; \
		exit 1; \
	fi
	@if [ ! -f "static/css/ubuntu-theme.css" ]; then \
		echo "❌ Required CSS file static/css/ubuntu-theme.css not found."; \
		exit 1; \
	fi
	@if [ ! -f "static/js/statistics.js" ]; then \
		echo "❌ Required JS file static/js/statistics.js not found."; \
		exit 1; \
	fi
	@if [ ! -f "data/supportedReleases.json" ]; then \
		echo "❌ data/supportedReleases.json not found."; \
		exit 1; \
	fi
	@if [ ! -f "config.json" ]; then \
		echo "❌ config.json not found."; \
		exit 1; \
	fi
	@echo "✅ All installation requirements met."
	@echo "Note: Statistics will be persisted to statistics_data.json (created automatically)"

# Create distribution package
.PHONY: dist
dist: web check-install-requirements
	@echo "Creating distribution package..."
	@mkdir -p dist/nvidia-driver-monitor
	@cp $(WEB_BINARY) dist/nvidia-driver-monitor/
	@cp data/supportedReleases.json dist/nvidia-driver-monitor/
	@cp config.json dist/nvidia-driver-monitor/
	@cp -r templates dist/nvidia-driver-monitor/
	@cp -r static dist/nvidia-driver-monitor/
	@cp *.service dist/nvidia-driver-monitor/
	@cp scripts/service/install-service.sh dist/nvidia-driver-monitor/
	@cp scripts/service/uninstall-service.sh dist/nvidia-driver-monitor/
	@cp README.md dist/nvidia-driver-monitor/
	@echo "✅ Distribution package created in dist/nvidia-driver-monitor/"
	@echo "Package contents:"
	@find dist/nvidia-driver-monitor -type f | sort

.PHONY: uninstall-service
uninstall-service:
	@echo "Uninstalling systemd service..."
	sudo ./scripts/service/uninstall-service.sh

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

.PHONY: troubleshoot-network
troubleshoot-network:
	@echo "Running network troubleshooting..."
	sudo ./troubleshoot-network.sh

.PHONY: fix-network
fix-network:
	@echo "Applying network connectivity fix..."
	sudo ./fix-network.sh

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
