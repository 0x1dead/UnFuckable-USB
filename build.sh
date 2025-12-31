#!/bin/bash

# UnFuckable USB Build Script
# Author: 0x1dead

VERSION="1.0.3"
APP_NAME="unfuckable-usb"
OUTPUT_DIR="dist"

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║           UnFuckable USB - Build Script v${VERSION}             ║"
echo "║        Making your data impossible to fuck with              ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

# Create output directory
mkdir -p ${OUTPUT_DIR}

# Get dependencies
echo "[*] Getting dependencies..."
go mod tidy

# Build flags for smaller binary
LDFLAGS="-s -w -X main.AppVersion=${VERSION}"

# Check for Windows resource compiler (for icon)
if [ -f "icon.ico" ]; then
    echo "[*] Found icon.ico, compiling Windows resources..."
    
    # Try different resource compilers
    if command -v x86_64-w64-mingw32-windres &> /dev/null; then
        x86_64-w64-mingw32-windres -o resource_amd64.syso resource.rc
        echo "    ✓ Compiled resource_amd64.syso"
    elif command -v windres &> /dev/null; then
        windres -o resource_amd64.syso resource.rc
        echo "    ✓ Compiled resource_amd64.syso"
    else
        echo "    ⚠ windres not found, Windows build will have no icon"
        echo "    Install mingw-w64 for icon support: apt install mingw-w64"
    fi
else
    echo "[*] No icon.ico found, Windows build will have no icon"
fi

# Build for Linux AMD64
echo "[*] Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o ${OUTPUT_DIR}/${APP_NAME}-linux-amd64 .
if [ $? -eq 0 ]; then
    echo "    ✓ ${OUTPUT_DIR}/${APP_NAME}-linux-amd64"
else
    echo "    ✗ Linux build failed"
fi

# Build for Linux ARM64
echo "[*] Building for Linux (arm64)..."
GOOS=linux GOARCH=arm64 go build -ldflags="${LDFLAGS}" -o ${OUTPUT_DIR}/${APP_NAME}-linux-arm64 .
if [ $? -eq 0 ]; then
    echo "    ✓ ${OUTPUT_DIR}/${APP_NAME}-linux-arm64"
else
    echo "    ✗ Linux ARM64 build failed"
fi

# Build for Windows AMD64 (with icon if available)
echo "[*] Building for Windows (amd64)..."
if [ -f "resource_amd64.syso" ]; then
    GOOS=windows GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o ${OUTPUT_DIR}/${APP_NAME}-windows-amd64.exe .
else
    GOOS=windows GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o ${OUTPUT_DIR}/${APP_NAME}-windows-amd64.exe .
fi
if [ $? -eq 0 ]; then
    echo "    ✓ ${OUTPUT_DIR}/${APP_NAME}-windows-amd64.exe"
else
    echo "    ✗ Windows build failed"
fi

# Build for Windows ARM64
echo "[*] Building for Windows (arm64)..."
GOOS=windows GOARCH=arm64 go build -ldflags="${LDFLAGS}" -o ${OUTPUT_DIR}/${APP_NAME}-windows-arm64.exe .
if [ $? -eq 0 ]; then
    echo "    ✓ ${OUTPUT_DIR}/${APP_NAME}-windows-arm64.exe"
else
    echo "    ✗ Windows ARM64 build failed"
fi

# Build for macOS AMD64
echo "[*] Building for macOS (amd64)..."
GOOS=darwin GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o ${OUTPUT_DIR}/${APP_NAME}-macos-amd64 .
if [ $? -eq 0 ]; then
    echo "    ✓ ${OUTPUT_DIR}/${APP_NAME}-macos-amd64"
else
    echo "    ✗ macOS build failed"
fi

# Build for macOS ARM64 (Apple Silicon)
echo "[*] Building for macOS (arm64/Apple Silicon)..."
GOOS=darwin GOARCH=arm64 go build -ldflags="${LDFLAGS}" -o ${OUTPUT_DIR}/${APP_NAME}-macos-arm64 .
if [ $? -eq 0 ]; then
    echo "    ✓ ${OUTPUT_DIR}/${APP_NAME}-macos-arm64"
else
    echo "    ✗ macOS ARM64 build failed"
fi

# Cleanup
rm -f resource_amd64.syso resource_arm64.syso

echo ""
echo "[*] Build complete!"
echo ""

# Show file sizes
echo "File sizes:"
ls -lh ${OUTPUT_DIR}/ | grep ${APP_NAME} | awk '{print "    " $9 " - " $5}'

echo ""
echo "Done!"
