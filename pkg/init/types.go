// SPDX-License-Identifier: Apache-2.0
package init

// InitSettings holds all collected settings across wizard tabs
type InitSettings struct {
	// Repo layout
	ArchiveLocation string // Local archive directory (default: "archive")

	// Tab 1: Signing Settings
	KeyName       string
	KeyEmail      string
	KeyExpiry     string
	KeyFormat     string // "armored" or "binary"
	HistoryFormat string // "armored" or "binary"
	KeyPassword   string // Used to encrypt private key

	// Tab 2: Key Generation (results)
	KeyGenerated  bool
	KeyPath       string
	PublicKeyPath string

	// Tab 3: File Generation (results)
	FilesCreated []string
}
