# libguestfs Namespace

Independent, reusable utilities for vendoring official libguestfs golang bindings (LGPL-2.1+) into Go projects.

## Why This Exists

Community golang bindings for libguestfs (`github.com/jonathongardner/guestfs`) are GPL-2.0 licensed, which is incompatible with Apache-2.0 and other permissive licenses. The official libguestfs project provides LGPL-2.1+ bindings that are compatible.

## Project Structure

```
scripts/libguestfs/
├── vendor-bindings.sh    # Core vendoring script (project-agnostic)
└── README.md            # This file

tasks/libguestfs/
└── Taskfile.dist.yaml   # Task wrapper for convenience
```

## Quick Start

### Using with Task (Recommended)

```bash
# Vendor with auto-detected system version
task libguestfs:vendor

# Specific version
task libguestfs:vendor VERSION=1.58.0

# Custom output directory (with auto-detected version)
task libguestfs:vendor VENDOR_DIR=internal/guestfs

# Show help
task libguestfs:help
```

### Using Script Directly

```bash
# Vendor with auto-detected system version
./scripts/libguestfs/vendor-bindings.sh

# Show help
./scripts/libguestfs/vendor-bindings.sh --help

# Specific version
./scripts/libguestfs/vendor-bindings.sh --version 1.58.0

# Custom output and version
./scripts/libguestfs/vendor-bindings.sh \
  --output internal/guestfs \
  --version 1.58.0
```

## How It Works

1. **Auto-detect version** (if not specified)
   - First tries to detect system libguestfs version using pkg-config
   - Falls back to distro-specific methods (dpkg, pacman) if pkg-config unavailable
   - If system version not found, queries https://download.libguestfs.org/ for latest stable
   - Falls back to 1.52.0 if all detection fails (Ubuntu 24.04 LTS version for cross-distro compatibility)
   - This ensures vendored bindings match your installed libguestfs library

2. **Try system packages first** (Ubuntu/Debian)
   - Checks for `golang-guestfs-dev` package
   - Fast path: copies pre-built bindings from `/usr/share/gocode`

3. **Verify downloads** (when building from source)
   - Downloads PGP signature (`.sig` file)
   - Imports libguestfs GPG key if needed
   - Verifies signature using GPG
   - Fails securely if signature is invalid
   - Can be skipped with `--skip-verify` (not recommended)

4. **Build from source** (Arch Linux, others)
   - Downloads verified libguestfs source
   - Configures with minimal features (golang only)
   - Builds bindings in temporary directory
   - Extracts generated `.go` files

5. **Vendors the bindings**
   - Copies to target directory
   - Adds LICENSE and README files
   - Ready to import in your Go code

## Security

The script verifies downloads using GPG signatures when building from source:
- Downloads both tarball and `.sig` signature file
- Imports Richard W.M. Jones's GPG key (libguestfs maintainer)
- Verifies signature before extraction
- Fails if signature is invalid

System packages (Ubuntu/Debian) are trusted through your package manager's verification.

## Requirements

### Runtime Requirements

Applications using these bindings need:
- libguestfs shared libraries (`libguestfs.so`)
- libguestfs development headers (build time only)

**Ubuntu/Debian:**
```bash
sudo apt install libguestfs-dev
```

**Arch Linux:**
```bash
sudo pacman -S libguestfs
```

### Build Requirements (only for source builds)

If system packages aren't available, building from source requires:
- `curl`, `tar`, `gcc`, `make`, `pkg-config`, `go`

**Ubuntu/Debian:**
```bash
sudo apt install build-essential curl tar gcc make pkg-config golang
```

**Arch Linux:**
```bash
sudo pacman -S base-devel curl tar gcc make pkg-config go
```

## Extracting to Other Projects

This namespace is designed to be project-agnostic. To use in another project:

### 1. Copy Files

```bash
# In your project root
mkdir -p scripts/libguestfs tasks/libguestfs
cp -r scripts/libguestfs/* scripts/libguestfs/
cp -r tasks/libguestfs/* tasks/libguestfs/
```

### 2. Add to Root Taskfile

In your `Taskfile.dist.yaml`:

```yaml
includes:
  libguestfs:
    taskfile: tasks/libguestfs
    dir: .
```

### 3. Vendor Bindings

```bash
task libguestfs:vendor VENDOR_DIR=your/preferred/path
```

### 4. Use in Code

```go
import "yourmodule/your/preferred/path"
```

## Script Options

The core script (`vendor-bindings.sh`) accepts:

- `--output DIR` - Target directory (default: `vendor/libguestfs.org/guestfs`)
- `--version VERSION` - libguestfs version (default: `1.56.2`)
- `--help` - Show usage

Environment variables:
- `VENDOR_DIR` - Alternative to `--output`
- `VERSION` - Alternative to `--version`

## License

Scripts: Apache-2.0 (from anvil project)
Vendored bindings: LGPL-2.1+ (from libguestfs project)

The LGPL-2.1+ license on the bindings is compatible with permissive licenses like Apache-2.0, MIT, BSD, etc.

## Links

- libguestfs upstream: https://libguestfs.org/
- Official API docs: https://libguestfs.org/guestfs-golang.3.html
- Ubuntu package: https://packages.ubuntu.com/golang-guestfs-dev
