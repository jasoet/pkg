# Code Review: `compress` Package

**Date:** 2026-03-21

## Package Summary

Provides gzip and tar/tar.gz/base64-encoded-tar.gz compression and extraction utilities. Advertises security protections against path traversal (zip slip) and decompression bombs. Two source files: `gz.go` and `tar.go` (~266 lines of production code).

---

## Issues Found

### Critical

**C1 — Silent truncation on decompression bomb — no error returned** (`gz.go:54`, `tar.go:168`)

`io.LimitReader` silently truncates output when limit is hit. `io.Copy` returns success even when limit was reached. Caller has no way to distinguish "file extracted fine" from "file was silently cut at 100 MB."

**Fix:** After `io.Copy`, check `written >= maxFileSize` and verify underlying reader is exhausted. Return `ErrSizeLimitExceeded`.

**C2 — Zip slip via symlink TOCTOU** (`tar.go:243`)

Path check is purely string-based — does not verify the final resolved path on disk. If a symlink already exists in `destinationDir` before extraction, writes could follow it outside the destination.

**Fix:** Use `filepath.EvalSymlinks` on the parent directory before writing each file.

### High

**H1 — `UnGz` path traversal check is fragile and can be bypassed** (`gz.go:34-36`)

```go
cleanDst := filepath.Clean(dst)
if strings.Contains(cleanDst, "..") { ... }
```

After `filepath.Clean`, `..` is resolved away on most paths, making the check effectively dead code.

**Fix:** Require `dst` to be absolute, or remove the check and document caller responsibility.

**H2 — `Tar` uses `strings.Replace` for path stripping** (`tar.go:41`)

`strings.Replace(file, sourceDirectory, "", -1)` replaces ALL occurrences, not just the prefix. If `sourceDirectory` appears elsewhere in the path, inner occurrences are also stripped.

**Fix:** Use `filepath.Rel(sourceDirectory, file)`.

### Medium

- M1: Errors not wrapped with `%w` — loss of error context; callers resort to string matching
- M2: `Tar` doesn't validate `sourceDirectory` is actually a directory
- M3: `UnTarGzBase64` decodes entire archive into memory (should stream)
- M4: File mode sanitization logic is redundant and misleadingly commented

### Low

- L1: `extractTarDirectory` has unnecessary `os.Stat` before `os.MkdirAll` (TOCTOU, plus MkdirAll is idempotent)
- L2: `validTarPath` checks for backslash — may reject valid Unix filenames
- L3: Named return values used inconsistently
- L4: Test intentionally skips an assertion
- L5: `UnGz` has hard-coded 100 MB limit with no override (unlike `UnTar`)

### Security

| Vulnerability | Status |
|---|---|
| Zip slip (path traversal in tar headers) | Mitigated via `validTarPath` + `HasPrefix` |
| Zip slip via symlinks in destination dir | **Not mitigated** |
| Decompression bomb (per-file) | Partially mitigated — silently truncates, no error |
| Decompression bomb (archive total) | Mitigated — returns error |
| `UnGz` path traversal check | Broken for relative paths |
| Setuid/setgid bits | Mitigated by `> 0o777` fallback |
| Symlink entries in tar | Safe — silently skipped |
| Memory exhaustion from large base64 | Not mitigated |

### Recommendations

1. Fix decompression bomb detection — return error on truncation
2. Fix `UnGz` path traversal check
3. Fix `Tar` path stripping — use `filepath.Rel`
4. Define sentinel error types (`ErrPathTraversal`, `ErrSizeLimitExceeded`)
5. Add symlink-safe path resolution with `filepath.EvalSymlinks`
6. Stream `UnTarGzBase64` instead of buffering
7. Add `UnGz` size options matching `UnTar` pattern
