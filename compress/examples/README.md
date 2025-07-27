# Compress Package Examples

This directory contains examples demonstrating how to use the `compress` package for file compression and archiving in Go applications.

## 📍 Example Code Location

**Full example implementation:** [/compress/examples/example.go](https://github.com/jasoet/pkg/blob/main/compress/examples/example.go)

## 🚀 Quick Reference for LLMs/Coding Agents

```go
// Basic usage pattern
import "github.com/jasoet/pkg/compress"

// Create tar archive
err := compress.Tar(sourceDir, tarWriter)

// Create tar.gz archive
err := compress.TarGz(sourceDir, tarGzWriter)

// Create base64-encoded tar.gz
base64String, err := compress.TarGzBase64(sourceDir)

// Extract tar archive
err := compress.Untar(tarReader, destDir)

// Extract tar.gz archive
err := compress.UntarGz(tarGzReader, destDir)

// Gzip compression
err := compress.Gzip(sourceReader, destWriter)

// Gzip decompression
err := compress.Gunzip(gzipReader, destWriter)
```

**Security features:**
- Path traversal protection in extraction
- Automatic permission sanitization
- Safe handling of symbolic links

## Overview

The `compress` package provides utilities for:
- Creating tar archives
- Creating tar.gz (compressed) archives
- Base64 encoding/decoding of compressed archives
- Extracting tar and tar.gz archives
- Direct gzip compression and decompression

## Running the Examples

To run the examples, use the following command from the `compress/examples` directory:

```bash
go run example.go
```

This will:
1. Create various types of archives from the sample data
2. Extract the archives to demonstrate decompression
3. Show error handling for invalid archives

## Example Descriptions

The [example.go](https://github.com/jasoet/pkg/blob/main/compress/examples/example.go) file demonstrates several use cases:

### 1. Creating a tar archive

```go
tarFile, err := os.Create("archive.tar")
defer tarFile.Close()
err = compress.Tar(sourceDirectory, tarFile)
```

### 2. Creating a tar.gz archive

```go
tarGzFile, err := os.Create("archive.tar.gz")
defer tarGzFile.Close()
err = compress.TarGz(sourceDirectory, tarGzFile)
```

### 3. Creating a base64-encoded tar.gz archive

```go
base64Encoded, err := compress.TarGzBase64(sourceDirectory)
```

### 4. Extracting a tar archive

```go
tarFile, err := os.Open("archive.tar")
defer tarFile.Close()
written, err := compress.UnTar(tarFile, extractDirectory)
```

### 5. Extracting a tar.gz archive

```go
tarGzFile, err := os.Open("archive.tar.gz")
defer tarGzFile.Close()
written, err := compress.UnTarGz(tarGzFile, extractDirectory)
```

### 6. Extracting a base64-encoded tar.gz archive

```go
written, err := compress.UnTarGzBase64(base64EncodedString, extractDirectory)
```

### 7. Using gzip compression directly

```go
sourceFile, err := os.Open("file.txt")
defer sourceFile.Close()
gzFile, err := os.Create("file.txt.gz")
defer gzFile.Close()
err = compress.Gz(sourceFile, gzFile)
```

### 8. Decompressing a gzip file directly

```go
gzFile, err := os.Open("file.txt.gz")
defer gzFile.Close()
written, err := compress.UnGz(gzFile, "decompressed_file.txt")
```

### 9. Error handling

The examples demonstrate proper error handling for all operations, including handling invalid archives.

## Sample Data Structure

The examples use a sample directory structure:
- `sample_data/file1.txt` - A simple text file
- `sample_data/file2.txt` - Another text file
- `sample_data/nested/nested_file.txt` - A file in a nested directory

This structure demonstrates that the compress package can handle nested directories and preserves the directory structure when extracting archives.

## Security Considerations

The compress package includes security measures to prevent path traversal attacks when extracting archives. It validates paths before extraction to ensure they don't contain:
- Absolute paths (starting with `/`)
- Path traversal sequences (`../`)
- Backslashes (`\`)

## Key Features

- **Tar Archives**: Create and extract standard tar archives
- **Compressed Archives**: Create and extract gzip-compressed tar archives
- **Base64 Encoding**: Convert compressed archives to/from base64-encoded strings
- **Direct Gzip**: Compress and decompress individual files with gzip
- **Security**: Protection against path traversal attacks
- **Error Handling**: Comprehensive error handling for all operations