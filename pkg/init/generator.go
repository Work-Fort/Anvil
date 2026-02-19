// SPDX-License-Identifier: Apache-2.0
package init

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// GenerateRepoFiles creates all repository files atomically.
// It returns a list of created files on success, or rolls back all changes on error.
func GenerateRepoFiles(settings InitSettings) ([]string, error) {
	var createdItems []string // Track files and directories for rollback

	// Helper function to track created items
	trackCreated := func(path string) {
		createdItems = append(createdItems, path)
	}

	// Rollback function to clean up on error
	rollback := func() {
		// Delete in reverse order
		for i := len(createdItems) - 1; i >= 0; i-- {
			os.RemoveAll(createdItems[i])
		}
	}

	// Create directories
	dirs := []string{
		"configs",
		"keys",
		"keys/history",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			rollback()
			return nil, fmt.Errorf("failed to create directory %s (rolled back): %w", dir, err)
		}
		trackCreated(dir)
	}

	// Generate anvil.yaml from template
	tmpl, err := template.New("repo").Parse(RepoConfigTemplate)
	if err != nil {
		rollback()
		return nil, fmt.Errorf("failed to parse repo config template (rolled back): %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, settings); err != nil {
		rollback()
		return nil, fmt.Errorf("failed to execute repo config template (rolled back): %w", err)
	}

	// Write anvil.yaml
	repoConfigPath := "anvil.yaml"
	if err := os.WriteFile(repoConfigPath, buf.Bytes(), 0644); err != nil {
		rollback()
		return nil, fmt.Errorf("failed to write %s (rolled back): %w", repoConfigPath, err)
	}
	trackCreated(repoConfigPath)

	// Write .gitignore (templated so archive location is included)
	gitignorePath := ".gitignore"
	gitignoreTmpl, err := template.New("gitignore").Parse(GitignoreTemplate)
	if err != nil {
		rollback()
		return nil, fmt.Errorf("failed to parse gitignore template (rolled back): %w", err)
	}
	buf.Reset()
	if err := gitignoreTmpl.Execute(&buf, settings); err != nil {
		rollback()
		return nil, fmt.Errorf("failed to execute gitignore template (rolled back): %w", err)
	}
	if err := os.WriteFile(gitignorePath, buf.Bytes(), 0644); err != nil {
		rollback()
		return nil, fmt.Errorf("failed to write %s (rolled back): %w", gitignorePath, err)
	}
	trackCreated(gitignorePath)

	// Write kernel config files
	kernelConfigs := map[string]string{
		filepath.Join("configs", "kernel-x86_64.config"):  X86ConfigTemplate,
		filepath.Join("configs", "kernel-aarch64.config"): Aarch64ConfigTemplate,
	}

	for path, content := range kernelConfigs {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			rollback()
			return nil, fmt.Errorf("failed to write %s (rolled back): %w", path, err)
		}
		trackCreated(path)
	}

	// Return only the files (not directories) that were created
	files := []string{
		repoConfigPath,
		gitignorePath,
		filepath.Join("configs", "kernel-x86_64.config"),
		filepath.Join("configs", "kernel-aarch64.config"),
	}

	return files, nil
}
