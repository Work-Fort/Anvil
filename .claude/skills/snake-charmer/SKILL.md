---
name: snake-charmer
description: Expert guidance for integrating Cobra and Viper for CLI configuration management with proper precedence, separation of concerns, and best practices
---

# Snake Charmer: Cobra + Viper Integration Guide

Expert patterns for building production-grade CLI applications with Cobra (commands) and Viper (configuration) in Go.

## Core Principles

### 1. Configuration Precedence (Highest to Lowest)
```
Environment Variables > Local Config File > User Config File > Defaults
```

**Why this order?**
- ENV: Highest - enables override in CI/CD, containers, one-off commands
- Local: Project-specific settings (`./<appname>.yaml`)
- User: Personal preferences (`~/.config/<appname>/config.yaml`)
- Defaults: Fallback values hardcoded in application

### 2. Separation of Concerns

**Three-layer architecture:**

```
pkg/config/
├── config.go       # Constants, paths, XDG setup
├── viper.go        # Viper initialization for global app config
└── operations.go   # Business logic: SetConfig, GetConfig, etc.

cmd/config/
├── set.go          # Thin wrapper: arg parsing + output
├── get.go          # Thin wrapper: arg parsing + output
└── list.go         # Thin wrapper: arg parsing + output
```

**DO:**
- Keep Viper initialization in `pkg/config/viper.go` (for app-wide config)
- Keep config operations in `pkg/config/operations.go` (business logic)
- Use Viper instances (not global) for isolated operations
- CMD layer calls pkg/config functions and formats output
- Return native Go types from pkg/config (structs, enums)

**DON'T:**
- Mix Viper calls directly in cmd files
- Use global Viper instance for write operations
- Put business logic in cmd layer

**Why this matters:**
- Business logic in `pkg/config` can be called from CLI, TUI, or API
- Viper instances provide isolation and testability
- CMD layer is pure translation (flags → Go types, results → text)

## Implementation Patterns

### Pattern 1: Viper Initialization in Config Package

**pkg/config/viper.go:**
```go
package config

import (
	"fmt"
	"path/filepath"
	"strings"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// InitViper sets up Viper with defaults and environment variable support
// Called once during application init(), before directories are created
func InitViper() {
	// Set config file type
	viper.SetConfigType(ConfigType) // e.g., "yaml"

	// Set defaults (lowest precedence)
	viper.SetDefault("example-flag", true)
	viper.SetDefault("another-setting", "default-value")

	// Enable environment variable support (highest precedence)
	// Maps "example-flag" to "APPNAME_EXAMPLE_FLAG"
	viper.SetEnvPrefix(EnvPrefix) // e.g., "MYAPP"
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	// Note: Config file reading deferred until after directory initialization
}

// LoadConfig reads config files in precedence order
// Precedence: ENV > ./myapp.yaml > ~/.config/myapp/config.yaml > defaults
func LoadConfig() error {
	// First: Read user config from XDG directory
	viper.SetConfigName(UserConfigFileName) // e.g., "config"
	viper.AddConfigPath(GlobalPaths.ConfigDir)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read user config: %w", err)
		}
		// Config file not found is OK - use defaults
	}

	// Second: Merge local project config (overrides user config)
	viper.SetConfigName(LocalConfigFileName) // e.g., "myapp"
	viper.AddConfigPath(".")

	if err := viper.MergeInConfig(); err != nil {
		// Ignore if local config doesn't exist
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to merge local config: %w", err)
		}
	}

	// Environment variables automatically override all file configs due to AutomaticEnv()
	return nil
}

// BindFlags binds all application flags to Viper
func BindFlags(flags *pflag.FlagSet) error {
	flagsToBind := []string{
		"example-flag",
		"another-setting",
		// Add more flags here as they're added to the application
	}

	for _, flagName := range flagsToBind {
		if err := viper.BindPFlag(flagName, flags.Lookup(flagName)); err != nil {
			return fmt.Errorf("failed to bind flag %s: %w", flagName, err)
		}
	}

	return nil
}

// GetExampleFlag returns the example-flag value (respects full precedence)
func GetExampleFlag() bool {
	return viper.GetBool("example-flag")
}
```

