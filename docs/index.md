# Anvil

Automated builder and manager for Firecracker-compatible Linux kernels.

Downloads, builds, and manages kernel binaries optimized for [AWS Firecracker](https://github.com/firecracker-microvm/firecracker) microVMs.

---

## Quick Start

### Use a pre-built kernel

```bash
# See available versions
anvil kernel versions

# Download the latest kernel
anvil kernel get

# Download a specific version
anvil kernel get 6.12.0
```

### Build from source

```bash
# Build the latest stable kernel
anvil build-kernel

# Build a specific version
anvil build-kernel --version 6.12.0
```

### Set up a Firecracker environment

```bash
# Get the Firecracker binary
anvil firecracker get

# Create a rootfs
anvil firecracker create-rootfs

# Test everything works
anvil firecracker test
```

---

## Install

Download the latest release from [GitHub Releases](https://github.com/Work-Fort/Anvil/releases) and place the binary in your `$PATH`.

```bash
# Verify the binary (recommended)
gpg --verify anvil.asc anvil
sha256sum -c SHA256SUMS --ignore-missing
```

Or self-update if you already have anvil installed:

```bash
anvil update
```

---

## Documentation

- [CLI Reference](usage.md) — all commands and flags
- [Kernel Configs](kernel-configs.md) — included kernel configurations
