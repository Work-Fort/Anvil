Getting Started
###############

Build Firecracker-compatible kernels from source with cryptographic verification.

Quick Start
***********

Install Type
============

.. tabs::

   .. tab:: Full

      Trust the internet, live care free:

      .. code-block:: bash

         curl -fsSL https://kernels.workfort.io/install-anvil.sh | sh

   .. tab:: Manual

      Download and verify with SHA256 checksums:

      .. code-block:: bash

         # Get latest kernel version from kernel.org
         VERSION=$(curl -s https://www.kernel.org/releases.json | jq -r '.latest_stable.version')

         # Download kernel
         curl -LO "https://github.com/Work-Fort/Anvil/releases/download/v${VERSION}/vmlinux-${VERSION}-x86_64.xz"

         # Download checksums
         curl -LO "https://github.com/Work-Fort/Anvil/releases/download/v${VERSION}/SHA256SUMS"

         # Verify checksum
         sha256sum -c SHA256SUMS --ignore-missing

         # Decompress
         xz -d "vmlinux-${VERSION}-x86_64.xz"

   .. tab:: No Helper

      Full security: PGP signature verification + SHA256 checksums:

      .. code-block:: bash

         # Get latest kernel version from kernel.org
         VERSION=$(curl -s https://www.kernel.org/releases.json | jq -r '.latest_stable.version')

         # Import Cracker Barrel signing key
         curl -s https://raw.githubusercontent.com/Work-Fort/Anvil/master/keys/signing-key.asc | gpg --import

         # Verify key fingerprint matches README
         gpg --fingerprint me@kazatron.com

         # Download kernel, checksums, and signature
         curl -LO "https://github.com/Work-Fort/Anvil/releases/download/v${VERSION}/vmlinux-${VERSION}-x86_64.xz"
         curl -LO "https://github.com/Work-Fort/Anvil/releases/download/v${VERSION}/SHA256SUMS"
         curl -LO "https://github.com/Work-Fort/Anvil/releases/download/v${VERSION}/SHA256SUMS.asc"

         # Verify PGP signature on checksums file
         gpg --verify SHA256SUMS.asc SHA256SUMS

         # Verify kernel checksum
         sha256sum -c SHA256SUMS --ignore-missing

         # Decompress
         xz -d "vmlinux-${VERSION}-x86_64.xz"
