# Bug: Tar Extraction Silently Drops Symlinks

## Summary

`pkg/util/compression.go:ExtractTarXzWithProgress()` only handles `tar.TypeDir`
and `tar.TypeReg` entries. All other entry types — including symlinks
(`tar.TypeSymlink`) and hardlinks (`tar.TypeLink`) — are silently skipped with a
debug log message.

This causes missing files in the extracted kernel source tree, breaking the
aarch64 kernel build on kernel 6.19+.

## Impact

**Severity:** Build-breaking for aarch64 cross-compilation on kernel >= 6.19

The kernel tarball contains:

```
lrwxrwxrwx linux-6.19.6/arch/arm64/tools/syscall_64.tbl -> ../../../scripts/syscall.tbl
```

This symlink is required for generating `unistd_64.h` during `make prepare
ARCH=arm64`. Without it, the build fails:

```
make[2]: *** No rule to make target 'arch/arm64/include/generated/uapi/asm/unistd_64.h', needed by 'all'.
```

The x86_64 build is unaffected because x86_64 has its syscall table as a regular
file, not a symlink.

## Root Cause

`pkg/util/compression.go` lines 310-337:

```go
switch header.Typeflag {
case tar.TypeDir:
    // Create directory
    ...
case tar.TypeReg:
    // Create file
    ...
default:
    log.Debugf("Skipping unsupported file type: %s (%c)", header.Name, header.Typeflag)
}
```

Missing cases:
- `tar.TypeSymlink` — should `os.Symlink(header.Linkname, target)`
- `tar.TypeLink` — should `os.Link(resolvedPath, target)`

The same bug exists in both `ExtractTarXzWithProgress` and `ExtractTarGz`.

## Other Dropped Symlinks

Full list of symlinks in the 6.19.6 kernel tarball relevant to arm64 builds:

| Path | Target | Impact |
|------|--------|--------|
| `arch/arm64/tools/syscall_64.tbl` | `../../../scripts/syscall.tbl` | **Build-breaking** — required for `make prepare` |
| `scripts/dtc/include-prefixes/arm64` | `../../../arch/arm64/boot/dts` | DTS compilation (not needed for vmlinux) |
| `scripts/kernel-doc` | `kernel-doc.py` | Documentation generation |
| `scripts/dummy-tools/nm`, `objcopy` | `ld` | Dummy tools for config-only builds |

## Fix

Add `tar.TypeSymlink` and `tar.TypeLink` handling to both `ExtractTarXzWithProgress`
and `ExtractTarGz` in `pkg/util/compression.go`.

For symlinks, validate that the resolved target stays within the extraction
directory (path traversal protection, same as for regular files).

## Workaround

Manually create the symlink after extraction:

```bash
ln -sf ../../../scripts/syscall.tbl \
  ~/.cache/anvil/build-kernel/build/linux-6.19.6/arch/arm64/tools/syscall_64.tbl
```

Note: This must be done after extraction and before the build, since
`--verification-level high` (default) deletes and re-downloads the source.

## Related: Missing CROSS_COMPILE in aarch64 build

Also fixed in the same session: `pkg/kernel/build.go` `make prepare` and
`make Image` for aarch64 were missing `CROSS_COMPILE=aarch64-linux-gnu-`,
causing the `make prepare` step to fail when trying to compile arm64 objects
with the host x86_64 gcc.

## Discovered

2026-03-08 during aarch64 Kata dual-hypervisor kernel config work. The symlink
bug existed since initial implementation but was latent because the x86_64
kernel tarball doesn't use symlinks for build-critical files. The
CROSS_COMPILE bug was latent because `make prepare` was only added for
arm64 kernels >= 6.11, and the previous aarch64 builds either didn't reach
this step or weren't cross-compiled.
