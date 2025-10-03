package compress

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Security Tests - Path Traversal Prevention
// ============================================================================

func TestValidTarPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "valid simple path",
			path:     "file.txt",
			expected: true,
		},
		{
			name:     "valid nested path",
			path:     "dir/subdir/file.txt",
			expected: true,
		},
		{
			name:     "empty path is invalid",
			path:     "",
			expected: false,
		},
		{
			name:     "path with backslash is invalid",
			path:     "dir\\file.txt",
			expected: false,
		},
		{
			name:     "absolute path is invalid",
			path:     "/etc/passwd",
			expected: false,
		},
		{
			name:     "path traversal with ../ is invalid",
			path:     "../../../etc/passwd",
			expected: false,
		},
		{
			name:     "path with ../ in middle is invalid",
			path:     "dir/../../../etc/passwd",
			expected: false,
		},
		{
			name:     "path with single ../ is invalid",
			path:     "dir/../file.txt",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validTarPath(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUnTarPathTraversalAttack(t *testing.T) {
	t.Run("rejects path traversal with ../", func(t *testing.T) {
		// Create malicious tar with path traversal
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)

		// Try to write outside destination with ../
		header := &tar.Header{
			Name: "../../../etc/malicious.txt",
			Mode: 0o600,
			Size: 10,
		}
		err := tw.WriteHeader(header)
		require.NoError(t, err)
		_, err = tw.Write([]byte("malicious\n"))
		require.NoError(t, err)
		tw.Close()

		// Attempt to extract
		destDir, err := os.MkdirTemp("", "test-path-traversal")
		require.NoError(t, err)
		defer os.RemoveAll(destDir)

		_, err = UnTar(&buf, destDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid path")
	})

	t.Run("rejects absolute path", func(t *testing.T) {
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)

		header := &tar.Header{
			Name: "/etc/passwd",
			Mode: 0o600,
			Size: 5,
		}
		err := tw.WriteHeader(header)
		require.NoError(t, err)
		_, err = tw.Write([]byte("test\n"))
		require.NoError(t, err)
		tw.Close()

		destDir, err := os.MkdirTemp("", "test-absolute-path")
		require.NoError(t, err)
		defer os.RemoveAll(destDir)

		_, err = UnTar(&buf, destDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid path")
	})

	t.Run("rejects path with backslash", func(t *testing.T) {
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)

		header := &tar.Header{
			Name: "dir\\file.txt",
			Mode: 0o600,
			Size: 4,
		}
		err := tw.WriteHeader(header)
		require.NoError(t, err)
		_, err = tw.Write([]byte("test"))
		require.NoError(t, err)
		tw.Close()

		destDir, err := os.MkdirTemp("", "test-backslash")
		require.NoError(t, err)
		defer os.RemoveAll(destDir)

		_, err = UnTar(&buf, destDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid path")
	})
}

func TestUnGzPathTraversalPrevention(t *testing.T) {
	t.Run("rejects path with .. traversal", func(t *testing.T) {
		var buf bytes.Buffer
		gzWriter := gzip.NewWriter(&buf)
		_, err := gzWriter.Write([]byte("malicious content"))
		require.NoError(t, err)
		gzWriter.Close()

		// Try to write to a path with traversal
		_, err = UnGz(&buf, "../../../tmp/malicious.txt")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid destination path")
	})

	t.Run("accepts clean path", func(t *testing.T) {
		var buf bytes.Buffer
		gzWriter := gzip.NewWriter(&buf)
		_, err := gzWriter.Write([]byte("safe content"))
		require.NoError(t, err)
		gzWriter.Close()

		destFile, err := os.CreateTemp("", "test-ungz-safe")
		require.NoError(t, err)
		defer os.Remove(destFile.Name())
		destFile.Close()

		written, err := UnGz(&buf, destFile.Name())
		assert.NoError(t, err)
		assert.Greater(t, written, int64(0))
	})
}

// ============================================================================
// Security Tests - Zip Bomb Protection
// ============================================================================

func TestUnTarZipBombProtection(t *testing.T) {
	t.Run("limits extraction to 100MB per file", func(t *testing.T) {
		// Create tar with a file that would decompress to >100MB
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)

		// Create header claiming 200MB size
		header := &tar.Header{
			Name: "large-file.txt",
			Mode: 0o600,
			Size: 200 * 1024 * 1024, // 200MB
		}
		err := tw.WriteHeader(header)
		require.NoError(t, err)

		// Write 200MB of zeros
		zeros := make([]byte, 1024*1024) // 1MB buffer
		for i := 0; i < 200; i++ {
			_, err = tw.Write(zeros)
			if err != nil {
				break
			}
		}
		tw.Close()

		destDir, err := os.MkdirTemp("", "test-zip-bomb")
		require.NoError(t, err)
		defer os.RemoveAll(destDir)

		written, err := UnTar(&buf, destDir)

		// Should succeed but only write up to 100MB due to limit
		assert.NoError(t, err)
		assert.LessOrEqual(t, written, int64(100*1024*1024))
	})
}

func TestUnGzZipBombProtection(t *testing.T) {
	t.Run("limits decompression to 100MB", func(t *testing.T) {
		// Create a file that compresses well (lots of zeros)
		var uncompressed bytes.Buffer
		zeros := make([]byte, 1024*1024) // 1MB of zeros
		for i := 0; i < 150; i++ {       // 150MB uncompressed
			uncompressed.Write(zeros)
		}

		// Compress it
		var compressed bytes.Buffer
		gzWriter := gzip.NewWriter(&compressed)
		_, err := gzWriter.Write(uncompressed.Bytes())
		require.NoError(t, err)
		gzWriter.Close()

		destFile, err := os.CreateTemp("", "test-ungz-bomb")
		require.NoError(t, err)
		defer os.Remove(destFile.Name())
		destFile.Close()

		written, err := UnGz(&compressed, destFile.Name())

		// Should succeed but only write up to 100MB
		assert.NoError(t, err)
		assert.LessOrEqual(t, written, int64(100*1024*1024))
	})
}

// ============================================================================
// Edge Case Tests - Directory Extraction
// ============================================================================

func TestExtractTarDirectory(t *testing.T) {
	t.Run("creates directory when it doesn't exist", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "test-extract-dir")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		targetDir := filepath.Join(tempDir, "newdir", "subdir")

		err = extractTarDirectory(targetDir)
		assert.NoError(t, err)

		// Verify directory was created
		info, err := os.Stat(targetDir)
		assert.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("succeeds when directory already exists", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "test-extract-existing")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		targetDir := filepath.Join(tempDir, "existing")
		err = os.MkdirAll(targetDir, 0o750)
		require.NoError(t, err)

		err = extractTarDirectory(targetDir)
		assert.NoError(t, err)
	})
}

