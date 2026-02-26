#!/usr/bin/env bash
set -e

# Default settings
REPO="frostyeti/cast"
GITHUB_URL="https://github.com"
API_URL="https://api.github.com/repos/$REPO/releases/latest"

# Detect OS and Arch
OS="$(uname -s)"
ARCH="$(uname -m)"

# Map OS
case "$OS" in
  Linux)    OS_NAME="linux" ;;
  Darwin)   OS_NAME="darwin" ;;
  CYGWIN*|MINGW32*|MSYS*|MINGW*|Windows_NT) OS_NAME="windows" ;;
  *)        
    if [ "$OS" = "Windows_NT" ] || [ "$WINDIR" != "" ]; then
        OS_NAME="windows"
    else
        echo "Unsupported OS: $OS"; exit 1 
    fi
    ;;
esac

# Map Architecture
case "$ARCH" in
  x86_64|amd64) ARCH_NAME="amd64" ;;
  arm64|aarch64) ARCH_NAME="arm64" ;;
  i386|i686)    ARCH_NAME="386" ;;
  *)            echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Default Install Directory
if [ -z "$CAST_INSTALL_DIR" ]; then
  if [ "$OS_NAME" = "windows" ]; then
    CAST_INSTALL_DIR="$USERPROFILE/AppData/Local/Programs/bin"
  else
    CAST_INSTALL_DIR="$HOME/.local/bin"
  fi
fi

# Ensure install directory exists
mkdir -p "$CAST_INSTALL_DIR"

# File extension
EXT="tar.gz"
if [ "$OS_NAME" = "windows" ]; then
  EXT="zip"
fi

echo "Fetching latest release information for $REPO..."

# Get download URL for latest release
# NOTE: If jq is not available, we can use grep/awk. We'll use grep for better portability.
if command -v jq >/dev/null 2>&1; then
  DOWNLOAD_URL=$(curl -s "$API_URL" | jq -r ".assets[]? // empty | select(.name | contains(\"cast-${OS_NAME}-${ARCH_NAME}\")) | select(.name | endswith(\".${EXT}\")) | .browser_download_url")
else
  # Grep logic - match exactly cast-linux-amd64-v0.1.0-alpha.0.tar.gz pattern
  DOWNLOAD_URL=$(curl -s "$API_URL" | grep -o "https://github.com/.*/releases/download/.*/cast-${OS_NAME}-${ARCH_NAME}-v.*\.${EXT}")
fi

if [ -z "$DOWNLOAD_URL" ]; then
  echo "Error: Could not find a release for $OS_NAME $ARCH_NAME"
  echo "Note: If there are no published releases yet, this will fail."
  exit 1
fi

echo "Downloading Cast from $DOWNLOAD_URL..."

TMP_DIR=$(mktemp -d)
TMP_FILE="$TMP_DIR/cast.$EXT"

curl -sL "$DOWNLOAD_URL" -o "$TMP_FILE"

echo "Extracting..."
if [ "$EXT" = "zip" ]; then
  unzip -q "$TMP_FILE" -d "$TMP_DIR"
else
  tar -xzf "$TMP_FILE" -C "$TMP_DIR"
fi

# Move binary to install dir
if [ "$OS_NAME" = "windows" ]; then
  mv "$TMP_DIR/cast.exe" "$CAST_INSTALL_DIR/"
  echo "Cast installed to $CAST_INSTALL_DIR/cast.exe"
else
  mv "$TMP_DIR/cast" "$CAST_INSTALL_DIR/"
  chmod +x "$CAST_INSTALL_DIR/cast"
  echo "Cast installed to $CAST_INSTALL_DIR/cast"
fi

# Clean up
rm -rf "$TMP_DIR"

# Check if CAST_INSTALL_DIR is in PATH
if [[ ":$PATH:" != *":$CAST_INSTALL_DIR:"* ]]; then
  echo "================================================================================"
  echo "WARNING: $CAST_INSTALL_DIR is not in your PATH."
  echo "Please add the following line to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
  echo "export PATH=\"\$PATH:$CAST_INSTALL_DIR\""
  echo "================================================================================"
fi

echo "Installation complete! Run 'cast --help' to get started."
