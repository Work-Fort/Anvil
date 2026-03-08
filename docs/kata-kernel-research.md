# Kata Guest Kernel Requirements

Research into what the Anvil microvm kernel needs for Kata Containers 3.27.0
compatibility.

## Status by Kernel Version

### 6.19.5 (current)

Nearly Kata-ready. The 6.19.5 config fixed all critical gaps from 6.1.164:

- `VIRTIO_MMIO_CMDLINE_DEVICES=y` — fixed
- `X86_MPPARSE=y` — fixed
- `BTRFS_FS_POSIX_ACL=y` — fixed
- `MEMCG_SWAP` — no longer needed (cgroups v2 has swap accounting built-in)
- `MEMORY_HOTPLUG_DEFAULT_ONLINE` — renamed to `MHP_DEFAULT_ONLINE_TYPE_ONLINE_AUTO`
  in kernel 6.14+, already set to `y`

One item remains:

| Config | Status | Impact |
|--------|--------|--------|
| `FS_DAX` | `not set` | Kata QEMU boots `kata-containers.img` via NVDIMM with DAX. Without `FS_DAX`, the guest kernel can't mount the rootfs and the agent never starts (vsock timeout). All dependencies (`ZONE_DEVICE`, `DAX`, `LIBNVDIMM`) are already enabled. |

Rebuild with `CONFIG_FS_DAX=y` is pending from the Anvil team.

### 6.1.164 (previous)

Had three critical gaps and two recommended gaps. See the "Gaps (6.1.164)"
section below for the full list.

## What the Anvil Kernel Already Has

Both versions have the core Kata requirements:

- **Cgroups:** all controllers (memcg, pids, freezer, cpusets, device, BPF)
- **Namespaces:** all present (uts, ipc, user, pid, net)
- **Vsock:** `VIRTIO_VSOCKETS` + `VIRTIO_VSOCKETS_COMMON` (agent ↔ shim communication)
- **Virtio:** `VIRTIO_MMIO`, `VIRTIO_BLK`, `VIRTIO_NET`, `VIRTIO_CONSOLE`
- **Filesystems:** ext4, btrfs, overlayfs, tmpfs, devtmpfs, proc, sysfs
- **Security:** seccomp + seccomp filter
- **Networking:** bridge, veth, netfilter, iptables, conntrack
- **Block devices:** loop, virtio-blk
- **Core:** epoll, signalfd, timerfd, futex, shmem, PTYs, ELF, serial console

## Gaps (6.1.164)

These were missing in the 6.1.164 config. Listed for reference — all except
`MEMORY_HOTPLUG_DEFAULT_ONLINE` are fixed in 6.19.5.

### Required — Kata will not work without these

| Config | 6.1.164 | 6.19.5 | Why |
|--------|---------|--------|-----|
| `VIRTIO_MMIO_CMDLINE_DEVICES` | `not set` | `y` | Firecracker passes virtio-mmio device addresses via kernel cmdline. Without this, the guest can't discover hotplugged block devices. |
| `X86_MPPARSE` | `not set` | `y` | MP table parsing. Firecracker uses this to communicate vCPU topology to the guest. |
| `MEMORY_HOTPLUG_DEFAULT_ONLINE` | `not set` | `not set` | Auto-online hotplugged memory. Not needed with `static_sandbox_resource_mgmt = true`. |

### Recommended — needed for QEMU path and DAX boot

These are needed if the same kernel is used with QEMU (dev), or if Kata boots
its guest image via DAX/NVDIMM:

| Config | Why |
|--------|-----|
| `ZONE_DEVICE` | Required for DAX/NVDIMM. Kata's QEMU config boots `kata-containers.img` via NVDIMM with DAX. |
| `BLK_DEV_PMEM` | Persistent memory block device. Needed for NVDIMM boot path. |
| `LIBNVDIMM` | NVDIMM library. Dependency for BLK_DEV_PMEM. |
| `VIRTIO_PCI` | Virtio over PCI bus. QEMU uses PCI, not MMIO. |
| `VIRTIO_FS` | Virtio-fs. QEMU shares overlayfs rootfs via virtio-fs. |
| `FUSE_FS` | FUSE layer. Dependency for VIRTIO_FS. |
| `NET_9P_VIRTIO` | 9p over virtio. Firecracker fallback for filesystem sharing. |
| `9P_FS` | 9p filesystem. Dependency for NET_9P_VIRTIO. |

### Optional — nice to have

| Config | 6.1.164 | 6.19.5 | Why |
|--------|---------|--------|-----|
| `BTRFS_FS_POSIX_ACL` | missing | `y` | POSIX ACLs on btrfs. Needed if container images set ACLs. |
| `ZSTD_COMPRESS` | missing | ? | Btrfs zstd compression support. |
| `MEMCG_SWAP` | missing | N/A | Swap accounting. Always-on in cgroups v2 (6.19.5 uses cgroupv2-only). |
| `CRYPTO_CRC32C_INTEL` | `not set` | ? | Hardware-accelerated CRC32C for btrfs. |

## Test Plan

1. Set the Anvil kernel in Kata config (see `nexus/lead/docs/kata-kernel-testing.md`)
2. Verify with Nexus:
   ```bash
   curl -X POST http://127.0.0.1:9600/v1/vms \
     -d '{"name":"anvil-test","role":"agent","runtime":"io.containerd.kata.v2"}'
   curl -X POST http://127.0.0.1:9600/v1/vms/anvil-test/start
   curl -X POST http://127.0.0.1:9600/v1/vms/anvil-test/exec \
     -d '{"cmd":["uname","-r"]}'
   # Should return 6.19.5 (Anvil kernel), not the Kata stock kernel
   ```

## References

- Nexus kernel testing guide: `nexus/lead/docs/kata-kernel-testing.md`
- Kata kernel config fragments: `github.com/kata-containers/kata-containers/tree/main/tools/packaging/kernel/configs/fragments/`
- Firecracker guest kernel policy: `github.com/firecracker-microvm/firecracker/blob/main/docs/kernel-policy.md`
- Kata virtualization design: `github.com/kata-containers/kata-containers/blob/main/docs/design/virtualization.md`
