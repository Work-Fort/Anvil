---
name: pre-alpha
description: Guidelines for pre-alpha development - prioritize speed and simplicity over backwards compatibility
---

# Pre-Alpha Development Philosophy

This project is in **pre-alpha** stage. This fundamentally changes how you should approach implementation.

## Core Principle: Speed Over Stability

Pre-alpha means we're exploring, iterating, and finding product-market fit. The code will change dramatically. **Optimize for velocity, not longevity.**

## What This Means in Practice

### ✅ DO: Make Breaking Changes Freely

```go
// Good - just change the schema directly
type Config struct {
    UseTUI bool   // Changed from UseTextUI - just update it
    Port   int    // Added new field - no migration needed
}

// Good - breaking SQL schema changes
ALTER TABLE users DROP COLUMN old_field;
ALTER TABLE users ADD COLUMN new_field TEXT;
```

In pre-alpha, **there is no production data to protect**. No users depend on stability. Just make the change.

### ✅ DO: Delete Unused Code Completely

```go
// Good - complete removal
// Just delete the old function, no deprecation wrapper

// Good - no backwards compat shims
// Old code path? Delete it.
// Old config key? Remove it from schema.
```

**Why?** Dead code creates confusion and maintenance burden. In pre-alpha, if it's not used, delete it immediately.

### ✅ DO: Keep Implementation Simple

```go
// Good - single straightforward implementation
func ProcessData(data string) error {
    // Simple, direct solution
    return process(data)
}

// Bad - unnecessary abstraction for "future flexibility"
type DataProcessor interface {
    Process(data string) error
}

func NewProcessor(mode string) DataProcessor {
    // Pre-alpha doesn't need this complexity
}
```

**YAGNI (You Aren't Gonna Need It)** is critical in pre-alpha. Don't build for hypothetical future requirements.

### ✅ DO: Use Direct SQL Schema Changes

```sql
-- Good - direct ALTER TABLE
ALTER TABLE config DROP COLUMN deprecated_setting;
ALTER TABLE config ADD COLUMN new_setting TEXT NOT NULL DEFAULT 'value';

-- No migration system needed
-- No backwards compatibility layer
-- Just change the schema
```

In pre-alpha, you can assume:
- Clean slate for each tester
- No data migration needed
- Breaking schema changes are fine

### ✅ DO: Update Config Schemas Without Migration

```go
// Good - just change the config structure
const CurrentConfigVersion = 2  // Bumped from 1

type Config struct {
    Version int
    NewField string  // Added directly, no migration code
}

// If old config exists, tell user to regenerate it
func LoadConfig() (*Config, error) {
    cfg := &Config{}
    if err := load(cfg); err != nil {
        return nil, err
    }

    if cfg.Version != CurrentConfigVersion {
        return nil, fmt.Errorf("config version mismatch: delete config and regenerate")
    }

    return cfg, nil
}
```

**Why?** Migration code is complexity. In pre-alpha, users can regenerate configs easily.

### ✅ DO: Rename Functions/Variables Freely

```go
// Good - just rename it everywhere
// Old: ProcessUserInput()
// New: HandleInput()

// Use your editor's rename refactor tool and commit the change
```

No need for:
- Deprecated wrapper functions
- Aliases for old names
- Gradual migration plans

Just rename and move on.

### ❌ DON'T: Build Backwards Compatibility Layers

```go
// Bad - unnecessary in pre-alpha
func NewFeature(data string) error {
    return newFeatureImpl(data)
}

// Backwards compat wrapper (WHY?!)
func OldFeature(data string) error {
    log.Warn("OldFeature deprecated, use NewFeature")
    return NewFeature(data)
}

// Good - just have NewFeature
func NewFeature(data string) error {
    return process(data)
}
```

**Exception:** If the feature itself deals with importing external data or files, you DO need to handle multiple versions of that external format.

### ❌ DON'T: Version Database Schemas

```go
// Bad - unnecessary migration system in pre-alpha
type Migration struct {
    Version int
    Up      func() error
    Down    func() error
}

var migrations = []Migration{
    {Version: 1, Up: migrate1Up, Down: migrate1Down},
    {Version: 2, Up: migrate2Up, Down: migrate2Down},
}

// Good - just have the current schema
const SchemaSQL = `
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);
`
```

In pre-alpha, users can regenerate their database from scratch. No migration system needed.

### ❌ DON'T: Add Feature Flags for Gradual Rollout

```go
// Bad - premature in pre-alpha
if config.EnableNewFeature {
    newImplementation()
} else {
    oldImplementation()
}

// Good - just ship the new implementation
newImplementation()
```

Feature flags are for production systems with users. Pre-alpha has no users to protect.

### ❌ DON'T: Create Abstraction Layers "For Future Use"

```go
// Bad - speculative abstraction
type StorageBackend interface {
    Save(key string, value []byte) error
    Load(key string) ([]byte, error)
}

type FileStorage struct{}
func (f *FileStorage) Save(key string, value []byte) error { /* ... */ }

type RedisStorage struct{}  // "We might need this later"
func (r *RedisStorage) Save(key string, value []byte) error { /* ... */ }

// Good - just implement what you need now
func SaveToFile(key string, value []byte) error {
    return os.WriteFile(filepath.Join(dataDir, key), value, 0644)
}
```

Build abstractions when you have 2+ concrete use cases, not before.

## When Backwards Compatibility DOES Matter

### ✅ Exception: External Data Formats

If your feature deals with importing or reading external data that users create, you DO need versioning:

