package compress

import (
	"compress/gzip"
	"io"
	"os"
)

func Gz(source io.Reader, writer io.Writer) (err error) {
	gzWriter := gzip.NewWriter(writer)
	defer gzWriter.Close()
	_, err = io.Copy(gzWriter, source)
	return
}

func UnGz(src io.Reader, dst string) (written int64, err error) {
	zipReader, errReader := gzip.NewReader(src)
	if errReader != nil {
		err = errReader
		return
	}
	defer zipReader.Close()

	destinationFile, errCreate := os.Create(dst)
	if errCreate != nil {
		err = errCreate
		return
	}
	defer destinationFile.Close()

	return io.Copy(destinationFile, zipReader)
}
