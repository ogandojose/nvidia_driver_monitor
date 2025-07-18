#!/bin/bash

# NVIDIA Driver Monitor Network Troubleshooting Script

SERVICE_NAME="nvidia-driver-monitor"
SERVICE_USER="nvidia-monitor"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

print_header() {
    echo -e "${BLUE}=== $1 ===${NC}"
}

echo "NVIDIA Driver Monitor Network Troubleshooting"
echo "============================================"
echo ""

# Test 1: Check if service is running
print_header "Service Status"
if systemctl is-active --quiet "$SERVICE_NAME"; then
    print_status "Service is running"
else
    print_error "Service is not running"
    echo "Start it with: sudo systemctl start $SERVICE_NAME"
fi

if systemctl is-enabled --quiet "$SERVICE_NAME"; then
    print_status "Service is enabled"
else
    print_warning "Service is not enabled"
fi

echo ""

# Test 2: Check network connectivity from service user
print_header "Network Connectivity Test"

print_status "Testing network connectivity as $SERVICE_USER user..."

# Test basic connectivity
echo "Testing basic network connectivity:"
if sudo -u "$SERVICE_USER" ping -c 1 8.8.8.8 >/dev/null 2>&1; then
    print_status "✓ Basic network connectivity works"
else
    print_error "✗ Basic network connectivity failed"
fi

# Test DNS resolution
echo "Testing DNS resolution:"
if sudo -u "$SERVICE_USER" nslookup www.nvidia.com >/dev/null 2>&1; then
    print_status "✓ DNS resolution works"
else
    print_error "✗ DNS resolution failed"
fi

# Test HTTPS connectivity to NVIDIA
echo "Testing HTTPS connectivity to NVIDIA:"
if sudo -u "$SERVICE_USER" curl -s --max-time 10 "https://www.nvidia.com" >/dev/null 2>&1; then
    print_status "✓ HTTPS connection to NVIDIA works"
else
    print_error "✗ HTTPS connection to NVIDIA failed"
fi

# Test the specific URL that's failing
echo "Testing specific driver archive URL:"
if sudo -u "$SERVICE_USER" curl -s --max-time 10 "https://www.nvidia.com/en-us/drivers/unix/linux-amd64-display-archive/" >/dev/null 2>&1; then
    print_status "✓ Driver archive URL accessible"
else
    print_error "✗ Driver archive URL not accessible"
fi

echo ""

# Test 3: Check systemd restrictions
print_header "Systemd Service Restrictions"

SERVICE_FILE="/etc/systemd/system/$SERVICE_NAME.service"
if [ -f "$SERVICE_FILE" ]; then
    echo "Checking service file for network restrictions:"
    
    if grep -q "IPAddressDeny=any" "$SERVICE_FILE"; then
        print_error "✗ Service has IPAddressDeny=any (blocks internet access)"
        echo "  Consider using the minimal service configuration"
    else
        print_status "✓ No IP address restrictions found"
    fi
    
    if grep -q "PrivateNetwork=true" "$SERVICE_FILE"; then
        print_error "✗ Service has PrivateNetwork=true (isolates network)"
    else
        print_status "✓ No private network isolation"
    fi
    
    if grep -q "RestrictAddressFamilies" "$SERVICE_FILE"; then
        print_status "Service has address family restrictions:"
        grep "RestrictAddressFamilies" "$SERVICE_FILE"
    fi
else
    print_error "Service file not found: $SERVICE_FILE"
fi

echo ""

# Test 4: Check recent logs
print_header "Recent Service Logs"
echo "Last 10 lines of service logs:"
journalctl -u "$SERVICE_NAME" -n 10 --no-pager

echo ""

# Test 5: Manual test
print_header "Manual Test"
echo "Testing manual execution as service user:"
echo "Running: sudo -u $SERVICE_USER /opt/nvidia-driver-monitor/nvidia-web-server -addr :8081"
echo "This will test if the binary can access the internet when run manually..."
echo ""

# Run a quick test
timeout 15 sudo -u "$SERVICE_USER" /opt/nvidia-driver-monitor/nvidia-web-server -addr :8081 &
TEST_PID=$!
sleep 5

if ps -p $TEST_PID > /dev/null; then
    print_status "✓ Manual test started successfully"
    kill $TEST_PID 2>/dev/null
else
    print_error "✗ Manual test failed to start"
fi

echo ""

# Recommendations
print_header "Recommendations"
echo "Based on the tests above, here are some recommendations:"
echo ""
echo "1. If DNS resolution failed:"
echo "   - Check /etc/resolv.conf"
echo "   - Ensure systemd-resolved is running"
echo ""
echo "2. If HTTPS connection failed:"
echo "   - Check firewall settings"
echo "   - Verify proxy settings if behind corporate firewall"
echo "   - Consider using the minimal service configuration"
echo ""
echo "3. If service has network restrictions:"
echo "   - Reinstall with: sudo ./install-service.sh"
echo "   - Choose option 2 (minimal configuration)"
echo ""
echo "4. If manual test works but service doesn't:"
echo "   - Check service file network restrictions"
echo "   - Increase timeout values in service file"
echo "   - Add network dependencies to service file"
echo ""
echo "5. To switch to minimal service configuration:"
echo "   - sudo systemctl stop $SERVICE_NAME"
echo "   - sudo cp nvidia-driver-monitor-minimal.service /etc/systemd/system/$SERVICE_NAME.service"
echo "   - sudo systemctl daemon-reload"
echo "   - sudo systemctl start $SERVICE_NAME"
