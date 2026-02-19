CLI Command Reference
=====================

Command List
------------

**kernel** - Manage pre-built kernel binaries::

    ✓ anvil kernel                                     # Launch TUI version manager
    ✓ anvil kernel get [VERSION] [--arch ARCH]         # Download pre-built kernel (alias: download)
    ✓ anvil kernel list                                # List installed kernels (launches TUI)
    ✓ anvil kernel versions                            # Show available versions (launches TUI)
    ✓ anvil kernel set [VERSION]                       # Set default kernel (alias: default, launches TUI if no version)
    ✓ anvil kernel remove [VERSION]                    # Remove installed kernel (launches TUI if no version)

**build-kernel** - Build kernels from source::

    ✓ anvil build-kernel [VERSION] [flags]             # Interactive TUI wizard (alias: build)
        --arch string                                            # Target architecture: x86_64, aarch64, or all
        --config string                                          # Custom kernel config file
        --force-rebuild                                          # Force rebuild even if cached build exists
        --verification-level string                              # Verification level: high, medium, disabled

**firecracker** - Manage Firecracker binaries::

    ✓ anvil firecracker                                # Launch TUI version manager
    ✓ anvil firecracker get [VERSION]                  # Download Firecracker binary (alias: download)
    ✓ anvil firecracker list                           # List installed versions (launches TUI)
    ✓ anvil firecracker versions                       # Show available versions (launches TUI)
    ✓ anvil firecracker set [VERSION]                  # Set default version (alias: default, launches TUI if no version)
    ✓ anvil firecracker remove [VERSION]               # Remove version (launches TUI if no version)
    ✓ anvil firecracker create-rootfs [flags]          # Create Alpine Linux rootfs for Firecracker (alias: mkrootfs)
        --alpine-version string                                  # Alpine Linux version (major.minor) (default "3.23")
        --alpine-patch string                                    # Alpine Linux patch version (default "3")
        --size int                                               # Size in MB (default 512)
        --output string                                          # Output file path (default: XDG_DATA_HOME/anvil/alpine-rootfs.ext4)
        --inject-binary                                          # Inject binary into rootfs
        --binary-path string                                     # Path to binary to inject (default: current executable)
        --binary-dest string                                     # Destination path in rootfs (default "/usr/bin/anvil")
        --force                                                  # Overwrite existing file

**clean** - Cleanup operations::

    ✓ anvil clean kernel [flags] [VERSION]             # Remove kernel binaries
        --remove-inactive                                        # Remove all non-default kernel versions
        --all-dangerous                                          # Remove all kernel data (requires confirmation)
        --force                                                  # Skip confirmation prompt
    ✓ anvil clean firecracker [flags]                  # Remove Firecracker binaries
        --remove-inactive                                        # Remove all non-default Firecracker versions
        --all-dangerous                                          # Remove all Firecracker data (requires confirmation)
        --force                                                  # Skip confirmation prompt
    ✓ anvil clean build-kernel [--arch ARCH]           # Remove kernel source and build artifacts
        --arch string                                            # Architecture to clean: x86_64, aarch64, or all
    ✓ anvil clean rootfs                               # Remove Alpine rootfs images

**Root commands**::

    ✓ anvil version                                    # Show version info
    ✓ anvil update [--version VERSION]                 # Update CLI from GitHub
    ✓ anvil completion [bash|zsh|fish]                 # Generate shell completion script
