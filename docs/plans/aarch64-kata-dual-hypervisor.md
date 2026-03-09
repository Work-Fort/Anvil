# Plan: Add Kata dual-hypervisor support to aarch64 kernel config

## Context

The x86_64 kernel config was updated with Kata Containers dual-hypervisor support (Firecracker + QEMU) and confirmed working by the Nexus team. The aarch64 config needs the same treatment — it was originally an Amazon Linux microvm config and has significantly more gaps than x86_64 did (PCI entirely disabled, no NVDIMM/DAX stack).

## File to modify

`configs/microvm-kernel-aarch64.config`

## Changes

Organized to mirror the x86_64 config, plus ARM64-specific options from Kata's upstream fragments.

### 1. Flip existing `not set` → `y`

| Line | Option | Why |
|------|--------|-----|
| 441 | `ARM64_PMEM` | ARM64 persistent memory support for NVDIMM/DAX |
| 585 | `ACPI_PROCESSOR` | ACPI CPU hotplug for QEMU |
| 587 | `ACPI_TABLE_UPGRADE` | ACPI table updates |
| 589 | `ACPI_CONTAINER` | ACPI container hotplug |
| 590 | `ACPI_HOTPLUG_MEMORY` | ACPI memory hotplug |
| 909 | `MEMORY_HOTPLUG_DEFAULT_ONLINE` → `MHP_DEFAULT_ONLINE_TYPE_ONLINE_AUTO` | Auto-online hotplugged memory (renamed in 6.14+) |
| 949 | `ZONE_DEVICE` | DAX/NVDIMM device memory zones |
| 1425 | `NET_9P` | 9p transport layer |
| 1445 | `PCI` | PCI bus support — QEMU uses PCI for virtio |
| 1636 | `BLK_DEV_SD` | SCSI disk hotplug |
| 1661 | `SCSI_VIRTIO` | Virtio SCSI transport |
| 1666 | `MD` | Device-mapper parent |
| 1793 | `SERIAL_AMBA_PL011` | ARM PL011 UART for QEMU serial console |
| 2224 | `VIRTIO_MEM` | Virtio memory hotplug |
| 2227 | `VIRTIO_MMIO_CMDLINE_DEVICES` | Firecracker device discovery |
| 2453 | `LIBNVDIMM` | NVDIMM library |
| 2454 | `DAX` | Direct Access support |
| 2528 | `FUSE_FS` | FUSE layer (dep for VIRTIO_FS) |
| 2530 | `OVERLAY_FS_REDIRECT_DIR` | OverlayFS redirect (Docker-in-Docker) |
| 2532 | `OVERLAY_FS_INDEX` | OverlayFS inode index |
| 2632 | `EROFS_FS` | Read-only container image layers |
| 2143 | `RTC_DRV_EFI` | EFI RTC for UEFI/ACPI boot |

### 2. Add missing options (after related entries)

| After | Add | Why |
|-------|-----|-----|
| `ACPI_HOTPLUG_MEMORY` | `ACPI_NFIT=y` | NVDIMM firmware interface table |
| `NET_9P` | `NET_9P_VIRTIO=y` | 9p over virtio transport |
| `PCI` | `PCI_MSI=y` | Message Signaled Interrupts |
| (PCI subtree) | `PCI_HOST_COMMON=y`, `PCI_HOST_GENERIC=y` | Generic PCI host for ARM64 mach-virt |
| (PCI subtree) | `VIRTIO_PCI=y` | Virtio over PCI |
| `SERIAL_AMBA_PL011` | `SERIAL_AMBA_PL011_CONSOLE=y` | Console output on PL011 |
| `VIRTIO_MEM` | `VIRTIO_PMEM=y` | Virtio persistent memory |
| `MD` | `BLK_DEV_DM=y`, `DM_VERITY=y`, `DM_INIT=y` | Device-mapper, integrity, boot-time DM |
| `LIBNVDIMM` | `BLK_DEV_PMEM=y` | Persistent memory block device |
| `DAX` | `FS_DAX=y` | DAX filesystem support |
| `FUSE_FS` | `VIRTIO_FS=y` | Virtio-fs for QEMU rootfs sharing |
| After network filesystems section | `9P_FS=y` | 9p filesystem |
| `BTRFS_FS` | `BTRFS_FS_POSIX_ACL=y` | POSIX ACLs on btrfs |

### 3. Device passthrough (VFIO)

| Line | Option | Why |
|------|--------|-----|
| 2217 | `VFIO` (flip to `y`) | Virtual Function I/O — device passthrough for `vfio_mode=vfio` |
| (add) | `VFIO_PCI=y` | PCI device passthrough (depends on PCI + VFIO) |

`IOMMU_SUPPORT=y` is already enabled (line 2296).

### 4. nftables

| Line | Option | Why |
|------|--------|-----|
| 1121 | `NF_TABLES` (flip to `y`) | nftables core — modern netfilter replacement, used by istio sidecars |

Key NFT_* sub-options to add after `NF_TABLES`:
- `NF_TABLES_INET=y`, `NF_TABLES_NETDEV=y` — inet and netdev family support
- `NFT_CT=y` — connection tracking
- `NFT_NAT=y`, `NFT_MASQ=y`, `NFT_REDIR=y` — NAT/masquerade/redirect
- `NFT_COMPAT=y` — iptables compatibility layer
- `NFT_REJECT=y`, `NFT_FIB=y`, `NFT_FIB_INET=y`, `NFT_FIB_IPV4=y`, `NFT_FIB_IPV6=y`
- `NFT_CONNLIMIT=y`, `NFT_LOG=y`, `NFT_NUMGEN=y`, `NFT_COUNTER=y`, `NFT_LIMIT=y`
- `NFT_SOCKET=y`, `NFT_TPROXY=y`, `NFT_SYNPROXY=y`
- `NFT_DUP_NETDEV=y`, `NFT_FWD_NETDEV=y`, `NFT_FIB_NETDEV=y`
- `NFT_BRIDGE_META=y`, `NFT_BRIDGE_REJECT=y`
- `NFT_DUP_IPV4=y`, `NFT_DUP_IPV6=y`
- `NFT_REJECT_INET=y`, `NFT_REJECT_IPV4=y`, `NFT_REJECT_IPV6=y`
- `NFT_XFRM=y`, `NFT_OSF=y`

### 5. Skipped (not needed for minimum working set)

- `EXPERT` + `CRYPTO_FIPS` chain — too wide-reaching, changes many defaults
- `DEBUG_INFO` — increases kernel size, not needed for production
- `NO_HZ_FULL`, `RANDOMIZE_BASE`, `ARM64_PSEUDO_NMI` — nice-to-have, not functional blockers
- `ETHERNET`, `TUN`, `MACVLAN`, `VXLAN` — advanced networking, not required for base Kata
- `NR_CPUS=255` — current 64 is sufficient
- `TCP_CONG_BBR` — performance tuning, not functional
- `IKCONFIG` — debugging aid only

## Verification

1. Build: `./build/anvil build-kernel --version 6.19.5 --config configs/microvm-kernel-aarch64.config --arch aarch64 --use-tui=false`
2. Confirm output config has all options enabled: grep for each CONFIG_ in the artifacts config
3. Commit once build passes