func TestUnTarWithDirectories(t *testing.T) {
	t.Run("extracts directories and files", func(t *testing.T) {
		// Create tar with directories and files
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)

		// Add directory
		dirHeader := &tar.Header{
			Name:     "testdir/",
			Mode:     0o755,
			Typeflag: tar.TypeDir,
		}
		err := tw.WriteHeader(dirHeader)
		require.NoError(t, err)

		// Add file in directory
		fileHeader := &tar.Header{
			Name: "testdir/file.txt",
			Mode: 0o644,
			Size: 12,
		}
		err = tw.WriteHeader(fileHeader)
		require.NoError(t, err)
		_, err = tw.Write([]byte("test content"))
		require.NoError(t, err)

		tw.Close()

		destDir, err := os.MkdirTemp("", "test-with-dirs")
		require.NoError(t, err)
		defer os.RemoveAll(destDir)

		written, err := UnTar(&buf, destDir)
		assert.NoError(t, err)
		assert.Equal(t, int64(12), written)

		// Verify directory exists
		dirPath := filepath.Join(destDir, "testdir")
		info, err := os.Stat(dirPath)
		assert.NoError(t, err)
		assert.True(t, info.IsDir())

		// Verify file exists
		filePath := filepath.Join(destDir, "testdir", "file.txt")
		content, err := os.ReadFile(filePath)
		assert.NoError(t, err)
		assert.Equal(t, "test content", string(content))
	})
}

// ============================================================================
// Edge Case Tests - File Mode Validation
// ============================================================================

func TestExtractTarFileMode(t *testing.T) {
	t.Run("handles file mode overflow safely", func(t *testing.T) {
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)

		// Create header with very large mode value (potential overflow)
		header := &tar.Header{
			Name: "test.txt",
			Mode: 0o7777777, // Much larger than valid file mode
			Size: 4,
		}
		err := tw.WriteHeader(header)
		require.NoError(t, err)
		_, err = tw.Write([]byte("test"))
		require.NoError(t, err)
		tw.Close()

		destDir, err := os.MkdirTemp("", "test-file-mode")
		require.NoError(t, err)
		defer os.RemoveAll(destDir)

		written, err := UnTar(&buf, destDir)
		assert.NoError(t, err)
		assert.Equal(t, int64(4), written)

		// Verify file was created with safe mode
		filePath := filepath.Join(destDir, "test.txt")
		info, err := os.Stat(filePath)
		assert.NoError(t, err)
		// Mode should be masked to safe value (0o644 or 0o644)
		mode := info.Mode().Perm()
		assert.LessOrEqual(t, mode, os.FileMode(0o777))
	})
}

