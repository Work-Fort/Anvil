# Hexagonal Architecture for Anvil

Design document examining how hexagonal (ports-and-adapters) architecture, as
implemented in Nexus and Sharkfin, would apply to Anvil.

## Reference Implementations

### Nexus

Nexus manages container-based VMs with networking, storage, and DNS. Its
architecture splits cleanly into four layers:

```
internal/domain/       Pure types + port interfaces (zero infra imports)
internal/app/          Use-case service (VMService)
internal/infra/        Adapter implementations
cmd/                   Wiring + entry points
```

**Domain** (`internal/domain/`):

- `vm.go` — VM, VMState, VMRole, CreateVMParams, ExecResult, VMFilter
- `drive.go` — Drive, CreateDriveParams
- `device.go` — Device, CreateDeviceParams
- `ports.go` — VMStore, Runtime, Network, DriveStore, DeviceStore, Storage, DNSManager

The domain package imports only `context`, `errors`, and `time`. No
infrastructure. No framework. This is the core constraint that makes the
architecture work — domain types are stable and testable in isolation.

**Application** (`internal/app/`):

A single `VMService` struct holds port interfaces and orchestrates use cases:

```go
type VMService struct {
    store       domain.VMStore
    runtime     domain.Runtime
    network     domain.Network
    driveStore  domain.DriveStore
    storage     domain.Storage
    deviceStore domain.DeviceStore
    dns         domain.DNSManager
    config      VMServiceConfig
}
```

Construction uses functional options to inject optional capabilities:

```go
svc := app.NewVMService(store, runtime, network,
    app.WithStorage(store, storageBackend),
    app.WithDeviceStore(store),
    app.WithDNS(dnsManager),
    app.WithConfig(cfg),
)
```

The service validates domain rules, coordinates across ports, and handles
rollback on failure (e.g., if runtime.Create fails, tear down networking that
was already set up).

**Adapters** (`internal/infra/`):

Each adapter implements one or more port interfaces:

| Adapter | Implements | Backed by |
|---------|-----------|-----------|
| `sqlite/` | VMStore, DriveStore, DeviceStore | SQLite database |
| `containerd/` | Runtime | containerd gRPC API |
| `cni/` | Network | CNI plugins + netns |
| `storage/` | Storage | btrfs subvolumes or plain directories |
| `dns/` | DNSManager | CoreDNS process |
| `httpapi/` | — (driving adapter) | HTTP server exposing VMService |

The `httpapi` adapter is the "driving" side — it receives external requests and
calls into `VMService`. All other adapters are "driven" — the service calls out
to them.

**Wiring** (`cmd/daemon.go`):

The command constructs concrete adapters and injects them:

```go
store, _ := sqlite.Open(dbPath)
runtime, _ := ctrd.New(socketPath, namespace)
network := cni.New(cniConfig)
storage := storage.NewBtrfsWithQuota(drivesDir, quotaHelper)

svc := app.NewVMService(store, runtime, network,
    app.WithStorage(store, storage),
    app.WithDeviceStore(store),
    app.WithDNS(dnsManager),
)

handler := httpapi.NewHandler(svc)
http.Serve(listener, handler)
```

No domain or application code knows about SQLite, containerd, or HTTP. The
wiring happens once, at the edge.

### Sharkfin

Sharkfin is a chat server. Same pattern, simpler because it has fewer adapter
types:

```
pkg/domain/            types.go (User, Channel, Message, etc.)
                       ports.go (UserStore, ChannelStore, MessageStore, etc.)
pkg/daemon/            Application layer (WebSocket handler, webhooks, MCP)
pkg/infra/             sqlite/ and postgres/ adapters
                       open.go — factory that selects adapter from DSN
```

The composite `Store` interface aggregates all store ports:

```go
type Store interface {
    UserStore
    ChannelStore
    MessageStore
    RoleStore
    SettingsStore
    Close() error
}
```

Both SQLite and PostgreSQL adapters implement this composite interface. The
`infra.Open(dsn)` factory auto-detects the backend from the connection string
and returns `domain.Store` — callers never know which database they're using.

### What Both Projects Share

1. **Domain owns the interfaces.** Port interfaces live in `domain/`, not in
   the adapter packages. The dependency arrow points inward.

2. **One application service per bounded context.** Nexus has `VMService`,
   Sharkfin has the daemon package. The service orchestrates ports but contains
   no I/O itself.

3. **Adapters are leaf packages.** They import domain types but nothing imports
   them except the wiring code in `cmd/`.

4. **Wiring happens in main.** Concrete adapter construction and injection
   happens in command handlers, not in libraries.

5. **Noop adapters for optional capabilities.** Nexus has `NoopNetwork` and
   `NoopDNSManager` for when features are disabled. This avoids nil checks
   throughout the service.

## Anvil Today

Anvil's current package layout:

