#!/bin/bash

# Run NVIDIA Web Server with Real Mock Data
# Complete setup for running the web interface with real captured API data

set -e

echo "🚀 STARTING NVIDIA WEB SERVER WITH REAL MOCK DATA"
echo "================================================="
echo ""

cd "$(dirname "$0")"

# Check if required binaries exist
if [ ! -f "nvidia-mock-server" ]; then
    echo "❌ nvidia-mock-server binary not found. Building..."
    make mock
fi

if [ ! -f "nvidia-web-server" ]; then
    echo "❌ nvidia-web-server binary not found. Building..."
    make web
fi

# Check if real mock data exists
if [ ! -d "test-data/launchpad/sources" ] || [ -z "$(ls -A test-data/launchpad/sources)" ]; then
    echo "❌ Real mock data not found. Run setup first:"
    echo "   bash setup-real-mock-data.sh"
    echo "   bash organize-real-mock-data.sh"
    exit 1
fi

echo "📋 Starting services..."
echo ""

# Start mock server
echo "1. 🗄️  Starting mock server with real data on port 9998..."
if pgrep -f "nvidia-mock-server" > /dev/null; then
    echo "   ⚠️  Mock server already running, killing existing process..."
    killall nvidia-mock-server 2>/dev/null || true
    sleep 2
fi

./nvidia-mock-server -data-dir test-data -port 9998 &
MOCK_PID=$!
echo "   ✅ Mock server started (PID: $MOCK_PID)"

# Wait for mock server to start
sleep 3

# Test mock server
echo "2. 🧪 Testing mock server..."
if curl -s "http://localhost:9998/launchpad/devel/ubuntu/+archive/primary/?ws.op=getPublishedSources&source_name=nvidia-graphics-drivers-570" | jq -e '.total_size > 0' >/dev/null 2>&1; then
    echo "   ✅ Mock server responding with real data"
else
    echo "   ❌ Mock server not responding properly"
    kill $MOCK_PID 2>/dev/null || true
    exit 1
fi

# Start web server
echo "3. 🌐 Starting web server on port 8080..."
if pgrep -f "nvidia-web-server" > /dev/null; then
    echo "   ⚠️  Web server already running, killing existing process..."
    killall nvidia-web-server 2>/dev/null || true
    sleep 2
fi

./nvidia-web-server --config config-real-mock.json --addr :8080 &
WEB_PID=$!
echo "   ✅ Web server started (PID: $WEB_PID)"

# Wait for web server to start
sleep 5

# Test web server
echo "4. 🧪 Testing web server..."
if curl -s http://localhost:8080/api/health | jq -e '.status == "healthy"' >/dev/null 2>&1; then
    echo "   ✅ Web server responding"
else
    echo "   ❌ Web server not responding properly"
    kill $MOCK_PID $WEB_PID 2>/dev/null || true
    exit 1
fi

echo ""
echo "🎉 SUCCESS! Both servers are running with real data"
echo "==============================================="
echo ""
echo "📋 Service URLs:"
echo "   🌐 Web Interface: http://localhost:8080"
echo "   🗄️  Mock Server:   http://localhost:9998"
echo ""
echo "🔗 API Endpoints:"
echo "   • Health:      http://localhost:8080/api/health"
echo "   • LRM Data:    http://localhost:8080/api/lrm"
echo "   • Statistics:  http://localhost:8080/api/statistics"
echo "   • Cache Status: http://localhost:8080/api/cache-status"
echo ""
echo "💡 The web server is now serving real NVIDIA driver data captured from live APIs!"
echo ""
echo "🛑 To stop both servers:"
echo "   killall nvidia-mock-server nvidia-web-server"
echo ""

# Keep script running to show process IDs
echo "📊 Process IDs:"
echo "   Mock Server: $MOCK_PID"
echo "   Web Server:  $WEB_PID"
echo ""
echo "⌨️  Press Ctrl+C to stop both servers..."

# Trap to cleanup on exit
cleanup() {
    echo ""
    echo "🛑 Stopping servers..."
    kill $MOCK_PID $WEB_PID 2>/dev/null || true
    echo "✅ Servers stopped"
    exit 0
}

trap cleanup SIGINT SIGTERM

# Wait for user interrupt
wait
