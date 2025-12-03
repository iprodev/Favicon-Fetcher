#!/bin/bash

# Build script for Linux/Mac

echo "========================================"
echo "Building Favicon Fetcher"
echo "Full format support: PNG, WebP, AVIF"
echo "========================================"

# Disable CGO for static build
export CGO_ENABLED=0

# Clean previous builds
rm -f favicon-server
rm -f bin/favicon-server

# Build
echo ""
echo "Compiling..."
go build -v -ldflags="-s -w" -o favicon-server ./cmd/server

if [ $? -eq 0 ]; then
    echo ""
    echo "========================================"
    echo "Build successful!"
    echo "========================================"
    echo ""
    echo "Binary: favicon-server"
    echo "Formats: PNG, WebP, AVIF"
    echo ""
    echo "To run:"
    echo "  ./favicon-server"
    echo ""
    echo "To run with custom settings:"
    echo "  ./favicon-server -port 8080 -log-level debug"
    echo ""
else
    echo ""
    echo "========================================"
    echo "Build FAILED!"
    echo "========================================"
    echo ""
    exit 1
fi
