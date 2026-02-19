// SPDX-License-Identifier: Apache-2.0
package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/download"
	"github.com/hashicorp/go-version"
)

// Release represents a GitHub release
type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// Asset represents a GitHub release asset
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Client handles GitHub API requests
type Client struct {
	token string
}

// NewClient creates a new GitHub API client
func NewClient() *Client {
	return &Client{
		token: config.GetGitHubToken(),
	}
}

// GetLatestRelease fetches the latest release for a repository
func (c *Client) GetLatestRelease(owner, repo string) (*Release, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", config.GitHubAPI, owner, repo)
	return c.getRelease(url)
}

// GetReleaseByTag fetches a specific release by tag
func (c *Client) GetReleaseByTag(owner, repo, tag string) (*Release, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/tags/%s", config.GitHubAPI, owner, repo, tag)
	return c.getRelease(url)
}

// GetReleases fetches multiple releases for a repository
func (c *Client) GetReleases(owner, repo string, perPage int) ([]Release, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases?per_page=%d", config.GitHubAPI, owner, repo, perPage)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.DoRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}

	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to decode releases: %w", err)
	}

	return releases, nil
}

// getRelease is a helper to fetch a single release
func (c *Client) getRelease(url string) (*Release, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.DoRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode release: %w", err)
	}

	return &release, nil
}

// DownloadFile downloads a file from a URL with automatic GitHub token injection
func (c *Client) DownloadFile(url, dest string, progressCallback download.ProgressCallback) error {
	opts := &download.Options{
		ProgressCallback: progressCallback,
	}

	if c.token != "" {
		opts.Headers = map[string]string{
			"Authorization": "token " + c.token,
		}
	}

	return download.FileWithOptions(url, dest, opts)
}

// DoRequest executes an HTTP request with automatic GitHub token injection
func (c *Client) DoRequest(req *http.Request) (*http.Response, error) {
	if c.token != "" {
		req.Header.Set("Authorization", "token "+c.token)
	}

	return http.DefaultClient.Do(req)
}

// StripVersionPrefix removes 'v' prefix from version strings
func StripVersionPrefix(version string) string {
	return strings.TrimPrefix(version, "v")
}

// SortReleasesBySemver sorts releases by semantic version in descending order (newest first)
func SortReleasesBySemver(releases []Release) []Release {
	sorted := make([]Release, len(releases))
	copy(sorted, releases)

	sort.Slice(sorted, func(i, j int) bool {
		// Strip 'v' prefix and parse versions
		v1, err1 := version.NewVersion(StripVersionPrefix(sorted[i].TagName))
		v2, err2 := version.NewVersion(StripVersionPrefix(sorted[j].TagName))

		// If either version is invalid, fall back to string comparison
		if err1 != nil || err2 != nil {
			return sorted[i].TagName > sorted[j].TagName
		}

		// Compare semantic versions (descending order)
		return v1.GreaterThan(v2)
	})

	return sorted
}
