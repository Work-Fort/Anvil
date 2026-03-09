#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#MISE description="Build anvil CLI"
#MISE depends=["build:vsock-server"]
set -euo pipefail

BUILD_DIR=build
GIT_SHORT_SHA=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
VERSION="${VERSION:-dev-$GIT_SHORT_SHA}"
DISABLE_UPDATE="${DISABLE_UPDATE:-false}"

mkdir -p "$BUILD_DIR"

echo "Copying vsock-server-standalone for embedding..."
cp "$BUILD_DIR/vsock-server-standalone" pkg/firecracker/embedded/vsock-server-standalone

echo "Building anvil CLI with embedded vsock-server..."
go build -mod=mod \
  -ldflags "-X github.com/Work-Fort/Anvil/cmd.Version=$VERSION -X github.com/Work-Fort/Anvil/cmd.DisableUpdate=$DISABLE_UPDATE" \
  -o "$BUILD_DIR/anvil"
echo "✓ Built $BUILD_DIR/anvil"
