#!/bin/bash

# NVIDIA Driver Monitor Service Uninstallation Script

set -e

# Configuration
SERVICE_NAME="nvidia-driver-monitor"
SERVICE_USER="nvidia-monitor"
SERVICE_GROUP="nvidia-monitor"
INSTALL_DIR="/opt/nvidia-driver-monitor"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
LOG_FILE="/var/log/${SERVICE_NAME}.log"

# Additional service files that might exist from older installations
ADDITIONAL_SERVICE_FILES=(
    "/etc/systemd/system/${SERVICE_NAME}-minimal.service"
    "/etc/systemd/system/${SERVICE_NAME}-https.service"
    "/etc/systemd/system/${SERVICE_NAME}-standard.service"
)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    print_error "This script must be run as root (use sudo)"
    exit 1
fi

print_status "Starting NVIDIA Driver Monitor service uninstallation..."

# Stop and disable any running services
services_to_check=(
    "$SERVICE_NAME"
    "${SERVICE_NAME}-minimal"
    "${SERVICE_NAME}-https"
    "${SERVICE_NAME}-standard"
)

for service in "${services_to_check[@]}"; do
    if systemctl is-active --quiet "$service" 2>/dev/null; then
        print_status "Stopping service: $service"
        systemctl stop "$service"
    fi
    
    if systemctl is-enabled --quiet "$service" 2>/dev/null; then
        print_status "Disabling service: $service"
        systemctl disable "$service"
    fi
done

# Remove systemd service files
print_status "Removing systemd service files..."
service_files_removed=0

# Remove main service file
if [ -f "$SERVICE_FILE" ]; then
    print_status "Removing main service file: $SERVICE_FILE"
    rm -f "$SERVICE_FILE"
    service_files_removed=$((service_files_removed + 1))
fi

# Remove any additional service files from older installations
for service_file in "${ADDITIONAL_SERVICE_FILES[@]}"; do
    if [ -f "$service_file" ]; then
        print_status "Removing additional service file: $service_file"
        rm -f "$service_file"
        service_files_removed=$((service_files_removed + 1))
    fi
done

# Reload systemd daemon if any service files were removed
if [ $service_files_removed -gt 0 ]; then
    print_status "Reloading systemd daemon..."
    systemctl daemon-reload
    print_status "Removed $service_files_removed service file(s)"
else
    print_warning "No service files found to remove"
fi

# Remove installation directory
if [ -d "$INSTALL_DIR" ]; then
    print_status "Removing installation directory..."
    rm -rf "$INSTALL_DIR"
fi

# Remove log file
if [ -f "$LOG_FILE" ]; then
    print_status "Removing log file..."
    rm -f "$LOG_FILE"
fi

# Optionally remove user and group
read -p "Remove service user and group? [y/N]: " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    if getent passwd "$SERVICE_USER" >/dev/null 2>&1; then
        print_status "Removing service user..."
        userdel "$SERVICE_USER"
    fi
    
    if getent group "$SERVICE_GROUP" >/dev/null 2>&1; then
        print_status "Removing service group..."
        groupdel "$SERVICE_GROUP"
    fi
fi

print_status "Uninstallation completed successfully!"
