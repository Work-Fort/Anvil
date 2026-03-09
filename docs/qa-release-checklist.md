# Anvil Release QA Checklist

> Reusable end-to-end QA process for major releases. Run through each section in order — later sections depend on earlier ones.

## Prerequisites

- [ ] `mise ci` passes (lint, vet, staticcheck, tests)
- [ ] MCP server binary rebuilt (`mise run build`) and restarted
- [ ] Clean working tree (`git status` shows no unexpected changes)

---

## 1. Configuration System

Test in a temp repo so you never modify production config:

```bash
export CONFIGDIR=$(mktemp -d)
cd "$CONFIGDIR"
ANVIL_SIGNING_PASSWORD=test_password anvil init \
  --key-name "QA Config Test" --key-email "qa@test.com" --use-tui=false
```

### CLI (run from $CONFIGDIR)

- [ ] `anvil config list` — shows all config values with sources
- [ ] `anvil config get signing.key.name` — returns correct value and source
- [ ] `anvil config set log-level info` — sets value, verify with `config get`
- [ ] `anvil config set log-level debug` — restore default
- [ ] `anvil config unset log-level` — removes override, falls back to default
- [ ] `anvil config schema` — outputs valid JSON Schema

### MCP (switch to $CONFIGDIR first)

- [ ] `set_repo_root` path=`$CONFIGDIR` — switches to test repo
- [ ] `config_list` — returns array of {key, value, source}
- [ ] `config_get` key=`signing.key.name` — matches CLI output
- [ ] `config_set` key=`log-level` value=`info` — sets value
- [ ] `config_get` key=`log-level` — confirms change
- [ ] `config_set` key=`log-level` value=`debug` — restore
- [ ] `config_get_paths` — returns all resolved directory paths
- [ ] `get_context` — returns mode, paths, cwd

### Cross-check

- [ ] Value set via CLI is visible via MCP `config_get`
- [ ] Value set via MCP is visible via `anvil config get`

Clean up:

```bash
rm -rf "$CONFIGDIR"
```

---

## 2. Kernel Version Discovery

### CLI

- [ ] `anvil kernel versions` — lists available versions from kernel.org
- [ ] `anvil kernel version-check <latest>` — exits 0, reports buildable
- [ ] `anvil kernel version-check 1.0.0` — exits 1, reports not buildable
- [ ] `anvil kernel list` — lists installed kernels (may be empty)

### MCP

- [ ] `kernel_versions` — returns latest version
- [ ] `kernel_version_check` version=`<latest>` — valid=true
- [ ] `kernel_version_check` version=`1.0.0` — valid=false
- [ ] `kernel_list` — matches CLI output
- [ ] `check_build_tools` — reports ready=true (all tools found)

---

## 3. Kernel Build

Pick a recent stable version (e.g. the latest from step 2).

### x86_64 Build (MCP async)

- [ ] `kernel_build` version=`<version>` arch=`x86_64` — returns build_id
- [ ] `kernel_build_status` build_id=`<id>` — shows running, phase, progress
- [ ] `kernel_build_log` build_id=`<id>` — returns recent output lines
- [ ] `kernel_build_wait` build_id=`<id>` — blocks until complete, returns stats
- [ ] Build completes successfully (status=completed)

### aarch64 Build (MCP async, cross-compile)

- [ ] `check_build_tools` arch=`aarch64` — reports ready=true (cross-compiler found)
- [ ] `kernel_build` version=`<version>` arch=`aarch64` — returns build_id
- [ ] `kernel_build_status` build_id=`<id>` — shows running, phase, progress
- [ ] `kernel_build_log` build_id=`<id>` — returns recent output lines
- [ ] `kernel_build_wait` build_id=`<id>` — blocks until complete, returns stats
- [ ] Build completes successfully (status=completed)

### CLI (verify result)

- [ ] `anvil kernel list` — shows the built version (if auto-installed)

### MCP (install both architectures)

- [ ] `kernel_install` version=`<version>` arch=`x86_64` set_default=true — installs from cache
- [ ] `kernel_info` version=`<version>` — shows files, is_default=true
- [ ] `kernel_install` version=`<version>` arch=`aarch64` — installs aarch64 from cache
- [ ] `kernel_list` — confirms both versions listed

