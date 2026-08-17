package compress_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jasoet/pkg/v3/compress"
)

// Gz compresses data from any io.Reader into any io.Writer.
func ExampleGz() {
	var buf bytes.Buffer
	if err := compress.Gz(bytes.NewReader([]byte("hello, world")), &buf); err != nil {
		panic(err)
	}

	fmt.Println("compressed bytes:", buf.Len())

	// Output:
	// compressed bytes: 36
}

// UnGz decompresses a gzip stream into a file at an absolute destination path.
func ExampleUnGz() {
	var buf bytes.Buffer
	if err := compress.Gz(bytes.NewReader([]byte("hello, world")), &buf); err != nil {
		panic(err)
	}

	dir, err := os.MkdirTemp("", "compress-example")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	dst := filepath.Join(dir, "out.txt")
	written, err := compress.UnGz(&buf, dst)
	if err != nil {
		panic(err)
	}

	content, err := os.ReadFile(dst)
	if err != nil {
		panic(err)
	}
	fmt.Printf("wrote %d bytes: %s\n", written, content)

	// Output:
	// wrote 12 bytes: hello, world
}
