package compress

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gzBytes returns content gzip-compressed into a buffer.
func gzBytes(t *testing.T, content []byte) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	_, err := gzWriter.Write(content)
	require.NoError(t, err)
	require.NoError(t, gzWriter.Close())
	return &buf
}

// tarEntry appends a single regular-file entry to a tar writer.
func tarEntry(t *testing.T, tw *tar.Writer, name string, content []byte) {
	t.Helper()
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: name,
		Mode: 0o644,
		Size: int64(len(content)),
	}))
	_, err := tw.Write(content)
	require.NoError(t, err)
}

// TestGuardRailSentinels is the error contract: every guard-rail rejection must
// match a documented sentinel via errors.Is so callers can match errors
// programmatically instead of comparing strings.
func TestGuardRailSentinels(t *testing.T) {
	tests := []struct {
		name     string
		run      func(t *testing.T) error
		sentinel error
	}{
		{
			name: "UnGz rejects relative destination path",
			run: func(t *testing.T) error {
				_, err := UnGz(gzBytes(t, []byte("data")), "relative/out.txt")
				return err
			},
			sentinel: ErrPathTraversal,
		},
		{
			name: "UnGz enforces per-file size limit",
			run: func(t *testing.T) error {
				dst := filepath.Join(t.TempDir(), "out.txt")
				_, err := UnGz(gzBytes(t, bytes.Repeat([]byte("x"), 100)), dst, WithMaxFileSize(10))
				return err
			},
			sentinel: ErrSizeLimitExceeded,
		},
		{
			name: "UnTar rejects entry with .. traversal",
			run: func(t *testing.T) error {
				var buf bytes.Buffer
				tw := tar.NewWriter(&buf)
				tarEntry(t, tw, "../escape.txt", []byte("bad"))
				require.NoError(t, tw.Close())
				_, err := UnTar(&buf, t.TempDir())
				return err
			},
			sentinel: ErrPathTraversal,
		},
		{
			name: "UnTar rejects entry with absolute path",
			run: func(t *testing.T) error {
				var buf bytes.Buffer
				tw := tar.NewWriter(&buf)
				tarEntry(t, tw, "/etc/passwd", []byte("bad"))
				require.NoError(t, tw.Close())
				_, err := UnTar(&buf, t.TempDir())
				return err
			},
			sentinel: ErrPathTraversal,
		},
		{
			name: "UnTar rejects entry with backslash",
			run: func(t *testing.T) error {
				var buf bytes.Buffer
				tw := tar.NewWriter(&buf)
				tarEntry(t, tw, `dir\file.txt`, []byte("bad"))
				require.NoError(t, tw.Close())
				_, err := UnTar(&buf, t.TempDir())
				return err
			},
			sentinel: ErrPathTraversal,
		},
		{
			name: "UnTar enforces per-file size limit",
			run: func(t *testing.T) error {
				var buf bytes.Buffer
				tw := tar.NewWriter(&buf)
				tarEntry(t, tw, "big.txt", bytes.Repeat([]byte("x"), 100))
				require.NoError(t, tw.Close())
				_, err := UnTar(&buf, t.TempDir(), WithMaxFileSize(10))
				return err
			},
			sentinel: ErrSizeLimitExceeded,
		},
		{
			name: "UnTar enforces total archive size limit",
			run: func(t *testing.T) error {
				var buf bytes.Buffer
				tw := tar.NewWriter(&buf)
				tarEntry(t, tw, "a.txt", bytes.Repeat([]byte("a"), 10))
				tarEntry(t, tw, "b.txt", bytes.Repeat([]byte("b"), 10))
				require.NoError(t, tw.Close())
				_, err := UnTar(&buf, t.TempDir(), WithMaxArchiveSize(15))
				return err
			},
			sentinel: ErrSizeLimitExceeded,
		},
		{
			name: "UnTar rejects destination that is not a directory",
			run: func(t *testing.T) error {
				destFile := filepath.Join(t.TempDir(), "file")
				require.NoError(t, os.WriteFile(destFile, []byte("x"), 0o600))
				var buf bytes.Buffer
				require.NoError(t, tar.NewWriter(&buf).Close())
				_, err := UnTar(&buf, destFile)
				return err
			},
			sentinel: ErrNotDirectory,
		},
		{
			name: "Tar rejects source that is not a directory",
			run: func(t *testing.T) error {
				srcFile := filepath.Join(t.TempDir(), "file")
				require.NoError(t, os.WriteFile(srcFile, []byte("x"), 0o600))
				var buf bytes.Buffer
				return Tar(srcFile, &buf)
			},
			sentinel: ErrNotDirectory,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run(t)
			require.Error(t, err)
			assert.True(t, errors.Is(err, tt.sentinel),
				"error %q should match sentinel %q via errors.Is", err, tt.sentinel)
		})
	}
}

// TestUnGzWithMaxArchiveSize verifies that WithMaxArchiveSize actually takes
// effect in UnGz. For a single-file stream the effective limit is the smaller
// of maxFileSize and maxArchiveSize.
func TestUnGzWithMaxArchiveSize(t *testing.T) {
	content := bytes.Repeat([]byte("x"), 100)

	t.Run("archive size below file size triggers limit", func(t *testing.T) {
		dst := filepath.Join(t.TempDir(), "out.txt")
		_, err := UnGz(gzBytes(t, content), dst, WithMaxArchiveSize(10))
		assert.ErrorIs(t, err, ErrSizeLimitExceeded)
	})

	t.Run("archive size above file size passes", func(t *testing.T) {
		dst := filepath.Join(t.TempDir(), "out.txt")
		written, err := UnGz(gzBytes(t, content), dst, WithMaxArchiveSize(1000))
		assert.NoError(t, err)
		assert.Equal(t, int64(len(content)), written)
	})
}
