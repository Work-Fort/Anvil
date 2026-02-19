// SPDX-License-Identifier: Apache-2.0
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/viper"
)

const (
	// GitHub repository
	GitHubRepo = "Work-Fort/Anvil"
	GitHubAPI  = "https://api.github.com"

	// Firecracker GitHub repository
	FirecrackerRepo = "firecracker-microvm/firecracker"

	// Configuration
	EnvPrefix        = "ANVIL" // Environment variable prefix for Viper
	ConfigFileName   = "config"         // Config file name for XDG config dir (without extension)
	LocalConfigFile  = "anvil" // Config file name for current directory (without extension)
	ConfigType       = "yaml"           // Config file type
	DefaultConfigExt = ".yaml"          // Default config file extension
)

// Paths holds all XDG-compliant directory paths
type Paths struct {
	DataDir   string
	CacheDir  string
	ConfigDir string
	BinDir    string

	// Subdirectories
	KernelsDir     string
	FirecrackerDir string
	KernelBuildDir string // Kernel source build working directory (in cache)
	KeysDir        string // PGP keys directory
	GnupgDir       string // GPG keyring directory
}

var (
	// GlobalPaths is the global paths instance
	GlobalPaths *Paths
)

func init() {
	GlobalPaths = GetPaths()
}

// GetPaths returns XDG-compliant directory paths
func GetPaths() *Paths {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to get home directory: %v\n", err)
			os.Exit(1)
		}
		dataHome = filepath.Join(home, ".local", "share")
	}

	cacheHome := os.Getenv("XDG_CACHE_HOME")
	if cacheHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to get home directory: %v\n", err)
			os.Exit(1)
		}
		cacheHome = filepath.Join(home, ".cache")
	}

	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to get home directory: %v\n", err)
			os.Exit(1)
		}
		configHome = filepath.Join(home, ".config")
	}

	dataDir := filepath.Join(dataHome, "anvil")
	cacheDir := filepath.Join(cacheHome, "anvil")
	configDir := filepath.Join(configHome, "anvil")
	binDir := filepath.Join(dataDir, "bin")

	return &Paths{
		DataDir:        dataDir,
		CacheDir:       cacheDir,
		ConfigDir:      configDir,
		BinDir:         binDir,
		KernelsDir:     filepath.Join(dataDir, "kernels"),
		FirecrackerDir: filepath.Join(dataDir, "firecracker"),
		KernelBuildDir: filepath.Join(cacheDir, "build-kernel"),
		KeysDir:        filepath.Join(dataDir, "keys"),
		GnupgDir:       filepath.Join(dataDir, "gnupg"),
	}
}

// IsRepoMode returns true when a anvil.yaml exists in the current
// working directory, meaning the CLI is operating within a managed repository.
func IsRepoMode() bool {
	_, err := os.Stat(filepath.Join(".", LocalConfigFile+DefaultConfigExt))
	return err == nil
}

// InitDirs creates all necessary directories
func InitDirs() error {
	dirs := []string{
		GlobalPaths.ConfigDir,
		GlobalPaths.KernelsDir,
		GlobalPaths.FirecrackerDir,
		GlobalPaths.BinDir,
		GlobalPaths.CacheDir,
		GlobalPaths.KernelBuildDir,
		GlobalPaths.KeysDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// GnupgDir needs special permissions (0700 for GPG security requirements)
	if err := os.MkdirAll(GlobalPaths.GnupgDir, 0700); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", GlobalPaths.GnupgDir, err)
	}

	return nil
}

// GetArch returns the system architecture (x86_64 or aarch64)
func GetArch() (string, error) {
	arch := runtime.GOARCH
	switch arch {
	case "amd64":
		return "x86_64", nil
	case "arm64":
		return "aarch64", nil
	default:
		return "", fmt.Errorf("unsupported architecture: %s", arch)
	}
}

// GetKernelName returns the kernel binary name based on architecture
func GetKernelName() (string, error) {
	arch, err := GetArch()
	if err != nil {
		return "", err
	}

	switch arch {
	case "x86_64":
		return "vmlinux", nil
	case "aarch64":
		return "Image", nil
	default:
		return "", fmt.Errorf("unsupported architecture: %s", arch)
	}
}

// GetGitHubToken returns the GitHub token (respects full config precedence)
// Priority: ENV:ANVIL_GITHUB_TOKEN > user config > defaults
func GetGitHubToken() string {
	return viper.GetString("github-token")
}
