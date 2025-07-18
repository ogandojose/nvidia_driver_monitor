#!/bin/bash

# Quick fix for network connectivity issues in existing service installations

SERVICE_NAME="nvidia-driver-monitor"
SERVICE_FILE="/etc/systemd/system/$SERVICE_NAME.service"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

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

print_status "Applying network connectivity fix to existing service..."

# Check if service exists
if [ ! -f "$SERVICE_FILE" ]; then
    print_error "Service file not found: $SERVICE_FILE"
    print_error "Please install the service first with: make install-service"
    exit 1
fi

# Stop the service
print_status "Stopping service..."
systemctl stop "$SERVICE_NAME"

# Backup original service file
print_status "Backing up original service file..."
cp "$SERVICE_FILE" "$SERVICE_FILE.backup"

# Apply fixes
print_status "Applying network fixes..."

# Replace network restrictions
sed -i 's/IPAddressDeny=any/# IPAddressDeny=any (disabled for internet access)/' "$SERVICE_FILE"
sed -i 's/IPAddressAllow=localhost/# IPAddressAllow=localhost (disabled for internet access)/' "$SERVICE_FILE"
sed -i 's/IPAddressAllow=127\.0\.0\.0\/8/# IPAddressAllow=127.0.0.0\/8 (disabled for internet access)/' "$SERVICE_FILE"
sed -i 's/IPAddressAllow=::1\/128/# IPAddressAllow=::1\/128 (disabled for internet access)/' "$SERVICE_FILE"

# Add internet access permission
if ! grep -q "IPAddressAllow=any" "$SERVICE_FILE"; then
    sed -i '/# Network settings/a IPAddressAllow=any' "$SERVICE_FILE"
fi

# Update network dependencies
sed -i 's/After=network\.target/After=network-online.target/' "$SERVICE_FILE"
sed -i 's/Wants=network\.target/Wants=network-online.target/' "$SERVICE_FILE"

# Add environment variables for timeouts if not present
if ! grep -q "HTTP_TIMEOUT" "$SERVICE_FILE"; then
    sed -i '/SyslogIdentifier=/a\\nEnvironment=HTTP_TIMEOUT=60s\nEnvironment=DIAL_TIMEOUT=30s\nEnvironment=TLS_HANDSHAKE_TIMEOUT=30s' "$SERVICE_FILE"
fi

# Reload systemd
print_status "Reloading systemd daemon..."
systemctl daemon-reload

# Start the service
print_status "Starting service..."
systemctl start "$SERVICE_NAME"

# Check status
sleep 3
if systemctl is-active --quiet "$SERVICE_NAME"; then
    print_status "✓ Service started successfully"
    print_status "Web interface should be available at: http://localhost:8080"
else
    print_error "✗ Service failed to start"
    echo "Check logs with: journalctl -u $SERVICE_NAME -f"
fi

print_status "Network connectivity fix applied!"
echo ""
echo "Changes made:"
echo "- Disabled IP address restrictions"
echo "- Updated network dependencies"
echo "- Added timeout environment variables"
echo "- Backup saved as: $SERVICE_FILE.backup"
echo ""
echo "If issues persist, run: make troubleshoot-network"