```go
// Good - file import needs to support multiple versions
func ImportData(filepath string) error {
    data, err := readFile(filepath)
    if err != nil {
        return err
    }

    // Detect version from file content
    version := detectVersion(data)

    switch version {
    case 1:
        return importV1(data)
    case 2:
        return importV2(data)
    default:
        return fmt.Errorf("unsupported file version: %d", version)
    }
}
```

**Why?** Users might have data files from previous versions. That's THEIR data, not internal app state.

### ✅ Exception: API Contracts with External Tools

If you expose an API that external scripts/tools call, version it:

```go
// Good - API versioning when external tools depend on it
// /api/v1/users
// /api/v2/users

func (s *Server) registerRoutes() {
    s.router.Handle("/api/v1/users", v1.HandleUsers)
    s.router.Handle("/api/v2/users", v2.HandleUsers)
}
```

**Why?** External callers can't update instantly when you change the API.

### ✅ Exception: Plugin Systems

If you have a plugin architecture where third parties write plugins:

```go
// Good - versioned plugin interface
type PluginV1 interface {
    Init() error
    Execute(ctx context.Context) error
}

type PluginV2 interface {
    Init(config Config) error
    Execute(ctx context.Context, input Input) (Output, error)
}
```

**Why?** Third-party plugins can't update on your schedule.

## Decision Tree: Do I Need Backwards Compatibility?

```
Is this for internal application state (config, database)?
├─ YES → No backwards compatibility needed in pre-alpha
└─ NO → Is this for external data/API/plugins?
    ├─ YES → Implement versioning
    └─ NO → No backwards compatibility needed
```

## Pre-Alpha Development Checklist

When implementing a feature:

- [ ] Is the solution as simple as possible? (No unnecessary abstraction)
- [ ] Did I delete dead code instead of commenting it out?
- [ ] Did I make breaking changes directly instead of adding compatibility layers?
- [ ] Am I building for current requirements, not hypothetical future needs?
- [ ] If I'm adding versioning/migration, is it actually needed? (See exceptions above)

## Examples

### Example 1: Adding a Config Field

```go
// ❌ Bad - over-engineered for pre-alpha
type Config struct {
    Version int
    Settings map[string]interface{}  // "Flexible for future"
}

func LoadConfig() (*Config, error) {
    cfg := &Config{}
    if err := load(cfg); err != nil {
        return nil, err
    }

    // Migration logic (unnecessary!)
    if cfg.Version == 1 {
        migrateV1ToV2(cfg)
    }

    return cfg, nil
}

// ✅ Good - direct and simple
type Config struct {
    UseTUI   bool   `json:"use_tui"`
    LogLevel string `json:"log_level"`
    NewField string `json:"new_field"`  // Just add it
}

func LoadConfig() (*Config, error) {
    cfg := &Config{}
    if err := load(cfg); err != nil {
        // Tell user to regenerate if structure changed
        return nil, fmt.Errorf("config error: %w (try deleting and regenerating)", err)
    }
    return cfg, nil
}
```

### Example 2: Changing Database Schema

```go
// ❌ Bad - migration system for pre-alpha
func ApplyMigrations(db *sql.DB) error {
    for _, m := range migrations {
        if err := m.Up(db); err != nil {
            return err
        }
    }
    return nil
}

// ✅ Good - single current schema
const Schema = `
CREATE TABLE IF NOT EXISTS config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT NOT NULL  -- Just added this, no migration
);
`

func InitDB(db *sql.DB) error {
    // If schema changed incompatibly, user deletes the DB file
    _, err := db.Exec(Schema)
    return err
}
```

### Example 3: Removing a Feature

```go
// ❌ Bad - deprecation ceremony in pre-alpha
// @deprecated Use NewFeature instead
func OldFeature() error {
    log.Warn("OldFeature is deprecated")
    return NewFeature()
}

// ✅ Good - just delete it
// (OldFeature has been removed - use NewFeature)
```

## Communication with Users

Since you're making breaking changes, communicate clearly:

### In Changelog/Releases

```markdown
## v0.2.0 (Pre-Alpha)

⚠️ BREAKING CHANGES (pre-alpha - expect these!):
- Config schema changed: delete `~/.config/myapp/config.json` and regenerate
- Database schema updated: delete `myapp.db` and reinitialize
- Renamed `--use-text-ui` flag to `--use-tui`

### New Features
- Added XYZ capability
```

### In Error Messages

```go
if err := loadConfig(); err != nil {
    return fmt.Errorf(
        "config load failed: %w\n\n" +
        "NOTE: Pre-alpha config format changed. Delete config and regenerate:\n" +
        "  rm ~/.config/myapp/config.json\n" +
        "  myapp config init",
        err,
    )
}
```

## When to Graduate from Pre-Alpha

Move to alpha/beta when:

1. **User Base**: You have external users (not just internal testing)
2. **Data Value**: Users have data they can't easily regenerate
3. **Breaking Change Pain**: Breaking changes cause significant user friction
4. **API Stability**: External tools/scripts depend on your interfaces

At that point:
- Start using migration systems for databases
- Version your APIs
- Provide deprecation periods for config changes
- Write upgrade guides for breaking changes

But until then: **stay lean, move fast, break things freely**.

## Summary

**Pre-alpha philosophy:**
- ✅ Make breaking changes directly
- ✅ Delete dead code completely
- ✅ Keep implementation simple
- ✅ No backwards compatibility for internal state
- ✅ Version only external interfaces (file formats, APIs, plugins)
- ✅ Optimize for velocity

**Remember:** Every line of backwards compatibility code is:
- More code to maintain
- More complexity to understand
- More surface area for bugs
- Slower iteration speed

In pre-alpha, **simplicity is speed**. Ship fast, learn fast, iterate fast.
