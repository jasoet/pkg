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
		gzWriter.Close()
		return err
	}
	return gzWriter.Close()
}

// UnGz decompresses gzip data from src and writes the result to the file at dst.
//
// Decompression is limited to 100 MB to prevent zip bomb attacks.
// dst must be an absolute path to prevent path traversal.
// Returns the number of bytes written and any error encountered.
func UnGz(src io.Reader, dst string) (written int64, err error) {
	// Validate destination path to prevent directory traversal
	if !filepath.IsAbs(dst) {
		return 0, fmt.Errorf("%w: destination must be an absolute path: %s", ErrPathTraversal, dst)
	}

	const maxSize = 100 * 1024 * 1024 // 100 MB

	zipReader, errReader := gzip.NewReader(src)
	if errReader != nil {
		err = errReader
		return written, err
	}
	defer zipReader.Close()

	destinationFile, errCreate := os.Create(dst)
	if errCreate != nil {
		err = errCreate
		return written, err
	}
	defer destinationFile.Close()

	// Limit decompression to prevent zip bombs (100MB limit)
	limitedReader := io.LimitReader(zipReader, maxSize)
	written, err = io.Copy(destinationFile, limitedReader)
	if err != nil {
		return written, err
	}

	if written >= maxSize {
		probe := make([]byte, 1)
		if n, _ := zipReader.Read(probe); n > 0 {
			return written, fmt.Errorf("%w: file exceeds maximum size of %d bytes", ErrSizeLimitExceeded, maxSize)
		}
	}

	return written, nil
}
