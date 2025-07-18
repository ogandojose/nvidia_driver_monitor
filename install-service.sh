#!/bin/bash

# NVIDIA Driver Monitor Service Installation Script
# This script installs the NVIDIA Driver Monitor web server as a systemd service

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

# Check if the binary exists
if [ ! -f "./nvidia-web-server" ]; then
    print_error "nvidia-web-server binary not found. Please build it first using 'make web'"
    exit 1
fi

print_status "Starting NVIDIA Driver Monitor service installation..."

# Create service user and group
print_status "Creating service user and group..."
if ! getent group "$SERVICE_GROUP" >/dev/null 2>&1; then
    groupadd --system "$SERVICE_GROUP"
    print_status "Created group: $SERVICE_GROUP"
else
    print_warning "Group $SERVICE_GROUP already exists"
fi

if ! getent passwd "$SERVICE_USER" >/dev/null 2>&1; then
    useradd --system --gid "$SERVICE_GROUP" --shell /bin/false \
            --home-dir "$INSTALL_DIR" --create-home "$SERVICE_USER"
    print_status "Created user: $SERVICE_USER"
else
    print_warning "User $SERVICE_USER already exists"
fi

# Create installation directory
print_status "Creating installation directory..."
mkdir -p "$INSTALL_DIR"
chown "$SERVICE_USER:$SERVICE_GROUP" "$INSTALL_DIR"
chmod 755 "$INSTALL_DIR"

# Copy binary and configuration files
print_status "Installing application files..."
cp "./nvidia-web-server" "$INSTALL_DIR/"
cp "./supportedReleases.json" "$INSTALL_DIR/"
chown "$SERVICE_USER:$SERVICE_GROUP" "$INSTALL_DIR/nvidia-web-server"
chown "$SERVICE_USER:$SERVICE_GROUP" "$INSTALL_DIR/supportedReleases.json"
chmod 755 "$INSTALL_DIR/nvidia-web-server"
chmod 644 "$INSTALL_DIR/supportedReleases.json"

# Install systemd service file
print_status "Installing systemd service file..."
cp "./nvidia-driver-monitor.service" "$SERVICE_FILE"
chown root:root "$SERVICE_FILE"
chmod 644 "$SERVICE_FILE"

# Create log file
print_status "Creating log file..."
touch "$LOG_FILE"
chown "$SERVICE_USER:$SERVICE_GROUP" "$LOG_FILE"
chmod 644 "$LOG_FILE"

# Reload systemd daemon
print_status "Reloading systemd daemon..."
systemctl daemon-reload

# Enable the service
print_status "Enabling service..."
systemctl enable "$SERVICE_NAME"

print_status "Installation completed successfully!"
echo ""
echo "Service management commands:"
echo "  Start service:   sudo systemctl start $SERVICE_NAME"
echo "  Stop service:    sudo systemctl stop $SERVICE_NAME"
echo "  Restart service: sudo systemctl restart $SERVICE_NAME"
echo "  Check status:    sudo systemctl status $SERVICE_NAME"
echo "  View logs:       sudo journalctl -u $SERVICE_NAME -f"
echo ""
echo "Web interface will be available at: http://localhost:8080"
echo "Service user: $SERVICE_USER"
echo "Install directory: $INSTALL_DIR"
echo ""
echo "To start the service now, run:"
echo "  sudo systemctl start $SERVICE_NAME"