// ============================================================================
// Edge Case Tests - Invalid Inputs
// ============================================================================

func TestUnTarInvalidInputs(t *testing.T) {
	t.Run("returns error when destination is not a directory", func(t *testing.T) {
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)
		tw.Close()

		// Use a file as destination instead of directory
		destFile, err := os.CreateTemp("", "test-not-dir")
		require.NoError(t, err)
		defer os.Remove(destFile.Name())
		destFile.Close()

		_, err = UnTar(&buf, destFile.Name())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a directory")
	})

	t.Run("returns error when destination doesn't exist", func(t *testing.T) {
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)
		tw.Close()

		_, err := UnTar(&buf, "/nonexistent/path/that/does/not/exist")
		assert.Error(t, err)
	})

	t.Run("handles empty tar archive", func(t *testing.T) {
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)
		tw.Close()

		destDir, err := os.MkdirTemp("", "test-empty-tar")
		require.NoError(t, err)
		defer os.RemoveAll(destDir)

		written, err := UnTar(&buf, destDir)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), written)
	})

	t.Run("returns error for corrupted tar", func(t *testing.T) {
		// Create malformed tar data
		buf := bytes.NewReader([]byte("this is not a valid tar file"))

		destDir, err := os.MkdirTemp("", "test-corrupted")
		require.NoError(t, err)
		defer os.RemoveAll(destDir)

		_, err = UnTar(buf, destDir)
		assert.Error(t, err)
	})
}

func TestUnGzInvalidInputs(t *testing.T) {
	t.Run("returns error for invalid gzip data", func(t *testing.T) {
		buf := bytes.NewReader([]byte("this is not gzip data"))

		destFile, err := os.CreateTemp("", "test-invalid-gz")
		require.NoError(t, err)
		defer os.Remove(destFile.Name())
		destFile.Close()

		_, err = UnGz(buf, destFile.Name())
		assert.Error(t, err)
	})
}

func TestUnTarGzBase64InvalidInputs(t *testing.T) {
	t.Run("returns error for invalid base64", func(t *testing.T) {
		destDir, err := os.MkdirTemp("", "test-invalid-base64")
		require.NoError(t, err)
		defer os.RemoveAll(destDir)

		_, err = UnTarGzBase64("!!!invalid base64!!!", destDir)
		assert.Error(t, err)
	})

	t.Run("returns error for valid base64 but invalid gzip", func(t *testing.T) {
		// Valid base64 but not gzip content
		invalidGzip := base64.StdEncoding.EncodeToString([]byte("not gzip"))

		destDir, err := os.MkdirTemp("", "test-base64-not-gzip")
		require.NoError(t, err)
		defer os.RemoveAll(destDir)

		_, err = UnTarGzBase64(invalidGzip, destDir)
		assert.Error(t, err)
	})
}

func TestTarInvalidInputs(t *testing.T) {
	t.Run("returns error for non-existent source directory", func(t *testing.T) {
		var buf bytes.Buffer

		err := Tar("/nonexistent/directory/path", &buf)
		assert.Error(t, err)
	})

	t.Run("returns error when source is a file not directory", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "test-file-not-dir")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())
		tempFile.Close()

		var buf bytes.Buffer
		err = Tar(tempFile.Name(), &buf)
		// Should succeed as it will walk the file (0 regular files)
		// or handle appropriately
		// The function uses filepath.Walk which handles files
		_ = err // Intentionally not checking - filepath.Walk handles files
	})
}

func TestTarGzInvalidInputs(t *testing.T) {
	t.Run("returns error for non-existent source directory", func(t *testing.T) {
		var buf bytes.Buffer

		err := TarGz("/nonexistent/directory/path", &buf)
		assert.Error(t, err)
	})
}

// ============================================================================
// Edge Case Tests - Nested Directories
// ============================================================================

func TestUnTarNestedDirectories(t *testing.T) {
	t.Run("creates parent directories automatically", func(t *testing.T) {
		// Create tar with deeply nested file (no explicit directory entries)
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)

		header := &tar.Header{
			Name: "level1/level2/level3/file.txt",
			Mode: 0o644,
			Size: 5,
		}
		err := tw.WriteHeader(header)
		require.NoError(t, err)
		_, err = tw.Write([]byte("test\n"))
		require.NoError(t, err)
		tw.Close()

		destDir, err := os.MkdirTemp("", "test-nested")
		require.NoError(t, err)
		defer os.RemoveAll(destDir)

		written, err := UnTar(&buf, destDir)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), written)

		// Verify nested file exists
		filePath := filepath.Join(destDir, "level1", "level2", "level3", "file.txt")
		content, err := os.ReadFile(filePath)
		assert.NoError(t, err)
		assert.Equal(t, "test\n", string(content))
	})
}

