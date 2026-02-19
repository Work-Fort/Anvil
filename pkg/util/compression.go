// SPDX-License-Identifier: Apache-2.0
package util

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/ulikunitz/xz"
)

// CompressXZ compresses a file using xz compression
func CompressXZ(src, dst string) error {
	return CompressXZWithProgress(src, dst, nil)
}

// CompressXZWithProgress compresses a file with progress tracking
func CompressXZWithProgress(src, dst string, progressCallback func(float64)) error {
	log.Debugf("Compressing %s to %s", src, dst)

	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Get source file size for progress tracking
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get source file info: %w", err)
	}
	uncompressedSize := srcInfo.Size()

	// Create destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	// Create xz writer with maximum compression
	xzWriter, err := xz.NewWriter(dstFile)
	if err != nil {
		return fmt.Errorf("failed to create xz writer: %w", err)
	}
	defer xzWriter.Close()

	// Wrap source file with progress reader if callback provided
	var reader io.Reader = srcFile
	if progressCallback != nil {
		reader = &progressReader{
			reader:   srcFile,
			total:    uncompressedSize,
			read:     0,
			callback: progressCallback,
			lastPct:  -1.0,
		}
	}

	// Copy data through xz compressor
	if _, err := io.Copy(xzWriter, reader); err != nil {
		return fmt.Errorf("failed to compress file: %w", err)
	}

	// Ensure all data is flushed
	if err := xzWriter.Close(); err != nil {
		return fmt.Errorf("failed to flush compressed data: %w", err)
	}

	log.Debugf("Successfully compressed %s to %s", src, dst)
	return nil
}

// DecompressXZ decompresses an xz file to a destination path
func DecompressXZ(src, dst string) error {
	return DecompressXZWithProgress(src, dst, nil)
}

// progressReader wraps a reader to track bytes read
type progressReader struct {
	reader   io.Reader
	total    int64
	read     int64
	callback func(float64)
	lastPct  float64
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.read += int64(n)

	if pr.callback != nil && pr.total > 0 {
		pct := float64(pr.read) / float64(pr.total)
		if pct > 1.0 {
			pct = 1.0
		}
		// Report every 1% for smoother progress updates
		if pct-pr.lastPct >= 0.01 || pct >= 0.99 {
			pr.callback(pct)
			pr.lastPct = pct
		}
	}

	return n, err
}

// DecompressXZWithProgress decompresses an xz file with progress tracking
func DecompressXZWithProgress(src, dst string, progressCallback func(float64)) error {
	log.Debugf("Decompressing %s to %s", src, dst)

	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Get source file size for progress tracking
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get source file info: %w", err)
	}
	compressedSize := srcInfo.Size()

	// Wrap source file with progress reader to track compressed bytes read
	var reader io.Reader = srcFile
	if progressCallback != nil {
		reader = &progressReader{
			reader:   srcFile,
			total:    compressedSize,
			read:     0,
			callback: progressCallback,
			lastPct:  -1.0,
		}
	}

	// Create xz reader
	xzReader, err := xz.NewReader(reader)
	if err != nil {
		return fmt.Errorf("failed to create xz reader: %w", err)
	}

	// Create destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	// Decompress
	if _, err := io.Copy(dstFile, xzReader); err != nil {
		return fmt.Errorf("failed to decompress: %w", err)
	}

	// Ensure 100% is reported
	if progressCallback != nil {
		progressCallback(1.0)
	}

	log.Debugf("Successfully decompressed to %s", dst)
	return nil
}

// ExtractTarGz extracts a tar.gz archive to a destination directory
func ExtractTarGz(src, dstDir string) error {
	log.Debugf("Extracting %s to %s", src, dstDir)

	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer srcFile.Close()

	// Create gzip reader
	gzReader, err := gzip.NewReader(srcFile)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzReader)

	// Extract files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Construct destination path
		target := filepath.Join(dstDir, header.Name)

		// Security check: prevent path traversal
		cleanTarget := filepath.Clean(target)
		cleanDstDir := filepath.Clean(dstDir) + string(filepath.Separator)
		if !strings.HasPrefix(cleanTarget+string(filepath.Separator), cleanDstDir) && cleanTarget != filepath.Clean(dstDir) {
			return fmt.Errorf("invalid path in archive: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}

		case tar.TypeReg:
			// Create parent directory if needed
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			// Create file
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}

			// Copy file contents
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to extract file: %w", err)
			}
			outFile.Close()

		default:
			log.Debugf("Skipping unsupported file type: %s (%c)", header.Name, header.Typeflag)
		}
	}

	log.Debugf("Successfully extracted archive to %s", dstDir)
	return nil
}

// ExtractTarXz extracts a tar.xz archive to a destination directory
func ExtractTarXz(src, dstDir string) error {
	return ExtractTarXzWithProgress(src, dstDir, nil)
}

// ExtractTarXzWithProgress extracts a tar.xz archive with progress tracking
func ExtractTarXzWithProgress(src, dstDir string, progressCallback func(float64)) error {
	log.Debugf("Extracting %s to %s", src, dstDir)

	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer srcFile.Close()

	// Get source file size for progress tracking
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get source file info: %w", err)
	}
	compressedSize := srcInfo.Size()

	// Wrap source file with progress reader to track compressed bytes read
	var reader io.Reader = srcFile
	if progressCallback != nil {
		reader = &progressReader{
			reader:   srcFile,
			total:    compressedSize,
			read:     0,
			callback: progressCallback,
			lastPct:  -1.0,
		}
	}

	// Create xz reader
	xzReader, err := xz.NewReader(reader)
	if err != nil {
		return fmt.Errorf("failed to create xz reader: %w", err)
	}

	// Create tar reader
	tarReader := tar.NewReader(xzReader)

	// Extract files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Construct destination path
		target := filepath.Join(dstDir, header.Name)

		// Security check: prevent path traversal
		cleanTarget := filepath.Clean(target)
		cleanDstDir := filepath.Clean(dstDir) + string(filepath.Separator)
		if !strings.HasPrefix(cleanTarget+string(filepath.Separator), cleanDstDir) && cleanTarget != filepath.Clean(dstDir) {
			return fmt.Errorf("invalid path in archive: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}

		case tar.TypeReg:
			// Create parent directory if needed
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			// Create file
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}

			// Copy file contents
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to extract file: %w", err)
			}
			outFile.Close()

		default:
			log.Debugf("Skipping unsupported file type: %s (%c)", header.Name, header.Typeflag)
		}
	}

	// Ensure 100% is reported
	if progressCallback != nil {
		progressCallback(1.0)
	}

	log.Debugf("Successfully extracted archive to %s", dstDir)
	return nil
}
