package compress

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Tar creates a tar archive of sourceDirectory and writes it to writer.
//
// Only regular files are included; symlinks and other special files are skipped.
// The caller is responsible for closing writer if needed.
func Tar(sourceDirectory string, writer io.Writer) error {
	fileInfo, err := os.Stat(sourceDirectory)
	if err != nil {
		return err
	}
	if !fileInfo.IsDir() {
		return fmt.Errorf("%w: %s", ErrNotDirectory, sourceDirectory)
	}

	tarWriter := tar.NewWriter(writer)
	defer func() { _ = tarWriter.Close() }()

	return filepath.Walk(sourceDirectory, func(file string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !fileInfo.Mode().IsRegular() {
			return nil
		}

		header, err := tar.FileInfoHeader(fileInfo, fileInfo.Name())
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourceDirectory, file)
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		openFile, err := os.Open(file)
		if err != nil {
			return err
		}
		defer func() { _ = openFile.Close() }()

		_, err = io.Copy(tarWriter, openFile)
		return err
	})
}

// TarGz creates a gzip-compressed tar archive of sourceDirectory and writes it to writer.
//
// The caller is responsible for closing writer if needed.
func TarGz(sourceDirectory string, writer io.Writer) error {
	if _, err := os.Stat(sourceDirectory); err != nil {
		return err
	}

	gzWriter := gzip.NewWriter(writer)
	if err := Tar(sourceDirectory, gzWriter); err != nil {
		_ = gzWriter.Close()
		return err
	}
	return gzWriter.Close()
}

// TarGzBase64 creates a gzip-compressed tar archive of sourceDirectory and returns it
// as a base64-encoded string.
func TarGzBase64(sourceDirectory string) (string, error) {
	var buf bytes.Buffer
	encoder := base64.NewEncoder(base64.StdEncoding, &buf)

	err := TarGz(sourceDirectory, encoder)
	if err != nil {
		return "", err
	}

	err = encoder.Close()
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// UnTarGzBase64 decodes a base64-encoded gzip tar archive and extracts it to destinationDir.
func UnTarGzBase64(encoded string, destinationDir string, opts ...ExtractOption) (int64, error) {
	decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(encoded))
	return UnTarGz(decoder, destinationDir, opts...)
}

// UnTarGz decompresses a gzip stream and extracts the tar archive to destinationDir.
func UnTarGz(src io.Reader, destinationDir string, opts ...ExtractOption) (int64, error) {
	zipReader, err := gzip.NewReader(src)
	if err != nil {
		return 0, err
	}
	defer func() { _ = zipReader.Close() }()

	return UnTar(zipReader, destinationDir, opts...)
}

