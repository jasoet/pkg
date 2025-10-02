# Compress Package

[![Go Reference](https://pkg.go.dev/badge/github.com/jasoet/pkg/v2/compress.svg)](https://pkg.go.dev/github.com/jasoet/pkg/v2/compress)

Secure file compression and decompression utilities with built-in protection against path traversal and zip bomb attacks.

## Overview

The `compress` package provides production-ready compression utilities for Gzip and Tar formats with comprehensive security validations. It includes protection against common security vulnerabilities including path traversal attacks and zip bombs.

## Features

- **Gzip Compression**: Fast single-file compression
- **Tar Archives**: Directory archiving with path preservation
- **Tar.gz Support**: Combined tar + gzip compression
- **Base64 Encoding**: Tar.gz archives encoded as base64 strings
- **Security Hardened**: Path traversal and zip bomb protection
- **100MB Safety Limit**: Prevents decompression bombs
- **Safe File Modes**: Validates and sanitizes file permissions

## Installation

```bash
go get github.com/jasoet/pkg/v2/compress
```

## Quick Start

### Gzip Compression

```go
package main

import (
    "os"
    "github.com/jasoet/pkg/v2/compress"
)

func main() {
    // Compress file
    sourceFile, _ := os.Open("input.txt")
    defer sourceFile.Close()

    outputFile, _ := os.Create("output.txt.gz")
    defer outputFile.Close()

    compress.Gz(sourceFile, outputFile)

    // Decompress file
    gzFile, _ := os.Open("output.txt.gz")
    defer gzFile.Close()

    compress.UnGz(gzFile, "decompressed.txt")
}
```

### Tar Archives

```go
import (
    "os"
    "github.com/jasoet/pkg/v2/compress"
)

// Create tar archive
outputFile, _ := os.Create("archive.tar")
defer outputFile.Close()

compress.Tar("/path/to/directory", outputFile)

// Extract tar archive
tarFile, _ := os.Open("archive.tar")
defer tarFile.Close()

compress.UnTar(tarFile, "/path/to/destination")
```

### Tar.gz (Combined)

```go
// Create tar.gz archive
outputFile, _ := os.Create("archive.tar.gz")
defer outputFile.Close()

compress.TarGz("/path/to/directory", outputFile)

// Extract tar.gz archive
tarGzFile, _ := os.Open("archive.tar.gz")
defer tarGzFile.Close()

compress.UnTarGz(tarGzFile, "/path/to/destination")
```

### Base64 Encoded Archives

```go
// Compress directory to base64 string
encoded, err := compress.TarGzBase64("/path/to/directory")
if err != nil {
    panic(err)
}

// Store or transmit encoded string
fmt.Println(encoded)

// Decompress from base64 string
written, err := compress.UnTarGzBase64(encoded, "/path/to/destination")
if err != nil {
    panic(err)
}

fmt.Printf("Wrote %d bytes\n", written)
```

## API Reference

### Gzip Functions

#### Gz

Compress data using gzip:

```go
func Gz(source io.Reader, writer io.Writer) error
```

**Example:**
```go
source, _ := os.Open("input.txt")
dest, _ := os.Create("output.gz")
compress.Gz(source, dest)
```

#### UnGz

Decompress gzip data with security checks:

```go
func UnGz(src io.Reader, dst string) (written int64, err error)
```

**Security Features:**
- Path traversal prevention (blocks `..`)
- 100MB decompression limit (zip bomb protection)

**Example:**
```go
gzFile, _ := os.Open("file.gz")
written, err := compress.UnGz(gzFile, "output.txt")
```

### Tar Functions

#### Tar

Create tar archive from directory:

```go
func Tar(sourceDirectory string, writer io.Writer) error
```

**Example:**
```go
outputFile, _ := os.Create("archive.tar")
compress.Tar("/my/directory", outputFile)
```

#### UnTar

Extract tar archive with security validation:

```go
func UnTar(src io.Reader, destinationDir string) (written int64, err error)
```

**Security Features:**
- Path traversal prevention
- Safe file mode validation (capped at 0o777)
- 100MB per-file limit

**Example:**
```go
tarFile, _ := os.Open("archive.tar")
written, err := compress.UnTar(tarFile, "/extract/here")
```

### Tar.gz Functions

#### TarGz

Create tar.gz archive:

```go
func TarGz(sourceDirectory string, writer io.Writer) error
```

**Example:**
```go
outputFile, _ := os.Create("archive.tar.gz")
compress.TarGz("/my/directory", outputFile)
```

#### UnTarGz

Extract tar.gz archive:

```go
func UnTarGz(src io.Reader, destinationDir string) (totalWritten int64, err error)
```

**Example:**
```go
tarGzFile, _ := os.Open("archive.tar.gz")
written, err := compress.UnTarGz(tarGzFile, "/extract/here")
```

### Base64 Functions

#### TarGzBase64

Compress directory to base64-encoded tar.gz string:

```go
func TarGzBase64(sourceDirectory string) (string, error)
```

**Use Case**: Transmit compressed directories via text protocols (JSON, API responses)

**Example:**
```go
encoded, err := compress.TarGzBase64("/my/directory")
// Send encoded string via API
```

#### UnTarGzBase64

Extract from base64-encoded tar.gz string:

```go
func UnTarGzBase64(encoded string, destinationDir string) (totalWritten int64, err error)
```

**Example:**
```go
// Receive encoded string from API
written, err := compress.UnTarGzBase64(encoded, "/extract/here")
```

## Security Features

### Path Traversal Protection

Prevents malicious archives from writing outside destination:

```go
// ✅ Protected: These paths are blocked
"../etc/passwd"          // Blocked: Contains ..
"/etc/passwd"            // Blocked: Absolute path
"dir/../../../etc/pass"  // Blocked: Traversal attempt
```

**Implementation:**
```go
// Path validation
if strings.Contains(path, "..") {
    return fmt.Errorf("invalid path")
}

// Ensure within destination
if !strings.HasPrefix(target, destinationDir) {
    return fmt.Errorf("path traversal attempt")
}
```

### Zip Bomb Protection

Limits decompression to prevent resource exhaustion:

```go
// 100MB limit per file
limitedReader := io.LimitReader(reader, 100*1024*1024)
io.Copy(dest, limitedReader)
```

**Why:**
- Small compressed file (1KB) can expand to gigabytes
- Exhausts disk space and memory
- Causes denial of service

**Protection:**
- Each file limited to 100MB decompressed
- Error returned if limit exceeded

### File Mode Validation

Sanitizes file permissions to prevent dangerous modes:

```go
// Cap at 0o777, use safe default for invalid modes
fileMode := header.Mode
if fileMode > 0o777 {
    fileMode = 0o644 // Safe default
}
safeMode := os.FileMode(fileMode & 0o777)
```

**Why:**
- Prevents setuid/setgid bits
- Prevents unsafe permissions
- Ensures consistent file modes

## Advanced Usage

### Streaming Compression

```go
// Compress from any reader
httpResponse, _ := http.Get("https://example.com/large-file")
defer httpResponse.Body.Close()

gzFile, _ := os.Create("output.gz")
defer gzFile.Close()

compress.Gz(httpResponse.Body, gzFile)
```

### Custom Writer

```go
// Compress to bytes buffer
var buf bytes.Buffer
compress.Gz(sourceReader, &buf)

// Compress to network connection
conn, _ := net.Dial("tcp", "server:8080")
compress.TarGz("/my/directory", conn)
```

### Directory Filtering

For selective archiving, walk directory manually:

```go
outputFile, _ := os.Create("filtered.tar")
tarWriter := tar.NewWriter(outputFile)
defer tarWriter.Close()

filepath.Walk("/my/dir", func(path string, info os.FileInfo, err error) error {
    // Skip .git directories
    if info.IsDir() && info.Name() == ".git" {
        return filepath.SkipDir
    }

    // Only include .go files
    if !info.IsDir() && filepath.Ext(path) == ".go" {
        // Add to tar manually
    }

    return nil
})
```

## Error Handling

```go
// Gzip decompression
written, err := compress.UnGz(reader, "output.txt")
if err != nil {
    switch {
    case strings.Contains(err.Error(), "invalid destination"):
        // Path traversal attempt
    case strings.Contains(err.Error(), "unexpected EOF"):
        // Corrupted archive
    default:
        // Other errors
    }
}

// Tar extraction
written, err := compress.UnTar(reader, "/dest")
if err != nil {
    switch {
    case strings.Contains(err.Error(), "invalid path"):
        // Path traversal attempt
    case strings.Contains(err.Error(), "not a directory"):
        // Destination is not a directory
    default:
        // Other errors
    }
}
```

## Best Practices

### 1. Validate Destination

```go
// ✅ Good: Check destination exists and is directory
info, err := os.Stat(destDir)
if err != nil {
    return err
}
if !info.IsDir() {
    return fmt.Errorf("destination must be directory")
}

compress.UnTarGz(reader, destDir)
```

### 2. Handle Large Files

```go
// ✅ Good: Stream large files
source, _ := os.Open("large-file.txt")
defer source.Close()

dest, _ := os.Create("output.gz")
defer dest.Close()

compress.Gz(source, dest) // Streams, low memory
```

### 3. Close Writers

```go
// ✅ Good: Ensure writers are closed
outputFile, _ := os.Create("archive.tar.gz")
defer outputFile.Close()

if err := compress.TarGz("/my/dir", outputFile); err != nil {
    return err
}
// Deferred close ensures data is flushed
```

### 4. Check Written Bytes

```go
// ✅ Good: Verify extraction
written, err := compress.UnTarGz(reader, "/dest")
if err != nil {
    return err
}

if written == 0 {
    log.Warn("No files extracted")
}
log.Printf("Extracted %d bytes", written)
```

### 5. Use Base64 for APIs

```go
// ✅ Good: Base64 for text transport
type Response struct {
    Archive string `json:"archive"`
}

encoded, _ := compress.TarGzBase64("/data")
response := Response{Archive: encoded}
json.Marshal(response)
```

## Testing

The package includes comprehensive tests with 86% coverage:

```bash
# Run tests
go test ./compress -v

# With coverage
go test ./compress -cover

# Security tests
go test ./compress -v -run TestSecurity
```

### Test Utilities

```go
func TestMyCompression(t *testing.T) {
    // Create temp directory
    tmpDir, _ := os.MkdirTemp("", "compress-test")
    defer os.RemoveAll(tmpDir)

    // Create test file
    testFile := filepath.Join(tmpDir, "test.txt")
    os.WriteFile(testFile, []byte("content"), 0o644)

    // Test compression
    var buf bytes.Buffer
    err := compress.Tar(tmpDir, &buf)
    assert.NoError(t, err)

    // Test decompression
    destDir, _ := os.MkdirTemp("", "extract")
    defer os.RemoveAll(destDir)

    written, err := compress.UnTar(&buf, destDir)
    assert.NoError(t, err)
    assert.Greater(t, written, int64(0))
}
```

## Troubleshooting

### Path Traversal Errors

**Problem**: `invalid path` or `path traversal` error

**Solution:**
```go
// Ensure clean paths
destDir := filepath.Clean("/my/destination")
written, err := compress.UnTar(reader, destDir)
```

### Zip Bomb Detection

**Problem**: Extraction stops at 100MB

**Solution:**
```go
// This is intentional security protection
// If you need larger files, extract programmatically:

tarReader := tar.NewReader(gzipReader)
for {
    header, err := tarReader.Next()
    if err == io.EOF {
        break
    }

    // Custom size limit
    limitedReader := io.LimitReader(tarReader, 500*1024*1024) // 500MB
    io.Copy(outputFile, limitedReader)
}
```

### Corrupted Archives

**Problem**: `unexpected EOF` or `invalid header`

**Solution:**
```go
// Verify archive integrity before processing
file, _ := os.Open("archive.tar.gz")
gzReader, err := gzip.NewReader(file)
if err != nil {
    return fmt.Errorf("not a valid gzip: %w", err)
}

tarReader := tar.NewReader(gzReader)
_, err = tarReader.Next()
if err != nil {
    return fmt.Errorf("not a valid tar: %w", err)
}
```

## Performance

- **Streaming**: Low memory usage for large files
- **Efficient**: Uses standard library compression
- **Minimal Overhead**: Security checks are fast (~microseconds)

**Benchmark:**
```
BenchmarkGz-8           1000    ~1ms/op (per MB)
BenchmarkTar-8          2000    ~500µs/op (per file)
BenchmarkSecurityCheck-8 100000  ~10µs/op (path validation)
```

## Examples

See [examples/](./examples/) directory for:
- File compression and decompression
- Directory archiving
- Base64 encoding for APIs
- Security edge cases
- Error handling patterns

## Related Packages

- **[config](../config/)** - Configuration management
- **[ssh](../ssh/)** - SSH file transfer

## License

MIT License - see [LICENSE](../LICENSE) for details.
