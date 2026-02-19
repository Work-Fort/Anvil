# Third-Party Notices

This project uses the following third-party libraries:

## libguestfs

- **Copyright:** Copyright (C) 2009-2023 Red Hat Inc.
- **License:** GNU Lesser General Public License v2.1 or later (LGPL-2.1-or-later)
- **SPDX Identifier:** LGPL-2.1-or-later
- **Source:** https://libguestfs.org/
- **Usage:** Dynamically linked at runtime for rootfs creation functionality
- **License Text:** See [third_party/libguestfs.org/guestfs/LICENSE.md](third_party/libguestfs.org/guestfs/LICENSE.md)

### Vendored Components

This project vendors the official libguestfs golang bindings (Go wrapper code) in
`third_party/libguestfs.org/guestfs/`:

- **Library bindings** (`guestfs.go`, `go.mod`) - LGPL 2.1 or later
- **Test files** (`*_test.go`) - GPL 2.0 or later (not distributed in binary)

The Go bindings dynamically link to the system's libguestfs shared library at runtime.
Users may replace the libguestfs library with a modified version, and the modified
library will be used without requiring recompilation of anvil.

**Important:** The LGPL allows dynamic linking from Apache 2.0 licensed code without
affecting the main project's license, as long as users can replace the LGPL library.

## Go Dependencies

All other dependencies are statically linked and use permissive licenses
compatible with Apache 2.0:

### MIT License

- **Bubble Tea** (github.com/charmbracelet/bubbletea) - Terminal UI framework
- **Bubbles** (github.com/charmbracelet/bubbles) - TUI components
- **Lipgloss** (github.com/charmbracelet/lipgloss) - Terminal styling
- **Huh** (github.com/charmbracelet/huh) - Form library
- **Log** (github.com/charmbracelet/log) - Structured logging
- **Glamour** (github.com/charmbracelet/glamour) - Markdown rendering
- **gopenpgp** (github.com/ProtonMail/gopenpgp/v3) - OpenPGP library

### Apache License 2.0

- **Cobra** (github.com/spf13/cobra) - CLI framework

### Mozilla Public License 2.0

- **go-version** (github.com/hashicorp/go-version) - Version comparison library
  - Note: MPL 2.0 is a weak copyleft license compatible with Apache 2.0.
    Only modifications to MPL-licensed files must remain MPL.

### BSD 3-Clause License

- **xz** (github.com/ulikunitz/xz) - XZ compression library

For a complete list of dependencies including transitive dependencies, see [go.mod](go.mod).
