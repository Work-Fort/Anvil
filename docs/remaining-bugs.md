# Remaining Bugs

## Bug #14: `anvil init` kernel config templates are empty stubs

**Status:** Known limitation (documented in QA)

`anvil init` creates empty kernel config templates (`configs/kernel-x86_64.config`, `configs/kernel-aarch64.config`) with just comments and no CONFIG_* options. When a kernel is built using these configs, `make olddefconfig` generates a minimal default config without Firecracker-required options (CONFIG_VIRTIO_MMIO, CONFIG_VIRTIO_BLK, CONFIG_EXT4_FS, etc.), producing a kernel that can't boot in Firecracker.

**Workaround:** Pass the anvil project's `configs/microvm-kernel-*.config` files explicitly via `config_file` parameter when building kernels for Firecracker use.

**Future fix:** Either ship full Firecracker-compatible configs in `anvil init`, or document that users must supply their own kernel configs.

## Bug #15: CLI build fails on Ubuntu 26.04 runners (libguestfs 1.58 header vs vendored 1.52 bindings)

**Status:** Fixed — bindings re-vendored at 1.58.1 and all workflows pinned to `ubuntu-26.04` / `ubuntu-26.04-arm`

GitHub's `ubuntu-latest` label moved to Ubuntu 26.04 (Oct 2026), which ships `libguestfs-dev` 1.58.1. The vendored Go bindings in `third_party/libguestfs.org/guestfs` were generated from libguestfs 1.52.0. In upstream libguestfs (1.54+), the `guestfs_application2` struct field `app2_spare1` was renamed to `app2_class`, so cgo fails:

```
third_party/libguestfs.org/guestfs/guestfs.go:301:35: c.app2_spare1 undefined (type *_Ctype_struct_guestfs_application2 has no field or method app2_spare1)
```

First seen in the scheduled "Check Signing Key Expiration" run 40198016923 (2026-10-25). The struct layout is unchanged (same `char *` slot), so this is a compile-time break only, not an ABI break.

**Fix:** Re-vendored `guestfs.go` from Ubuntu 26.04's `golang-guestfs-dev` 1.58.1 package (the test files and `go.mod` are unchanged upstream between 1.52.0 and 1.58.1) and bumped the default vendor version to 1.58.1. Workflows now pin `ubuntu-26.04` / `ubuntu-26.04-arm` instead of floating on `ubuntu-latest`.

**Compatibility note:** Building from source now requires libguestfs >= 1.54 headers (e.g. Ubuntu 26.04, current Arch). Release binaries are unaffected: a binary built on 26.04 needs only GLIBC_2.34 and is lazily bound, so it runs against libguestfs 1.52 on Ubuntu 24.04 (verified with `anvil version`). It references 8 symbols absent from 1.52 (`guestfs_command_out`, `guestfs_sh_out`, etc.) that Anvil never calls.
