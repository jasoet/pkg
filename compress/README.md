# Compress Package

[![Go Reference](https://pkg.go.dev/badge/github.com/jasoet/pkg/v3/compress.svg)](https://pkg.go.dev/github.com/jasoet/pkg/v3/compress)

Gzip and tar archive utilities with built-in protection against path traversal and zip bomb attacks.

## Features

- **Gzip**: stream compression (`Gz`) and decompression to a file (`UnGz`)
- **Tar**: directory archiving (`Tar`) and extraction (`UnTar`)
- **Tar.gz**: combined helpers (`TarGz`, `UnTarGz`)
- **Base64**: tar.gz archives as base64 strings (`TarGzBase64`, `UnTarGzBase64`)
- **Security hardened**: path traversal prevention, symlink resolution checks, per-file and total size limits, file mode sanitization
- **Error contract**: every guard-rail rejection wraps a documented sentinel, matchable with `errors.Is`

## Installation

```bash
go get github.com/jasoet/pkg/v3/compress
```

## Quick Start

```go
package main

import (
    "log"
    "os"
    "path/filepath"

    "github.com/jasoet/pkg/v3/compress"
)

func main() {
    // Compress a file with gzip.
    sourceFile, err := os.Open("input.txt")
    if err != nil {
        log.Fatal(err)
    }
    defer sourceFile.Close()

    outputFile, err := os.Create("output.txt.gz")
    if err != nil {
        log.Fatal(err)
    }
    defer outputFile.Close()

    if err := compress.Gz(sourceFile, outputFile); err != nil {
        log.Fatal(err)
    }

    // Decompress it again. UnGz requires an ABSOLUTE destination path.
    gzFile, err := os.Open("output.txt.gz")
    if err != nil {
        log.Fatal(err)
    }
    defer gzFile.Close()

    dst, err := filepath.Abs("decompressed.txt")
    if err != nil {
        log.Fatal(err)
    }
    if _, err := compress.UnGz(gzFile, dst); err != nil {
        log.Fatal(err)
    }
}
```

## API Reference

### Gzip

```go
func Gz(source io.Reader, writer io.Writer) error
func UnGz(src io.Reader, dst string, opts ...ExtractOption) (int64, error)
```

`Gz` streams gzip-compressed data from `source` into `writer`.

`UnGz` decompresses a gzip stream into the file at `dst` and returns the number
of bytes written. **`dst` must be an absolute path**; relative paths are
rejected with `ErrPathTraversal`. A gzip stream holds a single file, so both
size options apply to the same output — the effective limit is the smaller of
`WithMaxFileSize` and `WithMaxArchiveSize`.

### Tar

```go
func Tar(sourceDirectory string, writer io.Writer) error
func UnTar(src io.Reader, destinationDir string, opts ...ExtractOption) (int64, error)
```

`Tar` archives `sourceDirectory` into `writer`. Only regular files are
included; symlinks and other special files are skipped.

`UnTar` extracts a tar stream into `destinationDir`, which must already exist
and be a directory. Unlike `UnGz`, `destinationDir` may be a relative path —
entry paths inside the archive are validated to stay within it. Only regular
files and directories are extracted; other entry types are skipped.

### Tar.gz

```go
func TarGz(sourceDirectory string, writer io.Writer) error
func UnTarGz(src io.Reader, destinationDir string, opts ...ExtractOption) (int64, error)
```

Combined helpers: `UnTarGz` gunzips `src` and extracts it like `UnTar`.

### Base64

```go
func TarGzBase64(sourceDirectory string) (string, error)
func UnTarGzBase64(encoded string, destinationDir string, opts ...ExtractOption) (int64, error)
```

`TarGzBase64` archives and compresses a directory, returning it as a
base64-encoded string for text transport (JSON, API responses).
`UnTarGzBase64` reverses it, extracting like `UnTarGz`.

## Options

All extraction functions (`UnGz`, `UnTar`, `UnTarGz`, `UnTarGzBase64`) accept
`ExtractOption`s:

| Option | Default | Effect |
| --- | --- | --- |
| `WithMaxFileSize(size int64)` | 100 MB (`DefaultMaxFileSize`) | Maximum decompressed size of a single file |
| `WithMaxArchiveSize(size int64)` | 1 GB (`DefaultMaxArchiveSize`) | Maximum total extracted size of an archive |

For `UnGz` (single-file stream) the effective limit is `min(maxFileSize, maxArchiveSize)`.

```go
// Allow single files up to 500 MB, archive total up to 2 GB.
written, err := compress.UnTarGz(reader, destDir,
    compress.WithMaxFileSize(500*1024*1024),
    compress.WithMaxArchiveSize(2*1024*1024*1024),
)
```

## Error Handling

Guard-rail rejections wrap documented sentinels — match them with `errors.Is`,
never by comparing message strings:

| Sentinel | Returned when |
| --- | --- |
| `ErrPathTraversal` | `UnGz` destination is not absolute; a tar entry path is empty, absolute, contains `..` or `\`, escapes the destination, or resolves through a symlink outside it |
| `ErrSizeLimitExceeded` | A file exceeds `maxFileSize`, or the archive total exceeds `maxArchiveSize` |
| `ErrNotDirectory` | `Tar` source or `UnTar`/`UnTarGz` destination is not a directory |

```go
written, err := compress.UnTarGz(reader, destDir)
switch {
case errors.Is(err, compress.ErrPathTraversal):
    // Malicious or malformed entry path — reject the archive.
case errors.Is(err, compress.ErrSizeLimitExceeded):
    // Zip bomb protection triggered; written holds bytes extracted so far.
case errors.Is(err, compress.ErrNotDirectory):
    // Fix the destination and retry.
case err != nil:
    // I/O or corrupt-archive error (e.g. gzip.ErrHeader, io.ErrUnexpectedEOF).
}
```

Missing destinations, corrupt archives, and filesystem failures surface as the
underlying `os`/`gzip`/`tar` errors and are matchable with `errors.Is` against
`fs.ErrNotExist` and friends.

## Security Details

- **Path traversal prevention**: tar entry names are rejected when empty,
  absolute, or containing `..` or `\`; the joined target path must stay under
  the destination; parent directories are resolved with
  `filepath.EvalSymlinks` to stop symlink TOCTOU escapes.
- **Zip bomb protection**: extraction streams through `io.LimitReader`; one
  extra byte is probed past the limit so oversized content is detected and
  reported with `ErrSizeLimitExceeded`.
- **File mode sanitization**: extracted file modes are masked with `0o777`,
  stripping setuid/setgid/sticky bits; directories are created `0o750`.
- **`UnGz` vs `UnTar` path rules**: `UnGz` requires an absolute destination
  path, while `UnTar` accepts a relative destination directory. The asymmetry
  is intentional: `UnGz` writes to a caller-supplied file path and fails
  closed on ambiguity, whereas `UnTar` constrains archive-controlled entry
  paths inside the destination instead.

## Testing

```bash
go test ./compress/ -count=1        # all tests, including security suite
go test ./compress/ -v -run TestGuardRailSentinels
```

## License

MIT License - see [LICENSE](../LICENSE) for details.
