#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
set -euo pipefail

# Vendor libguestfs golang bindings
# This script extracts the generated golang bindings from libguestfs and vendors them
# into your project. Build once, vendor the generated code.
#
# Project-agnostic: can be extracted and used in any Go project

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Usage
usage() {
    cat <<EOF
Usage: $0 [OPTIONS]

Vendor libguestfs golang bindings into your Go project.

Options:
  --output DIR              Target directory for vendored bindings (default: third_party/libguestfs.org/guestfs)
  --version VERSION         libguestfs version to vendor (default: 1.52.0)
  --skip-verify             Skip GPG signature verification (not recommended)
  --help                    Show this help message

Examples:
  $0                                                    # Use default version (1.52.0), vendor to default location
  $0 --output internal/libguestfs                       # Default version, custom location
  $0 --version 1.58.0                                   # Specific version
  $0 --output myvendor/guestfs --version 1.58.0         # Both options

Environment Variables:
  VENDOR_DIR                Output directory (overridden by --output)
  VERSION                   libguestfs version (default: 1.52.0, overridden by --version)

The script tries system packages first (Ubuntu/Debian golang-guestfs-dev),
then falls back to building from source (Arch Linux and others).
EOF
    exit 0
}

# Set defaults
VENDOR_DIR="${VENDOR_DIR:-third_party/libguestfs.org/guestfs}"
LIBGUESTFS_VERSION="${VERSION:-1.52.0}"
SKIP_VERIFY="${SKIP_VERIFY:-false}"
TEMP_BUILD_DIR=""

# Parse arguments (overrides env vars)
while [[ $# -gt 0 ]]; do
    case $1 in
        --output)
            VENDOR_DIR="$2"
            shift 2
            ;;
        --version)
            LIBGUESTFS_VERSION="$2"
            shift 2
            ;;
        --skip-verify)
            SKIP_VERIFY=true
            shift
            ;;
        --help)
            usage
            ;;
        *)
            error "Unknown option: $1\nUse --help for usage information."
            ;;
    esac
done

# Cleanup on exit
cleanup() {
    if [ -n "$TEMP_BUILD_DIR" ] && [ -d "$TEMP_BUILD_DIR" ]; then
        info "Cleaning up temporary build directory..."
        rm -rf "$TEMP_BUILD_DIR"
    fi
}
trap cleanup EXIT INT TERM

# Check if bindings are already vendored
check_existing_bindings() {
    if [ -d "$VENDOR_DIR" ] && [ -f "$VENDOR_DIR/guestfs.go" ]; then
        warn "Bindings already vendored at $VENDOR_DIR"
        read -p "Overwrite existing bindings? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            info "Keeping existing bindings"
            exit 0
        fi
        info "Removing existing bindings..."
        rm -rf "$VENDOR_DIR"
    fi
}

# Import libguestfs GPG key
import_gpg_key() {
    # Richard W.M. Jones (Red Hat key) <rjones@redhat.com>
    local key_id="F7774FB1AD074A7E8C8767EA91738F73E1B768A0"

    info "Importing libguestfs GPG key..."
    # Try keyserver import
    if gpg --keyserver hkps://keyserver.ubuntu.com --recv-keys "$key_id" 2>/dev/null; then
        return 0
    fi

    # Fallback to keys.openpgp.org
    if gpg --keyserver hkps://keys.openpgp.org --recv-keys "$key_id" 2>/dev/null; then
        return 0
    fi

    warn "Failed to import GPG key from keyservers"
    return 1
}

