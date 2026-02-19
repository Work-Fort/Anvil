# Anvil

Automated builder for Firecracker-compatible Linux kernels. Checks for new stable kernels every 4 hours, optimized for AWS Firecracker microVMs.

**📖 [Full Documentation](https://Work-Fort.github.io/Anvil)**

## Features

- **Automated Builds Every 4 Hours** - Latest stable kernel from kernel.org
- **Multi-Architecture** - x86_64 and ARM64 (aarch64) support (ARM64 is experimental and untested)
- **Firecracker-Optimized** - Minimal, fast-booting kernel configurations
- **Cryptographically Verified** - PGP + SHA256 verification of all sources
- **GitHub Releases** - Pre-built kernels with checksums and signatures

## Quick Start

### Download Pre-Built Kernels

Visit the [Releases](../../releases) page to download pre-built kernels:

```bash
# Download kernel (replace VERSION with actual version, e.g., 6.18.9)
wget https://github.com/Work-Fort/Anvil/releases/latest/download/vmlinux-VERSION-x86_64.xz
wget https://github.com/Work-Fort/Anvil/releases/latest/download/SHA256SUMS

# Decompress and verify
xz -d vmlinux-VERSION-x86_64.xz
sha256sum -c SHA256SUMS --ignore-missing

# Use with Firecracker
firecracker --kernel-path vmlinux-VERSION-x86_64 ...
```

### Request a Kernel Build

Don't see the version you need? [Request a build](../../issues/new?template=build-request.yml) and we'll build it automatically.

## Verifying Releases

**All kernel releases are PGP-signed.** We strongly recommend verifying signatures:

```bash
# 1. Import Anvil release signing key (first time only)
curl -s https://raw.githubusercontent.com/Work-Fort/Anvil/master/keys/signing-key.asc | gpg --import

# Verify the key fingerprint matches (see below)
gpg --fingerprint me@kazatron.com

# 2. Download kernel, checksums, and signature
wget https://github.com/Work-Fort/Anvil/releases/latest/download/vmlinux-VERSION-x86_64.xz
wget https://github.com/Work-Fort/Anvil/releases/latest/download/SHA256SUMS
wget https://github.com/Work-Fort/Anvil/releases/latest/download/SHA256SUMS.asc

# 3. Verify PGP signature on checksums
gpg --verify SHA256SUMS.asc SHA256SUMS
# Should show: "Good signature from Anvil Release Signing"

# 4. Verify download immediately
sha256sum -c SHA256SUMS --ignore-missing
# Verifies: vmlinux-VERSION-x86_64.xz

# 5. Decompress kernel
xz -d vmlinux-VERSION-x86_64.xz

# 6. Verify kernel binary before use
sha256sum -c SHA256SUMS --ignore-missing
# Verifies: vmlinux-VERSION-x86_64
```

**Anvil Release Signing Key:**
```
Key ID: 7E7E22A24A116FBD
Fingerprint: F060 03AB F17F FF1D 4F24  F875 7E7E 22A2 4A11 6FBD
Email: me@kazatron.com
Expires: 2026-02-17 (Alpha key - 10 day expiry for testing rotation)
```

**Chain of Trust:**
1. Kernel sources verified with kernel.org autosigner PGP signature
2. Kernel sources verified with SHA256 checksums from kernel.org
3. Built kernels signed with Anvil release key
4. Users verify with Anvil public key

This ensures sources came from kernel.org and builds weren't tampered with.

## Building Locally

Install [Task](https://taskfile.dev) and build dependencies:

```bash
# Install Task
sh -c "$(curl -fsSL https://taskfile.dev/install.sh)" -- -d -b ~/.local/bin

# Install build dependencies
task dev:install-deps

# Get latest kernel (download or build)
task kernel:get

# Build specific version
task kernel:get KERNEL_VERSION=6.1

# Test kernel in Firecracker VM (requires KVM)
task firecracker:test-kernel

# See all available tasks
task --list
```

For detailed build instructions, security verification levels, and advanced usage, see the [full documentation](https://Work-Fort.github.io/Anvil).

## Documentation

- **[Getting Started](https://Work-Fort.github.io/Anvil/getting-started.html)** - Downloading and using kernels
- **[Building Locally](https://Work-Fort.github.io/Anvil/building.html)** - Build from source
- **[Security](https://Work-Fort.github.io/Anvil/security.html)** - Verification and threat model
- **[Configuration](https://Work-Fort.github.io/Anvil/configuration.html)** - Kernel configuration details
- **[CI/CD](https://Work-Fort.github.io/Anvil/ci-cd.html)** - GitHub Actions workflow
- **[Maintainer Guide](https://Work-Fort.github.io/Anvil/maintainer.html)** - Setting up signing keys

## Project Structure

```
.github/workflows/     # GitHub Actions workflows
configs/               # Firecracker kernel configurations
keys/                  # PGP signing keys
scripts/               # Build, signing, and testing scripts
tasks/                 # Task runner configurations
```

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

See the [full documentation](https://Work-Fort.github.io/Anvil) for development guidelines.

## License

Apache License 2.0. See [LICENSE.md](LICENSE.md) for details.

The Linux kernel itself is licensed under GPL-2.0.

### Third-Party Dependencies

This project uses [libguestfs](https://libguestfs.org/), which is licensed under
GNU LGPL 2.1. Anvil dynamically links to libguestfs at runtime for rootfs
creation functionality. Users can replace the libguestfs library without recompiling
anvil.

All other Go dependencies use permissive licenses (MIT, Apache 2.0, BSD, MPL 2.0)
that are compatible with Apache 2.0.

See [NOTICE.md](NOTICE.md) for full third-party notices and license information.

## References

- [Full Documentation](https://Work-Fort.github.io/Anvil)
- [Firecracker Documentation](https://github.com/firecracker-microvm/firecracker)
- [Kernel.org](https://kernel.org)
