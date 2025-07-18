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

# Stop and disable the service
if systemctl is-active --quiet "$SERVICE_NAME"; then
    print_status "Stopping service..."
    systemctl stop "$SERVICE_NAME"
fi

if systemctl is-enabled --quiet "$SERVICE_NAME"; then
    print_status "Disabling service..."
    systemctl disable "$SERVICE_NAME"
fi

# Remove systemd service file
if [ -f "$SERVICE_FILE" ]; then
    print_status "Removing systemd service file..."
    rm -f "$SERVICE_FILE"
    systemctl daemon-reload
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