# Verify GPG signature
verify_signature() {
    local tarball="$1"
    local signature="${tarball}.sig"

    if [ "$SKIP_VERIFY" = "true" ]; then
        warn "Skipping signature verification (--skip-verify)"
        return 0
    fi

    if ! command -v gpg >/dev/null 2>&1; then
        warn "GPG not available, skipping signature verification"
        warn "Install gnupg or use --skip-verify to suppress this warning"
        return 0
    fi

    info "Verifying GPG signature..."

    # Import key if needed
    if ! gpg --list-keys F7774FB1AD074A7E8C8767EA91738F73E1B768A0 >/dev/null 2>&1; then
        if ! import_gpg_key; then
            warn "Could not import GPG key, skipping verification"
            warn "To verify manually: gpg --recv-keys F7774FB1AD074A7E8C8767EA91738F73E1B768A0"
            return 0
        fi
    fi

    # Verify signature
    if ! gpg --verify "$signature" "$tarball" 2>&1 | grep -q "Good signature"; then
        error "GPG signature verification FAILED!\nThis could indicate a compromised download.\nUse --skip-verify to override (not recommended)."
    fi

    info "✓ GPG signature verified"
    return 0
}

# Try to use system-provided bindings (Ubuntu/Debian)
try_system_bindings() {
    info "Checking for system-provided golang-guestfs bindings..."

    # Check if golang-guestfs-dev package is installed
    if command -v dpkg >/dev/null 2>&1; then
        if dpkg -l golang-guestfs-dev 2>/dev/null | grep -q '^ii'; then
            info "Found golang-guestfs-dev package"

            # Find the bindings in /usr/share/gocode
            local binding_path="/usr/share/gocode/src/libguestfs.org/guestfs"
            if [ -d "$binding_path" ]; then
                info "Copying system bindings from $binding_path"
                mkdir -p "$VENDOR_DIR"
                cp -r "$binding_path"/* "$VENDOR_DIR/"
                info "Successfully vendored system bindings"
                return 0
            fi
        fi
    fi

    return 1
}

# Build libguestfs and extract golang bindings
build_and_extract_bindings() {
    info "Building libguestfs $LIBGUESTFS_VERSION from source to extract golang bindings..."

    # Check build dependencies
    local missing_deps=()
    for cmd in curl tar gcc make pkg-config go; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            missing_deps+=("$cmd")
        fi
    done

    if [ ${#missing_deps[@]} -gt 0 ]; then
        error "Missing build dependencies: ${missing_deps[*]}\nInstall them with your package manager."
    fi

    # Create temporary build directory
    TEMP_BUILD_DIR=$(mktemp -d -t libguestfs-build-XXXXXX)
    local original_dir="$PWD"
    cd "$TEMP_BUILD_DIR"

    info "Downloading libguestfs $LIBGUESTFS_VERSION..."
    local tarball="libguestfs-${LIBGUESTFS_VERSION}.tar.gz"
    local signature="${tarball}.sig"

    # Determine stable branch from version (e.g., 1.58.1 -> 1.58-stable)
    local major_minor=$(echo "$LIBGUESTFS_VERSION" | grep -oP '^\d+\.\d+')
    local stable_branch="${major_minor}-stable"
    local base_url="https://download.libguestfs.org/${stable_branch}"

    # Download tarball (show progress for large file)
    if ! curl -fL --progress-bar -o "$tarball" "${base_url}/${tarball}"; then
        error "Failed to download libguestfs from ${base_url}/${tarball}"
    fi

    # Download signature (quiet for small file)
    info "Downloading GPG signature..."
    if ! curl -fsSL -o "$signature" "${base_url}/${signature}"; then
        warn "Failed to download signature file, skipping verification"
    else
        verify_signature "$tarball"
    fi

    info "Extracting tarball..."
    tar -xzf "$tarball"
    cd "libguestfs-${LIBGUESTFS_VERSION}"

    info "Configuring libguestfs (golang bindings only)..."
    # Configure with minimal features - we only need golang bindings generated
    # Disable appliance, daemon, and most features to speed up build
    if ! ./configure \
        --enable-golang \
        --disable-appliance \
        --disable-daemon \
        --disable-ocaml \
        --disable-perl \
        --disable-python \
        --disable-ruby \
        --disable-java \
        --disable-haskell \
        --disable-php \
        --disable-erlang \
        --disable-lua \
        --disable-gobject 2>&1 | tee /tmp/libguestfs-configure.log | tail -20; then
        error "Configure failed. See /tmp/libguestfs-configure.log for full output."
    fi

    info "Building golang bindings..."
    # Only build the golang directory
    cd golang
    if ! make -j$(nproc) 2>&1 | tee /tmp/libguestfs-build.log | tail -20; then
        error "Build failed. See /tmp/libguestfs-build.log for full output."
    fi

    info "Extracting generated golang bindings..."
    local src_dir="$TEMP_BUILD_DIR/libguestfs-${LIBGUESTFS_VERSION}/golang/src/libguestfs.org/guestfs"

    if [ ! -d "$src_dir" ]; then
        error "Generated bindings not found at $src_dir"
    fi

    # Return to original directory
    cd "$original_dir"

    info "Copying bindings to $VENDOR_DIR..."
    mkdir -p "$VENDOR_DIR"
    cp -r "$src_dir"/* "$VENDOR_DIR/"

    info "Successfully vendored generated bindings"
}

# Add license header to vendored files
add_license_info() {
    info "Adding license information..."

    cat > "$VENDOR_DIR/LICENSE" <<'EOF'
These golang bindings were generated from libguestfs source code.

libguestfs is licensed under LGPL-2.1+
Source: https://libguestfs.org/

The generated golang bindings are derived from the libguestfs API
and are compatible with permissive licenses (Apache-2.0, MIT, etc.).

Generated bindings are provided as-is from libguestfs build process.
EOF

    cat > "$VENDOR_DIR/README.md" <<EOF
# Vendored libguestfs Golang Bindings

These are the official golang bindings for libguestfs, version ${LIBGUESTFS_VERSION}.

## Source

- Generated from: libguestfs ${LIBGUESTFS_VERSION}
- License: LGPL-2.1+ (compatible with permissive licenses)
- Upstream: https://libguestfs.org/

## Why Vendored?

These bindings are vendored to ensure:
1. Consistent build experience across platforms
2. Fast compilation (no binding generation step)
3. Works even when libguestfs source is unavailable

## Runtime Requirements

Applications using these bindings require:
- libguestfs shared libraries (libguestfs.so)
- libguestfs development headers at build time

### Installation

**Ubuntu/Debian:**
\`\`\`bash
sudo apt install libguestfs-dev
\`\`\`

**Arch Linux:**
\`\`\`bash
sudo pacman -S libguestfs
\`\`\`

## Usage

Import the vendored bindings in your Go code using the path where you vendored them.

For example, if vendored to \`vendor/libguestfs.org/guestfs\`:
\`\`\`go
import "yourmodule/vendor/libguestfs.org/guestfs"
\`\`\`

## Updates

To update these bindings to a newer version, re-run the vendoring script
with the desired version.
EOF
}

# Main execution
main() {
    info "Vendoring libguestfs golang bindings"
    info "  Target: $VENDOR_DIR"
    info "  Version: $LIBGUESTFS_VERSION"
    echo ""

    check_existing_bindings

    # Try system bindings first (fast path for Ubuntu/Debian)
    if try_system_bindings; then
        add_license_info
        info ""
        info "Done! Bindings vendored at $VENDOR_DIR"
        info ""
        info "To use in your code, import using your project's module path:"
        info "  import \"yourmodule/$VENDOR_DIR\""
        exit 0
    fi

    # Fall back to building from source (Arch Linux and others)
    warn "System bindings not found, building from source..."
    build_and_extract_bindings
    add_license_info

    info ""
    info "Done! Bindings vendored at $VENDOR_DIR"
    info ""
    info "To use in your code, import using your project's module path:"
    info "  import \"yourmodule/$VENDOR_DIR\""
}

main "$@"
