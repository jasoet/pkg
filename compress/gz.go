// Package compress provides gzip and tar archive utilities with security protections.
//
// Features include path traversal prevention, zip bomb protection (100 MB per-file limit),
// and file mode validation. Supports tar, gzip, tar.gz, and base64-encoded tar.gz formats.
package compress

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Gz compresses data from source using gzip and writes the compressed output to writer.
//
// The caller is responsible for closing writer if needed.
func Gz(source io.Reader, writer io.Writer) error {
	gzWriter := gzip.NewWriter(writer)
	if _, err := io.Copy(gzWriter, source); err != nil {
		_ = gzWriter.Close()
		return err
	}
	return gzWriter.Close()
}

// UnGz decompresses gzip data from src and writes the result to the file at dst.
//
// Decompression is limited by maxFileSize (default 100 MB) to prevent zip bomb attacks.
// dst must be an absolute path to prevent path traversal.
// Returns the number of bytes written and any error encountered.
func UnGz(src io.Reader, dst string, opts ...ExtractOption) (int64, error) {
	// Validate destination path to prevent directory traversal
	if !filepath.IsAbs(dst) {
		return 0, fmt.Errorf("%w: destination must be an absolute path: %s", ErrPathTraversal, dst)
	}

	cfg := defaultExtractConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	zipReader, errReader := gzip.NewReader(src)
	if errReader != nil {
		return 0, errReader
	}
	defer func() { _ = zipReader.Close() }()

	destinationFile, errCreate := os.Create(dst)
	if errCreate != nil {
		return 0, errCreate
	}
	defer func() { _ = destinationFile.Close() }()

	// Limit decompression to prevent zip bombs
	limitedReader := io.LimitReader(zipReader, cfg.maxFileSize)
	written, err := io.Copy(destinationFile, limitedReader)
	if err != nil {
		return written, err
	}

	if written >= cfg.maxFileSize {
		probe := make([]byte, 1)
		if n, _ := zipReader.Read(probe); n > 0 {
			return written, fmt.Errorf("%w: file exceeds maximum size of %d bytes", ErrSizeLimitExceeded, cfg.maxFileSize)
		}
	}

	return written, nil
}