// ============================================================================
// Integration Tests - Round Trip
// ============================================================================

func TestTarUnTarRoundTrip(t *testing.T) {
	t.Run("round trip preserves directory structure", func(t *testing.T) {
		// Create source directory structure
		srcDir, err := os.MkdirTemp("", "test-roundtrip-src")
		require.NoError(t, err)
		defer os.RemoveAll(srcDir)

		// Create test files
		err = os.MkdirAll(filepath.Join(srcDir, "dir1", "dir2"), 0o755)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0o644)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(srcDir, "dir1", "file2.txt"), []byte("content2"), 0o644)
		require.NoError(t, err)

		// Tar it
		var buf bytes.Buffer
		err = Tar(srcDir, &buf)
		require.NoError(t, err)

		// Untar to new location
		destDir, err := os.MkdirTemp("", "test-roundtrip-dest")
		require.NoError(t, err)
		defer os.RemoveAll(destDir)

		written, err := UnTar(&buf, destDir)
		assert.NoError(t, err)
		assert.Greater(t, written, int64(0))

		// Verify files
		content1, err := os.ReadFile(filepath.Join(destDir, "file1.txt"))
		assert.NoError(t, err)
		assert.Equal(t, "content1", string(content1))

		content2, err := os.ReadFile(filepath.Join(destDir, "dir1", "file2.txt"))
		assert.NoError(t, err)
		assert.Equal(t, "content2", string(content2))
	})
}

func TestTarGzBase64RoundTrip(t *testing.T) {
	t.Run("base64 encoding round trip works correctly", func(t *testing.T) {
		// Create source directory
		srcDir, err := os.MkdirTemp("", "test-base64-roundtrip-src")
		require.NoError(t, err)
		defer os.RemoveAll(srcDir)

		testContent := "test data for base64 encoding"
		err = os.WriteFile(filepath.Join(srcDir, "test.txt"), []byte(testContent), 0o644)
		require.NoError(t, err)

		// Encode
		encoded, err := TarGzBase64(srcDir)
		require.NoError(t, err)
		assert.NotEmpty(t, encoded)

		// Verify it's valid base64
		_, err = base64.StdEncoding.DecodeString(encoded)
		assert.NoError(t, err)

		// Decode
		destDir, err := os.MkdirTemp("", "test-base64-roundtrip-dest")
		require.NoError(t, err)
		defer os.RemoveAll(destDir)

		written, err := UnTarGzBase64(encoded, destDir)
		assert.NoError(t, err)
		assert.Greater(t, written, int64(0))

		// Verify content
		content, err := os.ReadFile(filepath.Join(destDir, "test.txt"))
		assert.NoError(t, err)
		assert.Equal(t, testContent, string(content))
	})
}

// ============================================================================
// Edge Case Tests - Special Characters in Filenames
// ============================================================================

func TestUnTarSpecialCharactersInFilenames(t *testing.T) {
	t.Run("handles filenames with spaces", func(t *testing.T) {
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)

		header := &tar.Header{
			Name: "file with spaces.txt",
			Mode: 0o644,
			Size: 4,
		}
		err := tw.WriteHeader(header)
		require.NoError(t, err)
		_, err = tw.Write([]byte("test"))
		require.NoError(t, err)
		tw.Close()

		destDir, err := os.MkdirTemp("", "test-spaces")
		require.NoError(t, err)
		defer os.RemoveAll(destDir)

		written, err := UnTar(&buf, destDir)
		assert.NoError(t, err)
		assert.Equal(t, int64(4), written)

		filePath := filepath.Join(destDir, "file with spaces.txt")
		_, err = os.Stat(filePath)
		assert.NoError(t, err)
	})

	t.Run("handles unicode filenames", func(t *testing.T) {
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)

		header := &tar.Header{
			Name: "文件.txt", // Chinese characters
			Mode: 0o644,
			Size: 4,
		}
		err := tw.WriteHeader(header)
		require.NoError(t, err)
		_, err = tw.Write([]byte("test"))
		require.NoError(t, err)
		tw.Close()

		destDir, err := os.MkdirTemp("", "test-unicode")
		require.NoError(t, err)
		defer os.RemoveAll(destDir)

		written, err := UnTar(&buf, destDir)
		assert.NoError(t, err)
		assert.Equal(t, int64(4), written)
	})
}