### Kernel Config Tools

- [ ] `kernel_config_list` config_file=`<project>/configs/microvm-kernel-x86_64.config` — lists options
- [ ] `kernel_config_get` config_file=`<config>` option=`CONFIG_VIRTIO` — returns value
- [ ] `kernel_config_list` config_file=`<config>` filter=`VIRTIO` — filtered results

---

## 4. Signing

### CLI

- [ ] `anvil signing list` — shows current key(s)
- [ ] `anvil signing check-expiry` — reports all_valid or warns
- [ ] `anvil signing check-expiry --days 365` — broader check window

### MCP

- [ ] `signing_list` — returns keys array with key_id, fingerprint, name, email
- [ ] `signing_check_expiry` — all_valid=true
- [ ] `signing_check_expiry` days=365 — broader window check

### Cross-check

- [ ] Key info from CLI matches MCP `signing_list` output

### Sign & Verify

Test in an isolated temp repo so you never touch production keys:

```bash
export TESTDIR=$(mktemp -d)
cd "$TESTDIR"
ANVIL_SIGNING_PASSWORD=test_password anvil init \
  --key-name "QA Test" --key-email "qa@test.com" --use-tui=false
```

Create test artifacts to sign:

```bash
mkdir -p "$TESTDIR/artifacts"
echo "hello" > "$TESTDIR/artifacts/test.bin"
sha256sum "$TESTDIR/artifacts/test.bin" > "$TESTDIR/artifacts/SHA256SUMS"
```

Run sign & verify from the test repo directory:

- [ ] CLI `cd "$TESTDIR" && ANVIL_SIGNING_PASSWORD=test_password anvil signing sign artifacts/` — exits 0
- [ ] CLI `cd "$TESTDIR" && anvil signing verify artifacts/` — exits 0
- [ ] MCP: `set_repo_root` path=`$TESTDIR` — switches context to test repo
- [ ] MCP: `signing_sign` path=`$TESTDIR/artifacts` — status=signed (requires `ANVIL_SIGNING_PASSWORD=test_password` in MCP server env)
- [ ] MCP: `signing_verify` path=`$TESTDIR/artifacts` — verified=true
- [ ] MCP: `set_repo_root` path=`<original_repo>` — switch back

Clean up:

```bash
rm -rf "$TESTDIR"
```

---

## 5. Firecracker

### Version Management

#### CLI

- [ ] `anvil firecracker versions` — lists available versions
- [ ] `anvil firecracker list` — lists installed versions

#### MCP

- [ ] `firecracker_versions` — returns available versions
- [ ] `firecracker_list` — returns installed versions, matches CLI

### Download & Set Default

- [ ] `firecracker_get` — downloads latest (or specify version)
- [ ] `firecracker_list` — confirms installed
- [ ] `firecracker_set_default` version=`<version>` — sets default
- [ ] `firecracker_list` — confirms is_default=true

### Rootfs Creation

- [ ] `firecracker_create_rootfs` — creates Alpine rootfs (default path)
- [ ] Verify file exists at reported path

### End-to-End VM Test

- [ ] `firecracker_test` — boots VM, tests vsock, reports success
- [ ] Check: boot_time and ping_round_trip are reasonable
- [ ] Check: VM cleaned up (no leftover processes)

### CLI Equivalent

- [ ] `anvil firecracker test` — same test via CLI, passes

---

## 6. Archive & Sign Release (repo mode only)

If in a repo with `anvil.yaml`:

- [ ] `set_repo_root` path=`<repo>` — switches to repo mode
- [ ] `archive_kernel` version=`<version>` arch=`x86_64` — archives x86_64 kernel
- [ ] `archive_kernel` version=`<version>` arch=`aarch64` — archives aarch64 kernel
- [ ] `archive_list` — shows both archived kernels
- [ ] `archive_get` version=`<version>` arch=`x86_64` — returns details
- [ ] `archive_get` version=`<version>` arch=`aarch64` — returns details

