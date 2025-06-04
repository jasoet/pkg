package compress

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnTar(t *testing.T) {
	archiveReader, err := os.Open("test_archive.tar")
	assert.Nil(t, err)
	assert.NotNil(t, archiveReader)

	destinationDir, err := ioutil.TempDir("/tmp", "test-un-tar")
	fmt.Printf("created temp dir: %s", destinationDir)
	assert.Nil(t, err)
	assert.NotNil(t, destinationDir)

	written, err := UnTar(archiveReader, destinationDir)
	assert.Nil(t, err)
	assert.NotNil(t, written)
}

func TestUnTarFailed(t *testing.T) {
	archiveReader, err := os.Open("test_archive.tar.gz")
	assert.Nil(t, err)
	assert.NotNil(t, archiveReader)

	destinationDir, err := ioutil.TempDir("/tmp", "test-un-tar")
	fmt.Printf("created temp dir: %s", destinationDir)
	assert.Nil(t, err)
	assert.NotNil(t, destinationDir)

	_, err = UnTar(archiveReader, destinationDir)
	assert.NotNil(t, err)
}

func TestUnTarGz(t *testing.T) {
	archiveReader, err := os.Open("test_archive.tar.gz")
	assert.Nil(t, err)
	assert.NotNil(t, archiveReader)

	destinationDir, err := ioutil.TempDir("/tmp", "test-un-tar-gz")
	fmt.Printf("created temp dir: %s", destinationDir)
	assert.Nil(t, err)
	assert.NotNil(t, destinationDir)

	written, err := UnTarGz(archiveReader, destinationDir)
	assert.Nil(t, err)
	assert.NotNil(t, written)
}

func TestUnTarGzFailed(t *testing.T) {
	archiveReader, err := os.Open("test_archive.tar")
	assert.Nil(t, err)
	assert.NotNil(t, archiveReader)

	destinationDir, err := ioutil.TempDir("/tmp", "test-un-tar-gz")
	fmt.Printf("created temp dir: %s", destinationDir)
	assert.Nil(t, err)
	assert.NotNil(t, destinationDir)

	_, err = UnTarGz(archiveReader, destinationDir)
	assert.NotNil(t, err)
}

func TestTar(t *testing.T) {
	archiveReader, err := os.Open("test_archive.tar")
	assert.Nil(t, err)
	assert.NotNil(t, archiveReader)

	destinationDir, err := ioutil.TempDir("/tmp", "test-tar")
	fmt.Printf("created temp dir: %s", destinationDir)
	assert.Nil(t, err)
	assert.NotNil(t, destinationDir)

	written, err := UnTar(archiveReader, destinationDir)
	assert.Nil(t, err)
	assert.NotNil(t, written)

	destinationFile, err := ioutil.TempFile("/tmp", "test-tar")
	assert.Nil(t, err)

	err = Tar(destinationDir, destinationFile)
	assert.Nil(t, err)
}

func TestTarGz(t *testing.T) {
	archiveReader, err := os.Open("test_archive.tar.gz")
	assert.Nil(t, err)
	assert.NotNil(t, archiveReader)

	destinationDir, err := ioutil.TempDir("/tmp", "test-tar-gz")
	fmt.Printf("created temp dir: %s", destinationDir)
	assert.Nil(t, err)
	assert.NotNil(t, destinationDir)

	written, err := UnTarGz(archiveReader, destinationDir)
	assert.Nil(t, err)
	assert.NotNil(t, written)

	destinationFile, err := ioutil.TempFile("/tmp", "test-tar-gz")
	assert.Nil(t, err)

	err = TarGz(destinationDir, destinationFile)
	assert.Nil(t, err)
}

func TestTarGzBase64(t *testing.T) {
	archiveReader, err := os.Open("test_archive.tar.gz")
	assert.Nil(t, err)
	assert.NotNil(t, archiveReader)

	destinationDir, err := ioutil.TempDir("/tmp", "test-tar-gz")
	fmt.Printf("created temp dir: %s", destinationDir)
	assert.Nil(t, err)
	assert.NotNil(t, destinationDir)

	written, err := UnTarGz(archiveReader, destinationDir)
	assert.Nil(t, err)
	assert.NotNil(t, written)

	base64, err := TarGzBase64(destinationDir)
	assert.Nil(t, err)
	assert.NotNil(t, base64)
}
func TestUnTarGzBase64(t *testing.T) {
	archiveReader, err := os.Open("test_archive.tar.gz")
	assert.Nil(t, err)
	assert.NotNil(t, archiveReader)

	destinationDir, err := ioutil.TempDir("/tmp", "test-tar-gz")
	fmt.Printf("created temp dir: %s", destinationDir)
	assert.Nil(t, err)
	assert.NotNil(t, destinationDir)

	written, err := UnTarGz(archiveReader, destinationDir)
	assert.Nil(t, err)
	assert.NotNil(t, written)

	base64, err := TarGzBase64(destinationDir)
	assert.Nil(t, err)
	assert.NotNil(t, base64)

	destinationUnTarDir, err := os.MkdirTemp("/tmp", "test-untar-gz")
	assert.Nil(t, err)
	assert.NotNil(t, base64)

	gzBase64, err := UnTarGzBase64(base64, destinationUnTarDir)
	assert.Nil(t, err)
	assert.NotNil(t, gzBase64)
}
