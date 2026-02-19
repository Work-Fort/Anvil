#!/bin/sh
# Cracker Barrel CLI Installer
# Usage: curl -fsSL https://kernels.workfort.io/install-anvil.sh | sh
#
# This script installs the anvil CLI tool.
# Note: This is a monorepo with multiple release types:
#   - Kernel releases: tagged as v6.18.9, v6.18.8, etc.
#   - CLI releases: tagged as cli-v1.0.0, cli-v1.0.1, etc.

set -e

# XDG Base Directory
XDG_DATA_HOME="${XDG_DATA_HOME:-$HOME/.local/share}"
CB_BIN_DIR="$XDG_DATA_HOME/anvil/bin"

echo "Cracker Barrel CLI Installer"
echo "=============================="
echo ""

# Check for required dependencies
# Note: anvil uses native Go implementations for compression, checksums,
# and tar extraction. Only GPG is required for PGP signature verification.
echo "Checking dependencies..."

if ! command -v gpg >/dev/null 2>&1; then
    echo "Error: Missing required dependency: gpg"
    echo ""
    echo "Install gpg/gnupg:"
    echo "  Ubuntu/Debian: sudo apt-get install gnupg"
    echo "  Arch: sudo pacman -S gnupg"
    echo "  Fedora/RHEL: sudo dnf install gnupg"
    echo "  macOS: brew install gnupg"
    exit 1
fi

echo "✓ All dependencies found"
echo ""

# Check if anvil already exists
if command -v anvil >/dev/null 2>&1; then
    EXISTING_PATH=$(command -v anvil)
    echo "anvil is already installed at: $EXISTING_PATH"
    echo ""

    # Check if it's the expected location
    if [ "$EXISTING_PATH" = "$CB_BIN_DIR/anvil" ]; then
        echo "✓ anvil is properly installed in $CB_BIN_DIR"
        echo "✓ anvil is on your PATH"
    else
        echo "Note: anvil is installed at a different location"
        echo "  Found at: $EXISTING_PATH"
        echo "  Expected: $CB_BIN_DIR/anvil"
    fi

    echo ""
    echo "Run 'anvil version' to check your version"
    echo "Run 'anvil update' to update to the latest version"
    exit 0
fi

# anvil not found, proceed with installation
echo "Installing anvil..."
echo ""

# Create bin directory
mkdir -p "$CB_BIN_DIR"

# Download latest anvil release
echo "Downloading anvil from GitHub..."

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)
        ARCH="x86_64"
        ;;
    aarch64|arm64)
        ARCH="aarch64"
        ;;
    *)
        echo "Error: Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
if [ "$OS" != "linux" ]; then
    echo "Error: Only Linux is currently supported"
    exit 1
fi

# Fetch all releases and find the latest CLI release
# Note: This repo contains both kernel releases (v6.x.x) and CLI releases (cli-v*)
echo "Fetching latest CLI release information..."
ALL_RELEASES=$(curl -fsSL "https://api.github.com/repos/Work-Fort/Anvil/releases" 2>&1)

if [ -z "$ALL_RELEASES" ] || echo "$ALL_RELEASES" | grep -q "API rate limit"; then
    echo "Error: Failed to fetch release information from GitHub"
    exit 1
fi

# Find the latest CLI release (tagged with cli-v prefix)
CLI_RELEASE=$(echo "$ALL_RELEASES" | grep -A 100 '"tag_name": "cli-v' | head -100)

if [ -z "$CLI_RELEASE" ]; then
    echo "Error: No CLI releases found"
    echo ""
    echo "The CLI has not been released yet. To build from source:"
    echo "  git clone https://github.com/Work-Fort/Anvil.git"
    echo "  cd anvil/cli"
    echo "  task install"
    exit 1
fi

# Extract download URL for our architecture
# Looking for asset like: anvil-linux-x86_64 or anvil-linux-aarch64
ASSET_NAME="anvil-${OS}-${ARCH}"
DOWNLOAD_URL=$(echo "$CLI_RELEASE" | grep -o "\"browser_download_url\": *\"[^\"]*${ASSET_NAME}\"" | head -1 | cut -d'"' -f4)

if [ -z "$DOWNLOAD_URL" ]; then
    echo "Error: Could not find release binary for ${OS}-${ARCH}"
    echo ""
    echo "Expected asset name: ${ASSET_NAME}"
    echo ""
    echo "Build from source instead:"
    echo "  git clone https://github.com/Work-Fort/Anvil.git"
    echo "  cd anvil/cli"
    echo "  task install"
    exit 1
fi

echo "Downloading from: $DOWNLOAD_URL"
if ! curl -fL -o "$CB_BIN_DIR/anvil" "$DOWNLOAD_URL"; then
    echo "Error: Failed to download anvil"
    exit 1
fi

chmod +x "$CB_BIN_DIR/anvil"

echo "✓ anvil installed to: $CB_BIN_DIR/anvil"
echo ""

# Check if bin directory is on PATH
if echo "$PATH" | grep -q "$CB_BIN_DIR"; then
    echo "✓ $CB_BIN_DIR is already on your PATH"
    echo ""
    echo "You can now run: anvil --help"
else
    echo "Setup required: Add anvil to your PATH"
    echo ""
    echo "Add this line to your shell configuration file:"
    echo "  ~/.bashrc (for bash)"
    echo "  ~/.zshrc (for zsh)"
    echo ""
    echo "  export PATH=\"\$PATH:$CB_BIN_DIR\""
    echo ""
    echo "Then reload your shell:"
    echo "  source ~/.bashrc"
    echo "  # or"
    echo "  source ~/.zshrc"
    echo ""
    echo "Or run anvil directly:"
    echo "  $CB_BIN_DIR/anvil --help"
fi

echo ""
echo "✓ Installation complete!"
