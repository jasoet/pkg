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

func Tar(sourceDirectory string, writer io.Writer) (err error) {
	if _, err = os.Stat(sourceDirectory); err != nil {
		return err
	}

	tarWriter := tar.NewWriter(writer)
	defer tarWriter.Close()

	err = filepath.Walk(sourceDirectory, func(file string, fileInfo os.FileInfo, err error) error {
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

		localDirectory := strings.Replace(file, sourceDirectory, "", -1)
		header.Name = strings.TrimPrefix(localDirectory, string(filepath.Separator))

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		openFile, err := os.Open(file)
		if err != nil {
			return err
		}
		if _, err := io.Copy(tarWriter, openFile); err != nil {
			return err
		}
		err = openFile.Close()
		return err
	})

	return err
}

func TarGz(sourceDirectory string, writer io.Writer) (err error) {
	if _, err = os.Stat(sourceDirectory); err != nil {
		return err
	}

	gzWriter := gzip.NewWriter(writer)
	defer gzWriter.Close()
	return Tar(sourceDirectory, gzWriter)
}

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

func UnTarGzBase64(encoded string, destinationDir string) (totalWritten int64, err error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return totalWritten, err
	}

	return UnTarGz(bytes.NewReader(decoded), destinationDir)
}

func UnTarGz(src io.Reader, destinationDir string) (totalWritten int64, err error) {
	zipReader, errReader := gzip.NewReader(src)
	if errReader != nil {
		err = errReader
		return totalWritten, err
	}
	defer zipReader.Close()

	return UnTar(zipReader, destinationDir)
}

// validTarPath validates that a tar entry path is safe to extract
func validTarPath(path string) bool {
	if path == "" ||
		strings.Contains(path, `\`) ||
		strings.HasPrefix(path, "/") ||
		strings.Contains(path, "../") {
		return false
	}
	return true
}

// extractTarDirectory creates a directory from a tar entry
func extractTarDirectory(target string) error {
	if _, err := os.Stat(target); os.IsNotExist(err) {
		if err := os.MkdirAll(target, 0o750); err != nil {
			return err
		}
	}
	return nil
}

// extractTarFile extracts a regular file from a tar entry
func extractTarFile(tarReader *tar.Reader, target string, header *tar.Header) (int64, error) {
	// Ensure parent directory exists
	parentDir := filepath.Dir(target)
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		if err := os.MkdirAll(parentDir, 0o750); err != nil {
			return 0, err
		}
	}

	// Validate file mode to prevent integer overflow
	fileMode := header.Mode
	if fileMode > 0o777 {
		fileMode = 0o644 // Use safe default
	}
	// Explicit conversion to prevent integer overflow
	safeMode := os.FileMode(fileMode & 0o777)
	fileToWrite, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, safeMode)
	if err != nil {
		return 0, err
	}
	defer fileToWrite.Close()

	// Limit decompression to prevent zip bombs (100MB limit)
	limitedReader := io.LimitReader(tarReader, 100*1024*1024)
	written, err := io.Copy(fileToWrite, limitedReader)
	if err != nil {
		return 0, err
	}

	return written, nil
}

func UnTar(src io.Reader, destinationDir string) (written int64, err error) {
	info, err := os.Stat(destinationDir)
	if err != nil {
		return 0, err
	}

	if !info.IsDir() {
		return 0, fmt.Errorf("%s is not a directory", destinationDir)
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
			return totalWritten, fmt.Errorf("tar contained invalid path %s\n", header.Name)
		}

		// Prevent path traversal attacks
		target := filepath.Join(destinationDir, header.Name)
		if !strings.HasPrefix(target, filepath.Clean(destinationDir)+string(os.PathSeparator)) {
			return totalWritten, fmt.Errorf("invalid file path: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := extractTarDirectory(target); err != nil {
				return totalWritten, err
			}
		case tar.TypeReg:
			written, err := extractTarFile(tarReader, target, header)
			if err != nil {
				return totalWritten, err
			}
			totalWritten += written
		}
	}

	return totalWritten, nil
}
