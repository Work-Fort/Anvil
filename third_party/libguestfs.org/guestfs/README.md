<!-- SPDX-License-Identifier: LGPL-2.1-or-later -->

# Vendored libguestfs Golang Bindings

These are the official golang bindings for libguestfs, version 1.52.0.

## Source

- Generated from: libguestfs 1.52.0
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
```bash
sudo apt install libguestfs-dev
```

**Arch Linux:**
```bash
sudo pacman -S libguestfs
```

## Usage

Import the vendored bindings in your Go code using the path where you vendored them.

For example, if vendored to `vendor/libguestfs.org/guestfs`:
```go
import "yourmodule/vendor/libguestfs.org/guestfs"
```

## Updates

To update these bindings to a newer version, re-run the vendoring script
with the desired version.
