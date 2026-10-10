#!/bin/bash

# Cross-compile the Go backend for all supported platforms
# Outputs binaries to resources/bin/<platform>-<arch>/

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DESKTOP_DIR="$(dirname "$SCRIPT_DIR")"
BACKEND_DIR="$DESKTOP_DIR/backend-go"
OUTPUT_DIR="$DESKTOP_DIR/resources/bin"

BINARY_NAME="businessos-server"

echo "Building Go backend for Electron distribution..."
echo "Backend directory: $BACKEND_DIR"
echo "Output directory: $OUTPUT_DIR"

# Navigate to backend directory
cd "$BACKEND_DIR"

# Ensure output directories exist
mkdir -p "$OUTPUT_DIR/darwin-arm64"
mkdir -p "$OUTPUT_DIR/darwin-x64"
mkdir -p "$OUTPUT_DIR/win32-x64"
mkdir -p "$OUTPUT_DIR/linux-x64"

# macOS ARM64 (Apple Silicon)
echo "Building for macOS ARM64..."
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o "$OUTPUT_DIR/darwin-arm64/$BINARY_NAME" ./cmd/server

# macOS x64 (Intel)
echo "Building for macOS x64..."
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o "$OUTPUT_DIR/darwin-x64/$BINARY_NAME" ./cmd/server

# Windows x64
echo "Building for Windows x64..."
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o "$OUTPUT_DIR/win32-x64/$BINARY_NAME.exe" ./cmd/server

# Linux x64
echo "Building for Linux x64..."
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "$OUTPUT_DIR/linux-x64/$BINARY_NAME" ./cmd/server

# Make binaries executable
chmod +x "$OUTPUT_DIR/darwin-arm64/$BINARY_NAME" 2>/dev/null || true
chmod +x "$OUTPUT_DIR/darwin-x64/$BINARY_NAME" 2>/dev/null || true
chmod +x "$OUTPUT_DIR/linux-x64/$BINARY_NAME" 2>/dev/null || true

echo ""
echo "Backend builds complete!"
echo "Binaries:"
ls -la "$OUTPUT_DIR"/*/ 2>/dev/null || echo "No binaries found"