### Pattern 2: Root Command Integration

**cmd/root.go:**
```go
package cmd

import (
	"github.com/spf13/cobra"
	"yourapp/pkg/config"
)

var (
	exampleFlag bool
)

var rootCmd = &cobra.Command{
	Use:   "myapp",
	Short: "Application description",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize directories
		if err := config.InitDirs(); err != nil {
			return err
		}

		// Load config files (now that directories exist)
		if err := config.LoadConfig(); err != nil {
			return err
		}

		// Update flag variables from Viper (respects precedence)
		exampleFlag = config.GetExampleFlag()

		return nil
	},
}

func init() {
	// Initialize Viper BEFORE adding flags
	config.InitViper()

	// Add flags
	rootCmd.PersistentFlags().BoolVar(&exampleFlag, "example-flag", true, "Example flag description")

	// Bind flags to Viper
	config.BindFlags(rootCmd.PersistentFlags())
}
```

### Pattern 3: Business Logic in pkg/config

**Best Practice: Separate business logic from CLI layer**

Business logic should live in `pkg/config/operations.go` using Viper instances for isolation.

**pkg/config/operations.go:**
```go
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"github.com/spf13/viper"
)

// ConfigScope indicates whether to operate on local or global config
type ConfigScope int

const (
	ScopeLocal ConfigScope = iota
	ScopeGlobal
)

// ConfigValue represents a configuration key-value pair with its source
type ConfigValue struct {
	Key    string
	Value  interface{}
	Source string
}

// SetConfigValue sets a configuration value in the specified scope
// Uses a Viper instance for isolation (not global viper)
func SetConfigValue(key, valueStr string, scope ConfigScope) error {
	configPath := getConfigPath(scope)

	// Create isolated Viper instance for this operation
	v := viper.New()
	v.SetConfigType(ConfigType)
	v.SetConfigFile(configPath)

	// Read existing config if it exists
	_ = v.ReadInConfig() // Ignore error if file doesn't exist

	// Parse and set value
	value := parseValue(valueStr)
	v.Set(key, value)

	// Write config using safe pattern
	// SafeWriteConfigAs first (creates if missing)
	if err := v.SafeWriteConfigAs(configPath); err != nil {
		// If file exists, overwrite with WriteConfigAs
		if _, ok := err.(viper.ConfigFileAlreadyExistsError); ok {
			if err := v.WriteConfigAs(configPath); err != nil {
				return fmt.Errorf("failed to write config: %w", err)
			}
		} else {
			return fmt.Errorf("failed to create config: %w", err)
		}
	}

	return nil
}

// GetConfigValue retrieves a configuration value and its source
// Uses global Viper for reading (respects full precedence chain)
func GetConfigValue(key string) (*ConfigValue, error) {
	if !viper.IsSet(key) {
		return nil, fmt.Errorf("configuration key not found: %s", key)
	}

	return &ConfigValue{
		Key:    key,
		Value:  viper.Get(key),
		Source: getConfigSource(key),
	}, nil
}

// ListConfigValues returns all configuration values with their sources
func ListConfigValues() ([]ConfigValue, error) {
	settings := viper.AllSettings()
	if len(settings) == 0 {
		return []ConfigValue{}, nil
	}

	keys := flattenKeys(settings, "")
	sort.Strings(keys)

	values := make([]ConfigValue, 0, len(keys))
	for _, key := range keys {
		values = append(values, ConfigValue{
			Key:    key,
			Value:  viper.Get(key),
			Source: getConfigSource(key),
		})
	}

	return values, nil
}

func getConfigPath(scope ConfigScope) string {
	if scope == ScopeGlobal {
		return filepath.Join(GlobalPaths.ConfigDir, ConfigFileName+".yaml")
	}
	return "./myapp.yaml"
}
```