```
pkg/kernel/       Build orchestration + kernel.org HTTP + os/exec + filesystem
pkg/signing/      PGP domain logic + gopenpgp + exec.Command + file I/O
pkg/firecracker/  Firecracker management + HTTP downloads + exec + vsock
pkg/download/     HTTP client with progress (well-isolated adapter)
pkg/github/       GitHub API client (well-isolated adapter)
pkg/config/       Viper config + XDG paths + theme
pkg/kconfig/      Kernel config parsing (mostly pure)
pkg/rootfs/       Alpine rootfs creation via libguestfs
pkg/util/         Compression, checksums
pkg/init/         Repository initialization logic
pkg/ui/           TUI components (Bubble Tea)
pkg/vsock/        Vsock protocol
```

The coupling pattern in `pkg/kernel/build.go` is representative:

```go
func Build(ctx context.Context, opts BuildOptions) (*BuildStats, error) {
    // 1. Calls kernel.org HTTP API to validate version
    // 2. Downloads tar.xz via pkg/download
    // 3. Verifies PGP signatures via pkg/signing
    // 4. Extracts with os/exec (tar)
    // 5. Runs make via os/exec
    // 6. Generates SHA256SUMS via os.WriteFile
    // 7. Copies artifacts via os.Rename
}
```

This function mixes domain decisions (what phases to run, how to handle cached
builds) with six different I/O concerns. Testing it requires a real filesystem,
network access, and installed build tools.

**What's already well-isolated:**

- `pkg/download/` — Pure HTTP adapter, could be moved directly to `infra/`
- `pkg/github/` — Pure GitHub API adapter
- `pkg/kconfig/` — Mostly pure parsing logic, good domain candidate
- `pkg/vsock/` — Protocol implementation, standalone

**What's tightly coupled:**

- `pkg/kernel/` — Mixes build orchestration with HTTP, exec, and filesystem
- `pkg/signing/` — Mixes key management logic with gopenpgp and exec
- `pkg/firecracker/` — Mixes test orchestration with process management

## Anvil Under Hexagonal Architecture

### Domain Layer

```
internal/domain/
├── kernel.go        Kernel, Version, BuildPhase, BuildOptions, BuildStats
├── signing.go       KeyInfo, GenerateKeyOptions, KeyFormat
├── firecracker.go   FirecrackerVersion, TestOptions, TestResult
├── config.go        ConfigValue, ConfigScope, PathLayout
├── kconfig.go       Option, DiffEntry (moved from pkg/kconfig)
├── errors.go        Sentinel errors
└── ports.go         All port interfaces
```

### Port Interfaces

```go
package domain

// VersionRegistry resolves kernel versions from an upstream source.
type VersionRegistry interface {
    LatestStable(ctx context.Context) (string, error)
    Validate(ctx context.Context, version string) error
    List(ctx context.Context) ([]string, error)
}

// SourceDownloader fetches and verifies kernel source archives.
type SourceDownloader interface {
    Download(ctx context.Context, version string, dest string, progress func(float64)) error
    Verify(ctx context.Context, version string, archivePath string) error
}

// KernelCompiler runs the kernel build process.
type KernelCompiler interface {
    Configure(ctx context.Context, srcDir string, configFile string) error
    Compile(ctx context.Context, srcDir string, arch string, jobs int) error
}

// ArtifactStore manages built kernel binaries on disk.
type ArtifactStore interface {
    Save(ctx context.Context, kernel *Kernel) error
    Get(ctx context.Context, version string) (*Kernel, error)
    List(ctx context.Context) ([]*Kernel, error)
    Delete(ctx context.Context, version string) error
    GetDefault(ctx context.Context) (*Kernel, error)
    SetDefault(ctx context.Context, version string) error
}

// ArchiveStore manages kernel archives for distribution.
type ArchiveStore interface {
    Archive(ctx context.Context, kernel *Kernel) error
    Get(ctx context.Context, version, arch string) (*ArchivedKernel, error)
    List(ctx context.Context) ([]*ArchivedKernel, error)
}

// Signer handles PGP signing and verification of artifacts.
type Signer interface {
    Sign(ctx context.Context, artifactsDir string) error
    Verify(ctx context.Context, artifactsDir string) error
    KeyInfo(ctx context.Context) (*KeyInfo, error)
    GenerateKey(ctx context.Context, opts GenerateKeyOptions) (*KeyInfo, error)
    RotateKey(ctx context.Context, opts GenerateKeyOptions) (*KeyInfo, error)
    ExportBackup(ctx context.Context, email, outputPath string) error
    ImportBackup(ctx context.Context, backupPath string) error
}

// FirecrackerManager handles Firecracker binary lifecycle.
type FirecrackerManager interface {
    Download(ctx context.Context, version string) error
    List(ctx context.Context) ([]FirecrackerVersion, error)
    Remove(ctx context.Context, version string) error
    SetDefault(ctx context.Context, version string) error
    Test(ctx context.Context, opts TestOptions) (*TestResult, error)
}
```

These interfaces reflect Anvil's actual operations — they're not generic
filesystem/network ports but domain-meaningful boundaries. A `SourceDownloader`
knows about kernel versions and archive formats; a `KernelCompiler` knows about
configure and make. This follows the Nexus pattern where `Runtime` knows about
containers (not generic process execution) and `Network` knows about namespaces
(not generic networking).

### Application Layer