### Sign & Verify archived kernels

Test signing in an isolated temp repo so you never touch production keys:

```bash
export SIGNDIR=$(mktemp -d)
cd "$SIGNDIR"
ANVIL_SIGNING_PASSWORD=test_password anvil init \
  --key-name "QA Sign Test" --key-email "qa-sign@test.com" --use-tui=false
```

Copy archived artifacts into the test repo for signing:

```bash
cp -r <repo>/archive "$SIGNDIR/archive"
```

Run sign & verify from the test repo directory:

- [ ] CLI `cd "$SIGNDIR" && ANVIL_SIGNING_PASSWORD=test_password anvil signing sign archive/x86_64/<version>` — exits 0
- [ ] CLI `cd "$SIGNDIR" && anvil signing verify archive/x86_64/<version>` — exits 0
- [ ] CLI `cd "$SIGNDIR" && ANVIL_SIGNING_PASSWORD=test_password anvil signing sign archive/aarch64/<version>` — exits 0
- [ ] CLI `cd "$SIGNDIR" && anvil signing verify archive/aarch64/<version>` — exits 0
- [ ] MCP: `set_repo_root` path=`$SIGNDIR` — switches context to test repo
- [ ] MCP: `signing_sign` path=`$SIGNDIR/archive/x86_64/<version>` — status=signed
- [ ] MCP: `signing_verify` path=`$SIGNDIR/archive/x86_64/<version>` — verified=true
- [ ] MCP: `signing_sign` path=`$SIGNDIR/archive/aarch64/<version>` — status=signed
- [ ] MCP: `signing_verify` path=`$SIGNDIR/archive/aarch64/<version>` — verified=true
- [ ] MCP: `set_repo_root` path=`<original_repo>` — switch back

Clean up:

```bash
rm -rf "$SIGNDIR"
```

---

## 7. Clean Operations

> Run these last — they remove data created in earlier steps.

### MCP (non-default cleanup)

- [ ] `clean_build` all=false — cleans build output, keeps source cache
- [ ] `clean_rootfs` — removes .ext4 files, returns removed list

### MCP (verify default protection)

If multiple kernel/firecracker versions installed:

- [ ] `clean_kernel` all=false — removes only non-default kernels
- [ ] `kernel_list` — confirms default kernel still present
- [ ] `clean_firecracker` all=false — removes only non-default versions
- [ ] `firecracker_list` — confirms default still present

### CLI equivalents

- [ ] `anvil clean build` — cleans build cache
- [ ] `anvil clean rootfs` — removes rootfs images

---

## 8. Init Command (fresh repo test)

Test in a temporary directory (use `ANVIL_SIGNING_PASSWORD` so the key is encrypted):

```bash
INITDIR=$(mktemp -d)
cd "$INITDIR"
ANVIL_SIGNING_PASSWORD=test_password anvil init \
  --key-name "Test" --key-email "test@test.com" --use-tui=false
```

- [ ] Creates `anvil.yaml`
- [ ] Creates `configs/kernel-x86_64.config`
- [ ] Creates `configs/kernel-aarch64.config`
- [ ] Creates `.gitignore`
- [ ] Generates signing key

Clean up temp directory after.

---

## 9. MCP Context Switching

- [ ] `get_context` — shows current mode
- [ ] `set_repo_root` path=`<repo>` — switches to repo mode
- [ ] `get_context` — mode=repo, paths reflect repo
- [ ] `set_user_mode` — switches back to user mode
- [ ] `get_context` — mode=user, paths reflect XDG

---

## 10. Backwards Compatibility

- [ ] `anvil build-kernel --help` — works (hidden alias)
- [ ] `anvil clean build-kernel --help` — works (alias for `clean build`)

---

## Sign-Off

| Area | Passed | Notes |
|------|--------|-------|
| Configuration | | |
| Kernel Discovery | | |
| Kernel Build | | |
| Signing | | |
| Firecracker | | |
| Archive | | |
| Clean Operations | | |
| Init | | |
| MCP Context | | |
| Backwards Compat | | |

**Tested by:** _______________
**Date:** _______________
**Version/Commit:** _______________
