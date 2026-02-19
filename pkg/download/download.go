// SPDX-License-Identifier: Apache-2.0
package download

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/charmbracelet/log"
)

// ProgressCallback is called periodically during download with current progress
// percent is a float between 0 and 1 representing completion percentage
type ProgressCallback func(percent float64)

// Options configures the download
type Options struct {
	ProgressCallback ProgressCallback
	Headers          map[string]string
}

// File downloads a file from URL to destination with optional progress callback
func File(url, dest string, progressCallback ProgressCallback) error {
	return FileWithOptions(url, dest, &Options{
		ProgressCallback: progressCallback,
	})
}

// FileWithOptions downloads a file with custom options
func FileWithOptions(url, dest string, opts *Options) error {
	log.Debugf("Downloading %s to %s", url, dest)

	if opts == nil {
		opts = &Options{}
	}

	// Create HTTP client with default settings
	client := &http.Client{}

	// Create the HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add custom headers
	for key, value := range opts.Headers {
		req.Header.Set(key, value)
	}

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Create destination file
	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Get total size for progress tracking
	totalSize := resp.ContentLength
	downloaded := int64(0)

	// Create a progress reader if callback provided
	if opts.ProgressCallback != nil && totalSize > 0 {
		// Wrap the response body with progress tracking
		buf := make([]byte, 32*1024) // 32KB buffer
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				downloaded += int64(n)
				if _, writeErr := out.Write(buf[:n]); writeErr != nil {
					return fmt.Errorf("failed to write: %w", writeErr)
				}

				// Report progress
				percent := float64(downloaded) / float64(totalSize)
				opts.ProgressCallback(percent)
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("failed to read: %w", err)
			}
		}
	} else {
		// No progress tracking, just copy
		if _, err := io.Copy(out, resp.Body); err != nil {
			return fmt.Errorf("failed to save: %w", err)
		}
	}

	log.Debugf("Download complete: %s", dest)
	return nil
}
