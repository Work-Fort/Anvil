# CLI Reference

## Global Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-l, --log-level` | `debug` | Log level: `disabled`, `debug`, `info`, `warn`, `error` |
| `--use-tui` | `true` | Enable terminal UI mode |

---

## anvil build-kernel

Build a Firecracker-compatible kernel from source. Downloads kernel source from kernel.org, verifies integrity, and builds with Firecracker-optimized configuration.

**Alias:** `build`

```
anvil build-kernel [version] [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-a, --arch` | host arch | Target architecture: `x86_64`, `aarch64`, or `all` |
| `-c, --config` | | Custom kernel config file |
| `-f, --force-rebuild` | `false` | Force rebuild even if cached build exists |
| `-q, --verification-level` | `high` | Verification level: `high`, `medium`, `disabled` |
| `-v, --version` | latest | Kernel version to build |

**Examples:**

```bash
# Build latest stable kernel
anvil build-kernel

# Build a specific version
anvil build-kernel --version 6.12.0

# Build for aarch64 (experimental)
anvil build-kernel --arch aarch64 --version 6.12.0

# Build with a custom config
anvil build-kernel --config ./my-kernel.config
```

---

## anvil kernel

Manage locally installed Firecracker kernel binaries.

### anvil kernel get

Download a kernel from GitHub releases, or build from source if unavailable.

**Alias:** `download`

```
anvil kernel get [version]
```

```bash
anvil kernel get          # Get latest
anvil kernel get 6.12.0   # Get specific version
```

### anvil kernel versions

Show available kernel versions from GitHub releases.

```
anvil kernel versions
```

### anvil kernel list

List locally installed kernel versions.

```
anvil kernel list
```

### anvil kernel set

Set a kernel version as the default.

**Alias:** `default`

```
anvil kernel set [version]
```

### anvil kernel remove

Remove a locally installed kernel version.

```
anvil kernel remove [version]
```

---

## anvil firecracker

Manage Firecracker binaries and run VMs.

### anvil firecracker get

Download a Firecracker binary from GitHub releases.

**Alias:** `download`

```
anvil firecracker get [version]
```

### anvil firecracker versions

Show available Firecracker versions.

```
anvil firecracker versions
```

### anvil firecracker list

List locally installed Firecracker versions.

```
anvil firecracker list
```

### anvil firecracker set

Set a Firecracker version as the default.

```
anvil firecracker set [version]
```

### anvil firecracker remove

Remove a locally installed Firecracker version.

```
anvil firecracker remove [version]
```

### anvil firecracker create-rootfs

Create an Alpine Linux-based ext4 rootfs image for Firecracker VMs.

**Alias:** `mkrootfs`

```
anvil firecracker create-rootfs [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--alpine-version` | `3.23` | Alpine Linux version (major.minor) |
| `--alpine-patch` | `3` | Alpine Linux patch version |
| `--binary-path` | current binary | Path to binary to inject |
| `--binary-dest` | `/usr/bin/anvil` | Destination path in rootfs |
| `--inject-binary` | `false` | Inject binary into rootfs |
| `-f, --force` | `false` | Overwrite existing file |
| `-o, --output` | `~/.local/share/anvil/alpine-rootfs.ext4` | Output file path |
| `-s, --size` | `512` | Size in MB |

### anvil firecracker test

Run an end-to-end integration test of Firecracker with vsock.

```
anvil firecracker test [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--kernel-version` | default kernel | Kernel version to test |
| `--rootfs` | `~/.local/share/anvil/alpine-rootfs.ext4` | Path to rootfs image |
| `--boot-timeout` | `10s` | Timeout for VM boot |
| `--ping-timeout` | `10s` | Timeout for vsock ping |

---

## anvil config

Manage anvil configuration. Configuration precedence (highest to lowest):

1. Environment variables (`ANVIL_*`)
2. Local config (`./anvil.yaml`)
3. User config (`~/.config/anvil/config.yaml`)
4. Defaults

By default, operates on local config. Use `--global` to operate on user config.

### anvil config list

List all configuration values.

```
anvil config list
```

### anvil config get

Get a configuration value.

```
anvil config get <key>
```

### anvil config set

Set a configuration value.

```
anvil config set <key> <value>
```

### anvil config unset

Remove a configuration value.

```
anvil config unset <key>
```

### anvil config schema

Export the configuration schema.

```
anvil config schema
```

---

## anvil signing

Manage PGP signing keys for artifact signing.

### anvil signing generate

Generate a new PGP signing key.

```
anvil signing generate
```

### anvil signing list

List all signing keys.

```
anvil signing list
```

### anvil signing sign

Sign release artifacts.

```
anvil signing sign
```

### anvil signing verify

Verify release artifact signatures.

```
anvil signing verify
```

### anvil signing export

Export an encrypted backup of the signing key.

```
anvil signing export
```

### anvil signing import

Import a signing key from an encrypted backup.

```
anvil signing import
```

### anvil signing import-key

Import a public key.

```
anvil signing import-key
```

### anvil signing rotate

Rotate the signing key.

```
anvil signing rotate
```

### anvil signing check-expiry

Check if signing keys are expiring soon.

```
anvil signing check-expiry
```

### anvil signing remove

Remove a signing key.

```
anvil signing remove
```

---

## anvil init

Initialize the current directory as a kernel release/signing repository. In interactive mode (default), launches a step-by-step wizard.

```
anvil init [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--key-name` | | Signing key name (required in non-interactive mode) |
| `--key-email` | | Signing key email (required in non-interactive mode) |
| `--key-expiry` | `1y` | Key expiry duration (`0`=never, `1y`, `2y`, `5y`) |
| `--key-format` | `armored` | Private key format: `armored`, `binary` |
| `--history-format` | `armored` | Public key history format: `armored`, `binary` |
| `--archive-location` | `archive` | Local archive directory (relative path inside repo) |

Non-interactive mode reads the key encryption password from `ANVIL_SIGNING_PASSWORD` or stdin.

---

## anvil clean

Clean cached data.

### anvil clean build-kernel

Clean kernel source and build artifacts.

### anvil clean kernel

Clean installed kernel data.

### anvil clean firecracker

Clean Firecracker data.

### anvil clean rootfs

Clean rootfs images.

---

## anvil vsock

Manage vsock communication between host and Firecracker VMs.

### anvil vsock server

Start a vsock JSON-RPC server inside a VM.

```
anvil vsock server
```

### anvil vsock client

Send a ping to a vsock server.

```
anvil vsock client
```

---

## anvil update

Update the anvil binary to the latest version from GitHub releases. Verifies PGP signature and SHA256 checksum before replacing the binary.

```
anvil update
```

---

## anvil version

Display the current version of anvil.

```
anvil version
```
