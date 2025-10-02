# Maintaining Guide

This document explains how to maintain and release patches for both v1 and v2 versions of this library.

## Branch Strategy

### `main` - v2 Development (Current)
- **Module Path:** `github.com/jasoet/pkg/v2`
- **Latest Version:** v2.0.0-beta.1
- **Purpose:** Active development for v2.x releases
- **Go Version:** 1.25.1+

### `release/v1` - v1 Maintenance
- **Module Path:** `github.com/jasoet/pkg` (no /v2 suffix)
- **Latest Version:** v1.6.0
- **Purpose:** Bug fixes and security patches for v1.x
- **Go Version:** 1.25.1+

## Releasing v1 Patches

When you need to release a bug fix or security patch for v1.x users:

### 1. Switch to v1 branch
```bash
git checkout release/v1
```

### 2. Apply fixes

**Option A: Cherry-pick from main**
```bash
# If the fix was already made on main
git cherry-pick <commit-hash>
```

**Option B: Direct fix**
```bash
# Make changes directly on release/v1
# Edit files, test, commit
git add .
git commit -m "fix: description of the fix"
```

### 3. Test thoroughly
```bash
# Run all tests
task test:all

# Or run individually
task test
task test:integration
task test:temporal
```

### 4. Tag and release
```bash
# Tag with appropriate version (e.g., v1.6.1, v1.6.2)
git tag v1.6.1
git push origin release/v1
git push origin v1.6.1
```

### 5. Return to main
```bash
git checkout main
```

## Releasing v2 Versions

For v2 development on `main` branch:

```bash
# Ensure you're on main
git checkout main

# Tag with v2.x.x version
git tag v2.0.0
git push origin main
git push origin v2.0.0
```

## Version Guidelines

### v1.x.x (Maintenance Only)
- **Patch releases** (v1.6.1, v1.6.2): Bug fixes, security patches
- **NO new features** - v1 is in maintenance mode
- **NO breaking changes** - maintain backward compatibility

### v2.x.x (Active Development)
- **Major releases** (v2.0.0, v3.0.0): Breaking changes allowed
- **Minor releases** (v2.1.0, v2.2.0): New features, backward compatible
- **Patch releases** (v2.0.1, v2.0.2): Bug fixes, security patches

## Import Paths for Users

### Using v1 (Maintenance Branch)
```go
import "github.com/jasoet/pkg/compress"
import "github.com/jasoet/pkg/server"
```

```bash
go get github.com/jasoet/pkg@v1.6.1
```

### Using v2 (Current)
```go
import "github.com/jasoet/pkg/v2/compress"
import "github.com/jasoet/pkg/v2/server"
```

```bash
go get github.com/jasoet/pkg/v2@v2.0.0
```

## Testing Strategy

Before any release (v1 or v2):

1. **Unit Tests:** `task test`
2. **Integration Tests:** `task test:integration`
3. **Temporal Tests:** `task test:temporal`
4. **Linting:** `task lint`

Or run everything:
```bash
task test:all
```

## Common Scenarios

### Security Patch for v1 Users

1. Identify the vulnerability
2. Fix on `release/v1` branch
3. Test thoroughly
4. Release as v1.6.x (patch version)
5. Optionally apply to main if relevant to v2

### Bug Fix Needed for Both v1 and v2

1. Fix on `main` first (for v2)
2. Cherry-pick to `release/v1`
3. Test on both branches
4. Release both versions:
   - v1.6.x (patch)
   - v2.0.x or v2.x.0 (depending on scope)

### Migration from v1 to v2

Users upgrading from v1 to v2 need to:
1. Update import paths: add `/v2` suffix
2. Update `go.mod`: `go get github.com/jasoet/pkg/v2@latest`
3. Review CHANGELOG for breaking changes

## Notes

- **Both versions can coexist** - Projects can use v1 and v2 simultaneously if needed
- **v1 lifecycle** - Will be maintained for critical security fixes, but new features go to v2
- **Semantic versioning** - Strictly followed for both v1 and v2
- **Go compatibility** - Both versions currently support Go 1.25.1+

## Questions?

If you have questions about releasing patches or version management, refer to:
- [Go Modules Version Management](https://go.dev/doc/modules/version-numbers)
- [Semantic Versioning 2.0](https://semver.org/)