```
internal/app/
├── kernel.go         KernelService — orchestrates build lifecycle
├── signing.go        SigningService — key and artifact signing
└── firecracker.go    FirecrackerService — binary management + testing
```

```go
type KernelService struct {
    registry   domain.VersionRegistry
    downloader domain.SourceDownloader
    compiler   domain.KernelCompiler
    artifacts  domain.ArtifactStore
    archives   domain.ArchiveStore
    signer     domain.Signer
}

func (s *KernelService) Build(ctx context.Context, opts domain.BuildOptions) (*domain.BuildStats, error) {
    // Pure orchestration:
    // 1. registry.Validate(version)
    // 2. downloader.Download(version, tmpDir, progress)
    // 3. downloader.Verify(version, archivePath)
    // 4. compiler.Configure(srcDir, configFile)
    // 5. compiler.Compile(srcDir, arch, jobs)
    // 6. artifacts.Save(kernel)
    // 7. signer.Sign(artifactsDir)  [if key available]
}
```

### Adapter Layer

```
internal/infra/
├── kernelorg/        Implements VersionRegistry (kernel.org JSON API)
├── github/           Implements VersionRegistry (GitHub releases) — for Firecracker
├── httpdl/           Implements SourceDownloader (HTTP + PGP verify)
├── makebuilder/      Implements KernelCompiler (os/exec make)
├── localfs/          Implements ArtifactStore, ArchiveStore (XDG directories)
├── gpg/              Implements Signer (gopenpgp)
├── firecracker/      Implements FirecrackerManager (download + exec)
└── mcp/              Driving adapter — MCP stdio server (current internal/mcp/)
```

### Driving Adapters

Anvil has three driving adapters — entry points that call into the application
layer:

1. **CLI commands** (`cmd/`) — Cobra handlers call service methods
2. **TUI wizards** (`pkg/ui/`) — Bubble Tea models call service methods
3. **MCP server** (`internal/mcp/`) — Tool handlers call service methods

Currently all three call directly into `pkg/kernel`, `pkg/signing`, etc. Under
hexagonal architecture they'd all call `app.KernelService`,
`app.SigningService`, etc.

### Wiring

```go
// cmd/root.go or cmd/wire.go
func buildServices() (*app.KernelService, *app.SigningService, *app.FirecrackerService) {
    registry := kernelorg.New()
    downloader := httpdl.New()
    compiler := makebuilder.New()
    artifacts := localfs.NewArtifactStore(config.GlobalPaths)
    archives := localfs.NewArchiveStore(config.GlobalPaths)
    signer := gpg.New(config.GlobalPaths.KeysDir)

    kernelSvc := app.NewKernelService(registry, downloader, compiler, artifacts, archives, signer)
    signingSvc := app.NewSigningService(signer)
    fcSvc := app.NewFirecrackerService(...)

    return kernelSvc, signingSvc, fcSvc
}
```

## Key Differences from Nexus

| Aspect | Nexus | Anvil |
|--------|-------|-------|
| Lifecycle | Long-running daemon | CLI invocations |
| State | SQLite database | Filesystem (XDG dirs) |
| Adapters | 6 driven + 1 driving (HTTP) | 6-7 driven + 3 driving (CLI, TUI, MCP) |
| Service count | 1 (VMService) | 3 (Kernel, Signing, Firecracker) |
| Port granularity | Coarse (Runtime covers create/start/stop/exec) | Mixed (Compiler is narrow, Signer is broad) |

Nexus has one service because VMs are the single aggregate root — drives,
devices, networking all attach to VMs. Anvil's domains are more independent:
you can sign artifacts without building, manage Firecracker without kernels.
Three services reflects this.

## What This Buys

**Testability.** A `KernelService` test can inject a mock `SourceDownloader`
that returns cached bytes, a mock `KernelCompiler` that returns instantly, and
a mock `ArtifactStore` backed by a map. No filesystem, no network, no make.

**Swappable backends.** The `ArtifactStore` port could be implemented as local
filesystem today, S3 tomorrow, or OCI registry later. The `VersionRegistry`
could switch from kernel.org to a private mirror. None of this touches
application logic.

**Multiple frontends sharing logic.** The CLI, TUI, and MCP server currently
duplicate orchestration. Under hex arch, all three call the same service
methods — the only difference is how they present progress and results.

**Noop adapters.** Following Nexus's pattern, optional capabilities (signing,
DNS, networking) can have noop implementations. A `NoopSigner` that does nothing
eliminates conditional signing logic throughout the codebase.

## What This Costs

**More files and indirection.** A direct `kernel.Build()` call becomes
`service.Build()` → `downloader.Download()` → `httpdl.Download()`. Three hops
instead of one. For a CLI tool (not a large team project), this is overhead.

**Interface design effort.** Getting port boundaries right requires thinking
about what's domain-meaningful vs what's incidental infrastructure. The
`KernelCompiler` interface above is straightforward; deciding whether
`SourceDownloader.Verify` belongs on the downloader or the signer is a judgment
call.

**Migration cost.** Anvil's current packages work. Refactoring them into
hexagonal layers touches every file. The migration needs to be incremental — you
can't rewrite everything at once.
