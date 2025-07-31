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

# Service file options
STANDARD_SERVICE_FILE="nvidia-driver-monitor.service"
HTTPS_SERVICE_FILE="nvidia-driver-monitor-https.service"

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

# Check if templates directory exists
if [ ! -d "./templates" ]; then
    print_error "templates directory not found. HTML template files are required."
    exit 1
fi

# Check if static directory exists
if [ ! -d "./static" ]; then
    print_error "static directory not found. CSS and JavaScript files are required."
    exit 1
fi

# Check if required template files exist
if [ ! -f "./templates/lrm_verifier.html" ]; then
    print_error "Required template file templates/lrm_verifier.html not found."
    exit 1
fi

if [ ! -f "./templates/statistics.html" ]; then
    print_error "Required template file templates/statistics.html not found."
    exit 1
fi

if [ ! -f "./templates/index.html" ]; then
    print_error "Required template file templates/index.html not found."
    exit 1
fi

# Check if required static files exist
if [ ! -f "./static/css/statistics.css" ]; then
    print_error "Required CSS file static/css/statistics.css not found."
    exit 1
fi

if [ ! -f "./static/js/statistics.js" ]; then
    print_error "Required JavaScript file static/js/statistics.js not found."
    exit 1
fi

# Check if required configuration files exist
if [ ! -f "./supportedReleases.json" ]; then
    print_error "supportedReleases.json not found. This file is required for the service."
    exit 1
fi

if [ ! -f "./config.json" ]; then
    print_error "config.json not found. This file is required for service configuration."
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
cp "./config.json" "$INSTALL_DIR/"
chown "$SERVICE_USER:$SERVICE_GROUP" "$INSTALL_DIR/nvidia-web-server"
chown "$SERVICE_USER:$SERVICE_GROUP" "$INSTALL_DIR/supportedReleases.json"
chown "$SERVICE_USER:$SERVICE_GROUP" "$INSTALL_DIR/config.json"
chmod 755 "$INSTALL_DIR/nvidia-web-server"
chmod 644 "$INSTALL_DIR/supportedReleases.json"
chmod 644 "$INSTALL_DIR/config.json"

# Copy templates directory
print_status "Installing template files..."
cp -r "./templates" "$INSTALL_DIR/"
chown -R "$SERVICE_USER:$SERVICE_GROUP" "$INSTALL_DIR/templates"
find "$INSTALL_DIR/templates" -type f -name "*.html" -exec chmod 644 {} \;
find "$INSTALL_DIR/templates" -type d -exec chmod 755 {} \;

# Copy static assets directory (CSS, JavaScript, etc.)
print_status "Installing static assets..."
cp -r "./static" "$INSTALL_DIR/"
chown -R "$SERVICE_USER:$SERVICE_GROUP" "$INSTALL_DIR/static"
find "$INSTALL_DIR/static" -type f \( -name "*.css" -o -name "*.js" \) -exec chmod 644 {} \;
find "$INSTALL_DIR/static" -type d -exec chmod 755 {} \;

# Function to generate self-signed certificate
generate_certificate() {
    local cert_file="$INSTALL_DIR/server.crt"
    local key_file="$INSTALL_DIR/server.key"
    
    print_status "Generating self-signed SSL certificate..."
    
    # Check if openssl is available
    if ! command -v openssl &> /dev/null; then
        print_error "OpenSSL is required to generate certificates. Please install openssl package."
        exit 1
    fi
    
    # Generate private key and certificate
    openssl req -x509 -newkey rsa:4096 -keyout "$key_file" -out "$cert_file" \
        -days 365 -nodes -subj "/C=US/ST=State/L=City/O=Organization/CN=localhost" \
        2>/dev/null
    
    if [ $? -eq 0 ]; then
        chown "$SERVICE_USER:$SERVICE_GROUP" "$cert_file" "$key_file"
        chmod 600 "$key_file"  # Private key should be readable only by owner
        chmod 644 "$cert_file"  # Certificate can be world-readable
        print_status "SSL certificate generated successfully"
    else
        print_error "Failed to generate SSL certificate"
        exit 1
    fi
}

# Install systemd service file
print_status "Installing systemd service file..."

# Ask user which service file to use
echo ""
echo "Choose service configuration:"
echo "1) HTTPS (recommended - encrypted connection on port 8443)"
echo "2) HTTP (standard - unencrypted connection on port 8080)"
read -p "Enter choice [1-2]: " -n 1 -r
echo ""

case $REPLY in
    1)
        SERVICE_SOURCE="$HTTPS_SERVICE_FILE"
        print_status "Using HTTPS service configuration"
        generate_certificate
        ;;
    2)
        SERVICE_SOURCE="$STANDARD_SERVICE_FILE"
        print_status "Using HTTP service configuration"
        ;;
    *)
        print_warning "Invalid choice, using HTTPS configuration (recommended)"
        SERVICE_SOURCE="$HTTPS_SERVICE_FILE"
        generate_certificate
        ;;
esac

if [ ! -f "./$SERVICE_SOURCE" ]; then
    print_error "Service file $SERVICE_SOURCE not found"
    exit 1
fi

cp "./$SERVICE_SOURCE" "$SERVICE_FILE"
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

# Display appropriate URL based on service type
if [ "$SERVICE_SOURCE" = "$HTTPS_SERVICE_FILE" ]; then
    echo "Web interface will be available at: https://localhost:8443"
    echo "Note: HTTPS uses a self-signed certificate. Your browser may show a security warning."
else
    echo "Web interface will be available at: http://localhost:8080"
fi

echo "Service user: $SERVICE_USER"
echo "Install directory: $INSTALL_DIR"
echo ""
echo "To start the service now, run:"
echo "  sudo systemctl start $SERVICE_NAME"
