//go:build example

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jasoet/pkg/compress"
)

func check(err error) {
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	// Get the current directory
	currentDir, err := os.Getwd()
	check(err)

	// Define paths
	sampleDataDir := filepath.Join(currentDir, "sample_data")
	outputDir := filepath.Join(currentDir, "output")

	// Create output directory if it doesn't exist
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		err = os.MkdirAll(outputDir, 0755)
		check(err)
	}

	// Example 1: Create a tar archive
	fmt.Println("Example 1: Creating a tar archive")
	tarFilePath := filepath.Join(outputDir, "archive.tar")
	tarFile, err := os.Create(tarFilePath)
	check(err)
	defer tarFile.Close()

	err = compress.Tar(sampleDataDir, tarFile)
	check(err)
	fmt.Printf("Created tar archive: %s\n\n", tarFilePath)

	// Example 2: Create a tar.gz archive
	fmt.Println("Example 2: Creating a tar.gz archive")
	tarGzFilePath := filepath.Join(outputDir, "archive.tar.gz")
	tarGzFile, err := os.Create(tarGzFilePath)
	check(err)
	defer tarGzFile.Close()

	err = compress.TarGz(sampleDataDir, tarGzFile)
	check(err)
	fmt.Printf("Created tar.gz archive: %s\n\n", tarGzFilePath)

	// Example 3: Create a base64-encoded tar.gz archive
	fmt.Println("Example 3: Creating a base64-encoded tar.gz archive")
	base64Encoded, err := compress.TarGzBase64(sampleDataDir)
	check(err)
	fmt.Printf("Created base64-encoded tar.gz archive (first 50 chars): %s...\n\n", base64Encoded[:50])

	// Example 4: Extract a tar archive
	fmt.Println("Example 4: Extracting a tar archive")
	extractTarDir := filepath.Join(outputDir, "extracted_tar")
	if _, err := os.Stat(extractTarDir); os.IsNotExist(err) {
		err = os.MkdirAll(extractTarDir, 0755)
		check(err)
	}

	tarFileToExtract, err := os.Open(tarFilePath)
	check(err)
	defer tarFileToExtract.Close()

	written, err := compress.UnTar(tarFileToExtract, extractTarDir)
	check(err)
	fmt.Printf("Extracted tar archive to: %s (bytes written: %d)\n\n", extractTarDir, written)

	// Example 5: Extract a tar.gz archive
	fmt.Println("Example 5: Extracting a tar.gz archive")
	extractTarGzDir := filepath.Join(outputDir, "extracted_tar_gz")
	if _, err := os.Stat(extractTarGzDir); os.IsNotExist(err) {
		err = os.MkdirAll(extractTarGzDir, 0755)
		check(err)
	}

	tarGzFileToExtract, err := os.Open(tarGzFilePath)
	check(err)
	defer tarGzFileToExtract.Close()

	written, err = compress.UnTarGz(tarGzFileToExtract, extractTarGzDir)
	check(err)
	fmt.Printf("Extracted tar.gz archive to: %s (bytes written: %d)\n\n", extractTarGzDir, written)

	// Example 6: Extract a base64-encoded tar.gz archive
	fmt.Println("Example 6: Extracting a base64-encoded tar.gz archive")
	extractBase64Dir := filepath.Join(outputDir, "extracted_base64")
	if _, err := os.Stat(extractBase64Dir); os.IsNotExist(err) {
		err = os.MkdirAll(extractBase64Dir, 0755)
		check(err)
	}

	written, err = compress.UnTarGzBase64(base64Encoded, extractBase64Dir)
	check(err)
	fmt.Printf("Extracted base64-encoded tar.gz archive to: %s (bytes written: %d)\n\n", extractBase64Dir, written)

	// Example 7: Using gzip compression directly
	fmt.Println("Example 7: Using gzip compression directly")
	textFilePath := filepath.Join(sampleDataDir, "file1.txt")
	gzFilePath := filepath.Join(outputDir, "file1.txt.gz")

	textFile, err := os.Open(textFilePath)
	check(err)
	defer textFile.Close()

	gzFile, err := os.Create(gzFilePath)
	check(err)
	defer gzFile.Close()

	err = compress.Gz(textFile, gzFile)
	check(err)
	fmt.Printf("Created gzip file: %s\n\n", gzFilePath)

	// Example 8: Decompressing a gzip file directly
	fmt.Println("Example 8: Decompressing a gzip file directly")
	gzFileToDecompress, err := os.Open(gzFilePath)
	check(err)
	defer gzFileToDecompress.Close()

	decompressedFilePath := filepath.Join(outputDir, "decompressed_file1.txt")
	written, err = compress.UnGz(gzFileToDecompress, decompressedFilePath)
	check(err)
	fmt.Printf("Decompressed gzip file to: %s (bytes written: %d)\n\n", decompressedFilePath, written)

	// Example 9: Error handling - trying to extract an invalid archive
	fmt.Println("Example 9: Error handling - trying to extract an invalid archive")
	invalidFilePath := filepath.Join(sampleDataDir, "file1.txt") // Not a tar archive
	invalidFile, err := os.Open(invalidFilePath)
	check(err)
	defer invalidFile.Close()

	extractInvalidDir := filepath.Join(outputDir, "extracted_invalid")
	if _, err := os.Stat(extractInvalidDir); os.IsNotExist(err) {
		err = os.MkdirAll(extractInvalidDir, 0755)
		check(err)
	}

	_, err = compress.UnTar(invalidFile, extractInvalidDir)
	if err != nil {
		fmt.Printf("Expected error occurred: %v\n", err)
	} else {
		fmt.Println("Error: Expected an error but none occurred")
	}

	fmt.Println("\nAll examples completed successfully!")
}
