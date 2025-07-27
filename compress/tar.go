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
		return
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

	return
}

func TarGz(sourceDirectory string, writer io.Writer) (err error) {
	if _, err = os.Stat(sourceDirectory); err != nil {
		return
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
		return
	}

	return UnTarGz(bytes.NewReader(decoded), destinationDir)
}

func UnTarGz(src io.Reader, destinationDir string) (totalWritten int64, err error) {
	zipReader, errReader := gzip.NewReader(src)
	if errReader != nil {
		err = errReader
		return
	}
	defer zipReader.Close()

	return UnTar(zipReader, destinationDir)
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

	validPath := func(path string) bool {
		if path == "" ||
			strings.Contains(path, `\`) ||
			strings.HasPrefix(path, "/") ||
			strings.Contains(path, "../") {
			return false
		}
		return true
	}

	var totalWritten int64
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return totalWritten, err
		}

		if !validPath(header.Name) {
			return totalWritten, fmt.Errorf("tar contained invalid path %s\n", header.Name)
		}

		// Prevent path traversal attacks
		target := filepath.Join(destinationDir, header.Name)
		if !strings.HasPrefix(target, filepath.Clean(destinationDir)+string(os.PathSeparator)) {
			return totalWritten, fmt.Errorf("invalid file path: %s", header.Name)
		}

		switch header.Typeflag {

		case tar.TypeDir:
			if _, err := os.Stat(target); os.IsNotExist(err) {
				if err := os.MkdirAll(target, 0750); err != nil {
					return totalWritten, err
				}
			}
		case tar.TypeReg:
			// Ensure parent directory exists
			parentDir := filepath.Dir(target)
			if _, err := os.Stat(parentDir); os.IsNotExist(err) {
				if err := os.MkdirAll(parentDir, 0750); err != nil {
					return totalWritten, err
				}
			}

			// Validate file mode to prevent integer overflow
		fileMode := header.Mode
		if fileMode > 0777 {
			fileMode = 0644 // Use safe default
		}
		// Explicit conversion to prevent integer overflow
		safeMode := os.FileMode(fileMode & 0777)
		fileToWrite, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, safeMode)
			if err != nil {
				return totalWritten, err
			}
			// Limit decompression to prevent zip bombs (100MB limit)
		limitedReader := io.LimitReader(tarReader, 100*1024*1024)
		written, err := io.Copy(fileToWrite, limitedReader)
			if err != nil {
				return totalWritten, err
			}

			totalWritten = totalWritten + written
			_ = fileToWrite.Close()
		}
	}

	return totalWritten, nil
}
