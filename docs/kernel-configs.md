# Kernel Configs

Anvil includes two Firecracker-optimized kernel configurations in the `configs/` directory:

| File | Architecture |
|------|--------------|
| `microvm-kernel-x86_64.config` | x86_64 |
| `microvm-kernel-aarch64.config` | aarch64 (ARM64) |

These configs are based on Amazon Linux 2023's microVM kernel configuration. They enable the minimal set of features needed for Firecracker while keeping boot times fast and the kernel image small.

Key characteristics:

- No modules — everything needed is built in
- virtio-net, virtio-block, virtio-vsock enabled
- No GUI, sound, or USB support
- Optimized for fast boot and small footprint

## Using a Custom Config

To build with your own config:

```bash
anvil build-kernel --config ./my-kernel.config --version 6.12.0
```

You can use the included configs as a starting point:

```bash
cp configs/microvm-kernel-x86_64.config my-kernel.config
# edit as needed
anvil build-kernel --config ./my-kernel.config
```

## ARM64 Note

The `microvm-kernel-aarch64.config` is included but ARM64 (aarch64) support is **experimental and untested**. Use on x86_64 hosts for production workloads.
