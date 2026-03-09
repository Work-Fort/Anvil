# Remaining Bugs

## Bug #14: `anvil init` kernel config templates are empty stubs

**Status:** Known limitation (documented in QA)

`anvil init` creates empty kernel config templates (`configs/kernel-x86_64.config`, `configs/kernel-aarch64.config`) with just comments and no CONFIG_* options. When a kernel is built using these configs, `make olddefconfig` generates a minimal default config without Firecracker-required options (CONFIG_VIRTIO_MMIO, CONFIG_VIRTIO_BLK, CONFIG_EXT4_FS, etc.), producing a kernel that can't boot in Firecracker.

**Workaround:** Pass the anvil project's `configs/microvm-kernel-*.config` files explicitly via `config_file` parameter when building kernels for Firecracker use.

**Future fix:** Either ship full Firecracker-compatible configs in `anvil init`, or document that users must supply their own kernel configs.
