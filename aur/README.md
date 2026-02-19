# AUR Package for Anvil

This directory contains the PKGBUILD for publishing anvil to the Arch User Repository (AUR).

## Setup

### 1. Create AUR Account

Register at https://aur.archlinux.org/register/

### 2. Generate SSH Key for AUR

```bash
ssh-keygen -t ed25519 -C "aur-anvil" -f ~/.ssh/aur
```

### 3. Add Public Key to AUR

1. Go to https://aur.archlinux.org/account/
2. Navigate to "My Account" → "SSH Public Key"
3. Paste the contents of `~/.ssh/aur.pub`

### 4. Add Private Key to GitHub Secrets

1. Go to your GitHub repository → Settings → Secrets and variables → Actions
2. Click "New repository secret"
3. Name: `AUR_SSH_PRIVATE_KEY`
4. Value: Paste the contents of `~/.ssh/aur` (the **private** key)

### 5. Initialize AUR Repository

```bash
# Clone the empty AUR repository
git clone ssh://aur@aur.archlinux.org/anvil.git aur-anvil
cd aur-anvil

# Copy PKGBUILD from this directory
cp ../aur/PKGBUILD .

# Update PKGBUILD with current version
sed -i 's/pkgver=.*/pkgver=YOUR_VERSION/' PKGBUILD

# Generate .SRCINFO
makepkg --printsrcinfo > .SRCINFO

# Commit and push initial version
git add PKGBUILD .SRCINFO
git commit -m "Initial commit: anvil YOUR_VERSION"
git push origin master
```

## How It Works

When a new CLI release is created (tagged with `cli-v*`):

1. The `publish-aur.yml` workflow automatically triggers
2. It downloads the source tarball from the GitHub release tag
3. Calculates checksum of the source tarball
4. Updates the PKGBUILD with the new version and checksum
5. Generates .SRCINFO
6. Commits and pushes to the AUR repository

When users install via AUR:
- `makepkg` downloads the source tarball
- Verifies the checksum
- Builds with `DISABLE_UPDATE=true` flag
- Installs the binary with updates disabled

## Building Locally

To test the PKGBUILD locally:

```bash
cd aur
makepkg -si
```

This will:
- Download the source tarball
- Verify the checksum
- Build anvil with `DISABLE_UPDATE=true` (~5 seconds)
- Run tests
- Install to your system

Note: The binary is built with `DISABLE_UPDATE=true`, so `anvil update` will direct users to their package manager.

## Package Details

- **Type**: Source package (builds from source with AUR-specific flags)
- **Build Time**: ~5 seconds on modern hardware
- **Update Flag**: Built with `DISABLE_UPDATE=true` so `anvil update` directs users to their package manager
- **Build Dependencies**: `go`, `git` (compile-time only)
- **Runtime Dependencies**: `libguestfs`
- **Architectures**: x86_64 and aarch64

## Troubleshooting

### "Permission denied (publickey)" when pushing to AUR

Make sure your SSH key is:
1. Added to your AUR account
2. Stored in GitHub Secrets as `AUR_SSH_PRIVATE_KEY`
3. Has correct permissions (`chmod 600 ~/.ssh/aur`)

### Workflow fails with "Could not calculate checksum"

The workflow expects a source tarball at:
```
https://github.com/YOUR_REPO/archive/refs/tags/cli-vX.Y.Z.tar.gz
```

Make sure the release tag exists and GitHub has generated the source tarball.

### .SRCINFO generation fails

The workflow uses a Docker container to generate .SRCINFO. If it fails, you can generate it manually:

```bash
makepkg --printsrcinfo > .SRCINFO
```

Then commit it to the AUR repo.
