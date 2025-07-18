#!/bin/bash

# NVIDIA Driver Monitor Service Management Script

SERVICE_NAME="nvidia-driver-monitor"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

print_header() {
    echo -e "${BLUE}$1${NC}"
}

# Function to check if service exists
service_exists() {
    systemctl list-unit-files | grep -q "^$SERVICE_NAME.service"
}

# Function to show service status
show_status() {
    if service_exists; then
        print_header "=== Service Status ==="
        systemctl status "$SERVICE_NAME" --no-pager
        echo ""
        
        print_header "=== Service Configuration ==="
        echo "Service file: /etc/systemd/system/$SERVICE_NAME.service"
        echo "Install directory: /opt/nvidia-driver-monitor"
        echo "Web interface: http://localhost:8080"
        echo "User: nvidia-monitor"
        echo ""
        
        if systemctl is-active --quiet "$SERVICE_NAME"; then
            print_status "Service is running"
        else
            print_warning "Service is not running"
        fi
        
        if systemctl is-enabled --quiet "$SERVICE_NAME"; then
            print_status "Service is enabled (will start on boot)"
        else
            print_warning "Service is not enabled"
        fi
    else
        print_error "Service is not installed"
        exit 1
    fi
}

# Function to show logs
show_logs() {
    if service_exists; then
        print_header "=== Recent Logs ==="
        journalctl -u "$SERVICE_NAME" --no-pager -n 50
        echo ""
        print_status "To follow logs in real-time, run:"
        echo "  sudo journalctl -u $SERVICE_NAME -f"
    else
        print_error "Service is not installed"
        exit 1
    fi
}

# Function to start service
start_service() {
    if service_exists; then
        print_status "Starting $SERVICE_NAME service..."
        systemctl start "$SERVICE_NAME"
        sleep 2
        if systemctl is-active --quiet "$SERVICE_NAME"; then
            print_status "Service started successfully"
            print_status "Web interface available at: http://localhost:8080"
        else
            print_error "Failed to start service"
            exit 1
        fi
    else
        print_error "Service is not installed"
        exit 1
    fi
}

# Function to stop service
stop_service() {
    if service_exists; then
        print_status "Stopping $SERVICE_NAME service..."
        systemctl stop "$SERVICE_NAME"
        sleep 2
        if ! systemctl is-active --quiet "$SERVICE_NAME"; then
            print_status "Service stopped successfully"
        else
            print_error "Failed to stop service"
            exit 1
        fi
    else
        print_error "Service is not installed"
        exit 1
    fi
}

# Function to restart service
restart_service() {
    if service_exists; then
        print_status "Restarting $SERVICE_NAME service..."
        systemctl restart "$SERVICE_NAME"
        sleep 2
        if systemctl is-active --quiet "$SERVICE_NAME"; then
            print_status "Service restarted successfully"
            print_status "Web interface available at: http://localhost:8080"
        else
            print_error "Failed to restart service"
            exit 1
        fi
    else
        print_error "Service is not installed"
        exit 1
    fi
}

# Function to enable service
enable_service() {
    if service_exists; then
        print_status "Enabling $SERVICE_NAME service..."
        systemctl enable "$SERVICE_NAME"
        print_status "Service enabled (will start on boot)"
    else
        print_error "Service is not installed"
        exit 1
    fi
}

# Function to disable service
disable_service() {
    if service_exists; then
        print_status "Disabling $SERVICE_NAME service..."
        systemctl disable "$SERVICE_NAME"
        print_status "Service disabled (will not start on boot)"
    else
        print_error "Service is not installed"
        exit 1
    fi
}

# Function to show usage
show_usage() {
    echo "NVIDIA Driver Monitor Service Management"
    echo ""
    echo "Usage: $0 {start|stop|restart|status|logs|enable|disable|help}"
    echo ""
    echo "Commands:"
    echo "  start    - Start the service"
    echo "  stop     - Stop the service"
    echo "  restart  - Restart the service"
    echo "  status   - Show service status and information"
    echo "  logs     - Show recent service logs"
    echo "  enable   - Enable service to start on boot"
    echo "  disable  - Disable service from starting on boot"
    echo "  help     - Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 start"
    echo "  $0 status"
    echo "  $0 logs"
    echo ""
    echo "Web interface: http://localhost:8080"
}

# Main script logic
case "$1" in
    start)
        start_service
        ;;
    stop)
        stop_service
        ;;
    restart)
        restart_service
        ;;
    status)
        show_status
        ;;
    logs)
        show_logs
        ;;
    enable)
        enable_service
        ;;
    disable)
        disable_service
        ;;
    help|--help|-h)
        show_usage
        ;;
    *)
        print_error "Unknown command: $1"
        echo ""
        show_usage
        exit 1
        ;;
esac
