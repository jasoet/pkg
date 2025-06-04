package compress

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGz(t *testing.T) {
	archiveReader, err := os.Open("test_archive.tar")
	assert.Nil(t, err)
	assert.NotNil(t, archiveReader)

	destinationFile, err := ioutil.TempFile("/tmp", "test-gz")
	assert.Nil(t, err)
	err = Gz(archiveReader, destinationFile)
	assert.Nil(t, err)

	readFile, err := os.Open(destinationFile.Name())
	assert.Nil(t, err)

	destinationDir, err := ioutil.TempDir("/tmp", "test-gs")
	_, err = UnTarGz(readFile, destinationDir)
	assert.Nil(t, err)
}

func TestUnGz(t *testing.T) {
	archiveReader, err := os.Open("test_archive.tar.gz")
	assert.Nil(t, err)
	assert.NotNil(t, archiveReader)

	destinationFile, err := ioutil.TempFile("/tmp", "test-gz")
	assert.Nil(t, err)
	_, err = UnGz(archiveReader, destinationFile.Name())
	assert.Nil(t, err)

	readFile, err := os.Open(destinationFile.Name())
	assert.Nil(t, err)

	destinationDir, err := ioutil.TempDir("/tmp", "test-gs")
	_, err = UnTar(readFile, destinationDir)
	assert.Nil(t, err)
}