**Why Viper instances?**
- **Isolation**: Each write operation uses its own instance
- **Testability**: Easier to test without global state pollution
- **Safety**: Write operations don't affect app-wide config
- **Best Practice**: Recommended in [Viper docs](https://github.com/spf13/viper#working-with-multiple-vipers) and [industry guides](https://oneuptime.com/blog/post/2026-01-07-go-viper-configuration/view)

### Pattern 4: Thin CLI Wrappers

**cmd/config/set.go (thin wrapper):**
```go
package config

import (
	"fmt"
	"github.com/spf13/cobra"
	"yourapp/pkg/config"
)

var globalFlag bool

func newSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set [key] [value]",
		Short: "Set configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]

			// Translate CLI flag to Go type
			scope := config.ScopeLocal
			if globalFlag {
				scope = config.ScopeGlobal
			}

			// Call business logic
			if err := config.SetConfigValue(key, value, scope); err != nil {
				return err
			}

			// Format output
			fmt.Printf("Set %s = %s\n", key, value)
			return nil
		},
	}

	cmd.Flags().BoolVar(&globalFlag, "global", false, "Set in user config")
	return cmd
}
```

**Benefits:**
- CMD layer is ~20 lines (translation only)
- Business logic reusable from TUI, API, tests
- Clear separation of concerns
- Native Go types (enums, structs) instead of strings/bools

### Pattern 5: XDG Base Directory Compliance

**pkg/config/config.go:**
```go
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// Application name
	AppName = "myapp"

	// Configuration
	EnvPrefix          = "MYAPP"        // Environment variable prefix
	UserConfigFileName = "config"       // User config: ~/.config/myapp/config.yaml
	LocalConfigFileName = "myapp"       // Local config: ./myapp.yaml
	ConfigType         = "yaml"         // Config file format
)

// Paths holds XDG-compliant directory paths
type Paths struct {
	ConfigDir string // XDG_CONFIG_HOME/myapp
	DataDir   string // XDG_DATA_HOME/myapp
	CacheDir  string // XDG_CACHE_HOME/myapp
}

var GlobalPaths *Paths

func init() {
	GlobalPaths = GetPaths()
}

// GetPaths returns XDG-compliant directory paths
func GetPaths() *Paths {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, _ := os.UserHomeDir()
		configHome = filepath.Join(home, ".config")
	}

	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, _ := os.UserHomeDir()
		dataHome = filepath.Join(home, ".local", "share")
	}

	cacheHome := os.Getenv("XDG_CACHE_HOME")
	if cacheHome == "" {
		home, _ := os.UserHomeDir()
		cacheHome = filepath.Join(home, ".cache")
	}

	return &Paths{
		ConfigDir: filepath.Join(configHome, AppName),
		DataDir:   filepath.Join(dataHome, AppName),
		CacheDir:  filepath.Join(cacheHome, AppName),
	}
}

// InitDirs creates all necessary directories
func InitDirs() error {
	dirs := []string{
		GlobalPaths.ConfigDir,
		GlobalPaths.DataDir,
		GlobalPaths.CacheDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}
```

## Common Patterns and Best Practices

### ✅ DO: Use SetEnvKeyReplacer for Hyphenated Keys

```go
// Maps "use-tui" to "MYAPP_USE_TUI"
viper.SetEnvPrefix("MYAPP")
viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
viper.AutomaticEnv()
```

Without this, `use-tui` won't map to `MYAPP_USE_TUI` - it would only check `MYAPP_use-tui`.

### ✅ DO: Use MergeInConfig for Layered Configs

```go
// Read base config
viper.SetConfigName("config")
viper.AddConfigPath(userConfigDir)
viper.ReadInConfig()

// Merge in overrides (preserves existing values)
viper.SetConfigName("myapp")
viper.AddConfigPath(".")
viper.MergeInConfig() // Overlay, don't replace
```

`ReadInConfig()` replaces everything; `MergeInConfig()` only overrides keys present in new config.

### ✅ DO: Use RunE for Error Handling

```go
// Good - proper error handling
RunE: func(cmd *cobra.Command, args []string) error {
	if err := doSomething(); err != nil {
		return err
	}
	return nil
}

// Bad - no error handling
Run: func(cmd *cobra.Command, args []string) {
	doSomething() // Error ignored
}
```

### ✅ DO: Validate Arguments with Cobra Validators

```go
// Exactly 2 arguments required
Args: cobra.ExactArgs(2)

// At least 1 argument
Args: cobra.MinimumNArgs(1)

// Range of arguments
Args: cobra.RangeArgs(1, 3)

// Custom validation
Args: func(cmd *cobra.Command, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("requires exactly 2 arguments")
	}
	return nil
}
```

### ✅ DO: Centralize Flag Binding

```go
// In pkg/config/viper.go
func BindFlags(flags *pflag.FlagSet) error {
	flagsToBind := []string{
		"use-tui",
		"log-level",
	}

	for _, flagName := range flagsToBind {
		if err := viper.BindPFlag(flagName, flags.Lookup(flagName)); err != nil {
			return err
		}
	}
	return nil
}
```

When adding a new flag, just add it to the slice - no changes needed in cmd files.

### ✅ DO: Handle Sensitive Configuration Securely

**Best Practice from [2026 guide](https://oneuptime.com/blog/post/2026-01-07-go-viper-configuration/view):**
> "Never log passwords or API keys. Consider using remote config stores for secrets."

```go
// Good - mask secrets in logs and output
func listConfigValues() {
	for key, value := range viper.AllSettings() {
		if isSensitive(key) {
			fmt.Printf("%s = ******* (masked)\n", key)
		} else {
			fmt.Printf("%s = %v\n", key, value)
		}
	}
}

func isSensitive(key string) bool {
	sensitive := []string{"password", "token", "secret", "key", "api-key"}
	keyLower := strings.ToLower(key)
	for _, s := range sensitive {
		if strings.Contains(keyLower, s) {
			return true
		}
	}
	return false
}

// Good - prefer environment variables for secrets
viper.SetDefault("api-token", "") // No default for secrets
if token := os.Getenv("MYAPP_API_TOKEN"); token == "" {
	log.Fatal("MYAPP_API_TOKEN environment variable required")
}

// Good - warn about secrets in config files
func ValidateConfig() error {
	if viper.ConfigFileUsed() != "" {
		for _, key := range getConfigKeys() {
			if isSensitive(key) && viper.InConfig(key) {
				log.Warnf("Sensitive key '%s' found in config file. Consider using ENV instead: %s",
					key, keyToEnvVar(key))
			}
		}
	}
	return nil
}
```

### ✅ DO: Use SetDefault for All Configurable Keys

```go
// In InitViper() - set defaults for ALL keys users might configure
viper.SetDefault("use-tui", true)
viper.SetDefault("log-level", "info")
viper.SetDefault("auto-update", false)
// ... all other configurable keys
```

**Why this matters:**
- Application works fine without any config files
- Users can always reset to defaults by removing keys
- `GetBool()`, `GetString()` etc. always return sensible values
- Follows [Twelve-Factor App](https://12factor.net/config) principles

**Best Practice from [2026 guide](https://oneuptime.com/blog/post/2026-01-07-go-viper-configuration/view):**
> "Always set defaults for all configuration values to ensure your application can start even without a config file."

### ❌ DON'T: Call viper.ReadInConfig() Before Directory Creation

```go
// Bad - directory might not exist yet
func init() {
	config.InitViper()
	viper.ReadInConfig() // Will fail if ~/.config/myapp doesn't exist
}

// Good - defer reading until directories are created
func init() {
	config.InitViper() // Sets up defaults, env vars, search paths
}

func Execute() {
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		config.InitDirs()        // Create directories first
		config.LoadConfig()      // Then read config files
		return nil
	}
}
```

### ❌ DON'T: Use String Comparisons for Boolean Config

```go
// Bad - fragile, doesn't respect type
if viper.GetString("config-save") != "false" {
	saveConfig()
}

// Good - uses proper type
if viper.GetBool("config-save") {
	saveConfig()
}
```

Viper handles type conversion automatically if you use the right getter.

### ❌ DON'T: Mix Viper Logic in cmd Files

```go
// Bad - Viper instance creation and logic in cmd layer
func setConfig(key, value string) error {
	v := viper.New()
	v.SetConfigFile(configPath)
	v.Set(key, value)
	return v.WriteConfigAs(configPath) // Business logic in cmd!
}

// Good - business logic in pkg/config, cmd calls it
func setConfig(key, value string) error {
	return config.SetConfigValue(key, value, config.ScopeLocal)
}
```

**Why this matters:**
- CMD layer only handles: arg parsing, scope mapping, output formatting
- `pkg/config` handles: Viper operations, file I/O, parsing, validation
- Enables reuse from TUI, API, tests - not just CLI

### ❌ DON'T: Forget to Handle ConfigFileNotFoundError

```go
// Bad - treats "no config file" as error
if err := viper.ReadInConfig(); err != nil {
	return err // Fails on first run when config doesn't exist
}

// Good - distinguishes between "not found" and "real error"
if err := viper.ReadInConfig(); err != nil {
	if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
		return err // Only return non-notfound errors
	}
}
```

### ❌ DON'T: Auto-Save Config Files

**Anti-Pattern: Implicit Config Persistence**

```go
// Bad - auto-saves config on first run or when values change
func LoadConfig() error {
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Auto-create config with defaults
			if viper.GetBool("config-save") {
				viper.SafeWriteConfigAs(configPath)
			}
		}
	}
	return nil
}

// Also bad - tracking "was it newly created" to decide on save
func LoadConfig() (configFileNotFound bool, err error) {
	// Returns whether to auto-save...
}
```

**Why This is Bad:**

1. **Surprise File Creation**: Users don't expect commands to create config files in their home directory or current directory
2. **Unclear Behavior**: "Which commands create files? When? Where?"
3. **Configuration Complexity**: Needs a `config-save` flag to control auto-save behavior, adding another config layer
4. **Debugging Nightmare**: Users wonder why config files appear unexpectedly
5. **Violation of Unix Philosophy**: Tools should do nothing unless explicitly asked

**Good Pattern: Explicit Config Commands**

Use explicit `config set/get/list` commands instead. See Pattern 3 and Pattern 4 for the recommended instance-based implementation.

**Benefits of Explicit Pattern:**

- ✅ **Predictable**: Config files only exist when user runs `myapp config set`
- ✅ **No Surprises**: No unexpected file creation
- ✅ **Simpler Mental Model**: "Config files = explicit user action"
- ✅ **Better UX**: `myapp config set use-tui false` is self-documenting
- ✅ **Cleaner Code**: No auto-save logic, no `config-save` flag needed
- ✅ **Defaults Work**: Application works fine with no config files via `viper.SetDefault()`

**Migration Path:** If you have auto-save logic, remove it and implement explicit `config set/get/list` commands per Pattern 3 and 4.

## Testing Configuration

### Unit Test Pattern

```go
func TestConfigPrecedence(t *testing.T) {
	// Save original ENV
	origEnv := os.Getenv("MYAPP_USE_TUI")
	defer os.Setenv("MYAPP_USE_TUI", origEnv)

	// Set up test config
	viper.Reset()
	viper.SetDefault("use-tui", true)
	viper.SetEnvPrefix("MYAPP")
	viper.AutomaticEnv()

	// Test ENV override
	os.Setenv("MYAPP_USE_TUI", "false")
	if viper.GetBool("use-tui") != false {
		t.Error("ENV variable should override default")
	}
}
```

## Config Command Checklist

When implementing a full `config` command, ensure:

- [ ] `config get [key]` - Retrieve value, respects precedence
- [ ] `config set [key] [value]` - Set value
- [ ] `config unset [key]` - Remove value
- [ ] `config list` - Show all configuration with sources
- [ ] `config edit` - Open editor for config file
- [ ] `--global` flag on set/unset/edit (default to local)
- [ ] `--file <path>` flag for custom config file
- [ ] Dot notation support for nested keys (`ui.theme`)
- [ ] Boolean value parsing (true/false/yes/no/enable/disable)
- [ ] Show precedence in `list` output (via `viper.ConfigFileUsed()`)

## Advanced Patterns

### Pattern: Config File Precedence Display

```go
func showConfigSource(key string) {
	value := viper.Get(key)

	// Check if from ENV
	envKey := strings.ToUpper(viper.GetEnvPrefix() + "_" +
		strings.ReplaceAll(key, "-", "_"))
	if os.Getenv(envKey) != "" {
		fmt.Printf("%s = %v (from ENV: %s)\n", key, value, envKey)
		return
	}

	// Check if from config file
	if viper.ConfigFileUsed() != "" {
		fmt.Printf("%s = %v (from %s)\n", key, value, viper.ConfigFileUsed())
		return
	}

	// Default
	fmt.Printf("%s = %v (default)\n", key, value)
}
```

### Pattern: Nested Configuration with Struct Unmarshal

**Best Practice from [2026 guide](https://oneuptime.com/blog/post/2026-01-07-go-viper-configuration/view):**
> "Prefer Unmarshal into structs over individual Get calls for better type safety and organization."

```go
// For config like:
// server:
//   host: localhost
//   port: 8080
//   timeout: 30s

type ServerConfig struct {
	Host    string        `mapstructure:"host"`
	Port    int           `mapstructure:"port"`
	Timeout time.Duration `mapstructure:"timeout"`
}

// Good - type-safe struct unmarshaling
func GetServerConfig() (ServerConfig, error) {
	var cfg ServerConfig
	if err := viper.UnmarshalKey("server", &cfg); err != nil {
		return ServerConfig{}, fmt.Errorf("failed to unmarshal server config: %w", err)
	}
	return cfg, nil
}

// Avoid - individual Get calls (error-prone, verbose)
func GetServerConfig() ServerConfig {
	return ServerConfig{
		Host:    viper.GetString("server.host"),
		Port:    viper.GetInt("server.port"),
		Timeout: viper.GetDuration("server.timeout"),
	}
}
```

**Benefits of struct unmarshaling:**
- **Type Safety**: Compilation checks field types
- **Validation**: Can add struct tags for validation
- **Maintainability**: Config structure is explicit
- **Less Error-Prone**: Typos in keys caught at unmarshal time

### Pattern: Required vs Optional Config with Validation

**Best Practice from [2026 guide](https://oneuptime.com/blog/post/2026-01-07-go-viper-configuration/view):**
> "Always validate your configuration after loading to catch errors early."

```go
func ValidateConfig() error {
	// Check required keys
	required := []string{"database.url", "api.key"}
	for _, key := range required {
		if !viper.IsSet(key) {
			return fmt.Errorf("required config key missing: %s", key)
		}
	}

	// Validate value constraints
	if port := viper.GetInt("server.port"); port < 1 || port > 65535 {
		return fmt.Errorf("invalid server.port: %d (must be 1-65535)", port)
	}

	// Validate enums
	logLevel := viper.GetString("log-level")
	validLevels := []string{"debug", "info", "warn", "error"}
	if !contains(validLevels, logLevel) {
		return fmt.Errorf("invalid log-level: %s (must be one of: %v)", logLevel, validLevels)
	}

	return nil
}

// Call in PersistentPreRunE after LoadConfig()
func (cmd *cobra.Command) PersistentPreRunE(args []string) error {
	config.InitDirs()
	config.LoadConfig()

	// Validate before proceeding
	if err := config.ValidateConfig(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	return nil
}
```

**Why validate early?**
- Fail fast with clear error messages
- Catch misconfigurations before runtime failures
- Better UX than cryptic errors deep in execution

## Migration Guide: Adding Viper to Existing Cobra App

1. **Install Viper**: `go get github.com/spf13/viper`

2. **Create config package**:
   ```bash
   mkdir -p pkg/config
   touch pkg/config/config.go pkg/config/viper.go
   ```

3. **Move constants to config package** (app name, paths, env prefix)

4. **Implement InitViper()** with defaults and AutomaticEnv()

5. **Implement LoadConfig()** with ReadInConfig() and MergeInConfig()

6. **Update root command**:
   - Add `config.InitViper()` to `init()`
   - Add `config.LoadConfig()` to `PersistentPreRunE`
   - Replace direct flag access with `config.GetXXX()` functions

7. **Bind flags**: Call `config.BindFlags(rootCmd.PersistentFlags())`

8. **Test precedence**: Verify ENV > local > user > defaults works

## Troubleshooting

### Issue: Environment Variables Not Working

**Symptom**: Setting `MYAPP_USE_TUI=false` doesn't override config

**Solutions**:
1. Check `SetEnvPrefix()` is called
2. Check `AutomaticEnv()` is called
3. Add `SetEnvKeyReplacer(strings.NewReplacer("-", "_"))` for hyphenated keys
4. Verify env var name matches: `PREFIX_KEY_WITH_UNDERSCORES`

### Issue: Config File Not Found

**Symptom**: `viper.ReadInConfig()` returns error on first run

**Solution**: Handle `ConfigFileNotFoundError` - see "DON'T: Forget to Handle ConfigFileNotFoundError" above.

### Issue: Local Config Not Overriding User Config

**Symptom**: User config wins over local config

**Solution**: Use `MergeInConfig()` for local config, not `ReadInConfig()`:
```go
viper.ReadInConfig()       // User config
viper.MergeInConfig()      // Local config (overlays)
```

### Issue: Flag Changes Not Persisting

**Symptom**: `--flag=value` works but doesn't save to config

**Solution**: Flags don't auto-save. Implement explicit `config set`:
```go
// In PersistentPreRunE, after loading config
flagValue := config.GetFlag()  // Gets from precedence chain

// To save: use config set command
myapp config set flag-name value
```

## Summary

**The Snake Charmer pattern ensures**:
1. ✅ Proper configuration precedence (ENV > local > user > defaults)
2. ✅ Clean separation of concerns (config pkg vs cmd pkg)
3. ✅ Business logic in pkg/config with native Go types
4. ✅ Viper instances for isolated config operations
5. ✅ XDG Base Directory compliance
6. ✅ Robust error handling
7. ✅ Type-safe config access
8. ✅ Maintainable flag binding
9. ✅ Git-style config command support
10. ✅ Environment variable override support

By following these patterns, you'll build CLI applications that behave like professional tools (git, kubectl, docker) with predictable, powerful configuration management.

## References

This guide incorporates best practices from:

### Official Documentation
- [GitHub - spf13/viper: Go configuration with fangs](https://github.com/spf13/viper)
- [viper package - pkg.go.dev](https://pkg.go.dev/github.com/spf13/viper)

### Best Practices (2026)
- [How to Manage Configuration in Go with Viper](https://oneuptime.com/blog/post/2026-01-07-go-viper-configuration/view) - Modern patterns including instance-based approach, validation, and Twelve-Factor App compliance
- [Structuring Viper Config Files in Golang](https://tillitsdone.com/blogs/viper-config-file-best-practices/) - File organization and precedence patterns
- [Handling Go configuration with Viper - LogRocket Blog](https://blog.logrocket.com/handling-go-configuration-viper/) - Integration patterns with Cobra

### Known Issues & Solutions
- [Viper Issue #1514: WriteConfig() behaviour with SetConfigFile()](https://github.com/spf13/viper/issues/1514) - Why to use `*As` variants
- [Viper Issue #430: Config file initialization](https://github.com/spf13/viper/issues/430) - SafeWriteConfig vs WriteConfig patterns

### Additional Resources
- [Managing configuration in a Go application using Viper](https://reintech.io/blog/managing-configuration-go-application-viper-tutorial)
- [Configuration Management in Go with Viper](https://itnext.io/configuration-management-in-go-67234eaed35d)
