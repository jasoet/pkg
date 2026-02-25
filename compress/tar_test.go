package compress

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnTar(t *testing.T) {
	archiveReader, err := os.Open("test_archive.tar")
	require.NoError(t, err)
	defer archiveReader.Close()

	destinationDir := t.TempDir()
	t.Logf("created temp dir: %s", destinationDir)

	written, err := UnTar(archiveReader, destinationDir)
	assert.NoError(t, err)
	assert.Greater(t, written, int64(0))
}

func TestUnTarFailed(t *testing.T) {
	archiveReader, err := os.Open("test_archive.tar.gz")
	require.NoError(t, err)
	defer archiveReader.Close()

	destinationDir := t.TempDir()
	t.Logf("created temp dir: %s", destinationDir)

	_, err = UnTar(archiveReader, destinationDir)
	assert.Error(t, err)
}

func TestUnTarGz(t *testing.T) {
	archiveReader, err := os.Open("test_archive.tar.gz")
	require.NoError(t, err)
	defer archiveReader.Close()

	destinationDir := t.TempDir()
	t.Logf("created temp dir: %s", destinationDir)

	written, err := UnTarGz(archiveReader, destinationDir)
	assert.NoError(t, err)
	assert.Greater(t, written, int64(0))
}

func TestUnTarGzFailed(t *testing.T) {
	archiveReader, err := os.Open("test_archive.tar")
	require.NoError(t, err)
	defer archiveReader.Close()

	destinationDir := t.TempDir()
	t.Logf("created temp dir: %s", destinationDir)

	_, err = UnTarGz(archiveReader, destinationDir)
	assert.Error(t, err)
}

func TestTar(t *testing.T) {
	archiveReader, err := os.Open("test_archive.tar")
	require.NoError(t, err)
	defer archiveReader.Close()

	destinationDir := t.TempDir()
	t.Logf("created temp dir: %s", destinationDir)

	written, err := UnTar(archiveReader, destinationDir)
	assert.NoError(t, err)
	assert.Greater(t, written, int64(0))

	destinationFile, err := os.CreateTemp(t.TempDir(), "test-tar")
	require.NoError(t, err)
	defer destinationFile.Close()

	err = Tar(destinationDir, destinationFile)
	assert.NoError(t, err)
}

func TestTarGz(t *testing.T) {
	archiveReader, err := os.Open("test_archive.tar.gz")
	require.NoError(t, err)
	defer archiveReader.Close()

	destinationDir := t.TempDir()
	t.Logf("created temp dir: %s", destinationDir)

	written, err := UnTarGz(archiveReader, destinationDir)
	assert.NoError(t, err)
	assert.Greater(t, written, int64(0))

	destinationFile, err := os.CreateTemp(t.TempDir(), "test-tar-gz")
	require.NoError(t, err)
	defer destinationFile.Close()

	err = TarGz(destinationDir, destinationFile)
	assert.NoError(t, err)
}

func TestTarGzBase64(t *testing.T) {
	archiveReader, err := os.Open("test_archive.tar.gz")
	require.NoError(t, err)
	defer archiveReader.Close()

	destinationDir := t.TempDir()
	t.Logf("created temp dir: %s", destinationDir)

	written, err := UnTarGz(archiveReader, destinationDir)
	assert.NoError(t, err)
	assert.Greater(t, written, int64(0))

	b64, err := TarGzBase64(destinationDir)
	assert.NoError(t, err)
	assert.NotEmpty(t, b64)
}

func TestUnTarGzBase64(t *testing.T) {
	archiveReader, err := os.Open("test_archive.tar.gz")
	require.NoError(t, err)
	defer archiveReader.Close()

	destinationDir := t.TempDir()
	t.Logf("created temp dir: %s", destinationDir)

	written, err := UnTarGz(archiveReader, destinationDir)
	assert.NoError(t, err)
	assert.Greater(t, written, int64(0))

	b64, err := TarGzBase64(destinationDir)
	require.NoError(t, err)
	assert.NotEmpty(t, b64)

	destinationUnTarDir := t.TempDir()

	gzBase64, err := UnTarGzBase64(b64, destinationUnTarDir)
	assert.NoError(t, err)
	assert.Greater(t, gzBase64, int64(0))
}
