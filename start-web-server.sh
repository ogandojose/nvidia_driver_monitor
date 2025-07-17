#!/bin/bash

# NVIDIA Driver Package Web Service Startup Script

echo "Building NVIDIA Driver Package Web Service..."
go build -o web-server ./cmd/web/

if [ $? -eq 0 ]; then
    echo "Build successful!"
    echo "Starting web server on port 8080..."
    echo "Open your browser and navigate to: http://localhost:8080"
    echo "Press Ctrl+C to stop the server"
    echo ""
    ./web-server -addr :8080
else
    echo "Build failed!"
    exit 1
fi