// validTarPath validates that a tar entry path is safe to extract.
func validTarPath(path string) bool {
	if path == "" ||
		strings.Contains(path, `\`) || // Backslash check prevents Windows-style path separator usage on Unix
		strings.HasPrefix(path, "/") {
		return false
	}
	// Reject only path elements that are exactly ".." (parent-directory
	// traversal). Names that merely contain ".." — e.g. "report..final.txt" —
	// are legitimate and must be allowed.
	for _, elem := range strings.Split(path, "/") {
		if elem == ".." {
			return false
		}
	}
	return true
}

// extractTarDirectory creates a directory from a tar entry
func extractTarDirectory(target string) error {
	return os.MkdirAll(target, 0o750)
}

const (
	// DefaultMaxFileSize is the default maximum size for a single extracted file (100 MB).
	DefaultMaxFileSize int64 = 100 * 1024 * 1024
	// DefaultMaxArchiveSize is the default maximum total extracted size for an archive (1 GB).
	DefaultMaxArchiveSize int64 = 1024 * 1024 * 1024
)

// extractTarFile extracts a regular file from a tar entry.
//
// remainingArchive is how much of the whole-archive size budget is still
// available; the file is capped at the smaller of maxFileSize and
// remainingArchive so the archive total is enforced mid-file rather than only
// after a file is fully written. On any error the partially written target is
// removed so no partial output is left on disk.
func extractTarFile(tarReader *tar.Reader, target string, header *tar.Header, maxFileSize, remainingArchive int64) (int64, error) {
	// Ensure parent directory exists
	parentDir := filepath.Dir(target)
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		if err := os.MkdirAll(parentDir, 0o750); err != nil {
			return 0, err
		}
	}

	// Refuse to write through a pre-existing symlink at the leaf target: a
	// symlink planted in the destination must not redirect the write outside it.
	if info, errLstat := os.Lstat(target); errLstat == nil && info.Mode()&os.ModeSymlink != 0 {
		return 0, fmt.Errorf("%w: refusing to write through symlink %s", ErrPathTraversal, header.Name)
	}

	safeMode := os.FileMode(header.Mode & 0o777) // Strip setuid/setgid/sticky bits
	// O_NOFOLLOW (where supported) closes the TOCTOU window if a symlink appears
	// at target between the Lstat check and the open; O_TRUNC clears any stale
	// trailing bytes when overwriting an existing longer file with shorter
	// content.
	fileToWrite, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC|oNoFollow, safeMode)
	if err != nil {
		return 0, err
	}

	// Effective cap: never exceed the per-file limit, and never exceed what
	// remains of the whole-archive budget.
	effectiveLimit := maxFileSize
	archiveBound := false
	if remainingArchive < effectiveLimit {
		effectiveLimit = remainingArchive
		archiveBound = true
	}

	// Limit decompression to prevent zip bombs.
	limitedReader := io.LimitReader(tarReader, effectiveLimit)
	written, err := io.Copy(fileToWrite, limitedReader)
	if err != nil {
		_ = fileToWrite.Close()
		_ = os.Remove(target)
		return written, err
	}

	if written >= effectiveLimit {
		probe := make([]byte, 1)
		if n, _ := tarReader.Read(probe); n > 0 {
			_ = fileToWrite.Close()
			_ = os.Remove(target)
			if archiveBound {
				return written, fmt.Errorf("file %s: %w (archive total exceeds %d bytes)", header.Name, ErrSizeLimitExceeded, remainingArchive)
			}
			return written, fmt.Errorf("file %s: %w (max %d bytes)", header.Name, ErrSizeLimitExceeded, maxFileSize)
		}
	}

	if err := fileToWrite.Close(); err != nil {
		_ = os.Remove(target)
		return written, err
	}

	return written, nil
}

// ExtractOption configures extraction behavior for UnGz, UnTar, UnTarGz,
// and UnTarGzBase64.
type ExtractOption func(*extractConfig)

type extractConfig struct {
	maxFileSize    int64
	maxArchiveSize int64
}

func defaultExtractConfig() extractConfig {
	return extractConfig{
		maxFileSize:    DefaultMaxFileSize,
		maxArchiveSize: DefaultMaxArchiveSize,
	}
}

// WithMaxFileSize sets the maximum allowed size for a single extracted file.
// Default: 100 MB.
func WithMaxFileSize(size int64) ExtractOption {
	return func(c *extractConfig) { c.maxFileSize = size }
}

// WithMaxArchiveSize sets the maximum total extracted size for the entire archive.
// Default: 1 GB.
func WithMaxArchiveSize(size int64) ExtractOption {
	return func(c *extractConfig) { c.maxArchiveSize = size }
}

// UnTar extracts a tar archive from src into destinationDir.
//
// Includes security protections: path traversal prevention, file mode validation,
// per-file size limit (default 100 MB), and total archive size limit (default 1 GB)
// to prevent zip bombs. Use ExtractOption to customize limits.
// Unlike UnGz, destinationDir may be a relative path; entry paths inside the
// archive are still validated to stay within destinationDir.
func UnTar(src io.Reader, destinationDir string, opts ...ExtractOption) (written int64, err error) {
	cfg := defaultExtractConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	info, err := os.Stat(destinationDir)
	if err != nil {
		return 0, err
	}

	if !info.IsDir() {
		return 0, fmt.Errorf("%w: %s", ErrNotDirectory, destinationDir)
	}

	tarReader := tar.NewReader(src)

	var totalWritten int64
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return totalWritten, err
		}

		if !validTarPath(header.Name) {
			return totalWritten, fmt.Errorf("%w: tar contained invalid path %s", ErrPathTraversal, header.Name)
		}

		// Prevent path traversal attacks. Use a filepath.Rel-based containment
		// check so a relative destination such as "." is accepted while entries
		// that escape the destination are still rejected.
		target := filepath.Join(destinationDir, header.Name)
		rel, errRel := filepath.Rel(destinationDir, target)
		if errRel != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return totalWritten, fmt.Errorf("%w: invalid file path: %s", ErrPathTraversal, header.Name)
		}

		// Prevent symlink TOCTOU attacks: resolve symlinks in parent directory
		parentDir := filepath.Dir(target)
		if resolvedParent, errSym := filepath.EvalSymlinks(parentDir); errSym == nil {
			resolvedDest := filepath.Clean(destinationDir)
			if rd, errRD := filepath.EvalSymlinks(destinationDir); errRD == nil {
				resolvedDest = rd
			}
			if !strings.HasPrefix(resolvedParent, resolvedDest+string(os.PathSeparator)) && resolvedParent != resolvedDest {
				return totalWritten, fmt.Errorf("%w: symlink resolves outside destination: %s", ErrPathTraversal, header.Name)
			}
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := extractTarDirectory(target); err != nil {
				return totalWritten, err
			}
		case tar.TypeReg:
			// Pass the remaining archive budget so the per-file cap also honors
			// the whole-archive limit mid-file instead of only after the fact.
			remainingArchive := cfg.maxArchiveSize - totalWritten
			written, err := extractTarFile(tarReader, target, header, cfg.maxFileSize, remainingArchive)
			totalWritten += written
			if err != nil {
				return totalWritten, err
			}
		}
	}

	return totalWritten, nil
}
