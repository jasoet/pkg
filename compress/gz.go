package compress

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func Gz(source io.Reader, writer io.Writer) (err error) {
	gzWriter := gzip.NewWriter(writer)
	defer gzWriter.Close()
	_, err = io.Copy(gzWriter, source)
	return err
}

func UnGz(src io.Reader, dst string) (written int64, err error) {
	// Validate destination path to prevent directory traversal
	cleanDst := filepath.Clean(dst)
	if strings.Contains(cleanDst, "..") {
		return 0, fmt.Errorf("invalid destination path: %s", dst)
	}

	zipReader, errReader := gzip.NewReader(src)
	if errReader != nil {
		err = errReader
		return written, err
	}
	defer zipReader.Close()

	destinationFile, errCreate := os.Create(cleanDst)
	if errCreate != nil {
		err = errCreate
		return written, err
	}
	defer destinationFile.Close()

	// Limit decompression to prevent zip bombs (100MB limit)
	limitedReader := io.LimitReader(zipReader, 100*1024*1024)
	return io.Copy(destinationFile, limitedReader)
}
