package compress

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGz(t *testing.T) {
	archiveReader, err := os.Open("test_archive.tar")
	require.NoError(t, err)
	defer archiveReader.Close()

	destinationFile, err := os.CreateTemp(t.TempDir(), "test-gz")
	require.NoError(t, err)
	defer destinationFile.Close()

	err = Gz(archiveReader, destinationFile)
	assert.NoError(t, err)

	readFile, err := os.Open(destinationFile.Name())
	require.NoError(t, err)
	defer readFile.Close()

	destinationDir := t.TempDir()
	_, err = UnTarGz(readFile, destinationDir)
	assert.NoError(t, err)
}

func TestUnGz(t *testing.T) {
	archiveReader, err := os.Open("test_archive.tar.gz")
	require.NoError(t, err)
	defer archiveReader.Close()

	destPath := t.TempDir() + "/test-ungz"
	_, err = UnGz(archiveReader, destPath)
	assert.NoError(t, err)

	readFile, err := os.Open(destPath)
	require.NoError(t, err)
	defer readFile.Close()

	destinationDir := t.TempDir()
	_, err = UnTar(readFile, destinationDir)
	assert.NoError(t, err)
}
