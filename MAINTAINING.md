# Maintaining Guide

This document explains how to maintain and release this library.

## Branch Strategy

### `main` - Active Development
- **Module Path:** `github.com/jasoet/pkg/v2`
- **Purpose:** Active development for v2.x releases
- **Go Version:** 1.26+

## Releasing

Releases are fully automated via [semantic-release](https://github.com/semantic-release/semantic-release) on every push to `main`.

### What triggers a release

| Commit Type | Release | Example |
|---|---|---|
| `feat` | Minor (v2.x.0) | `feat(server): add gRPC interceptor` |
| `fix` | Patch (v2.0.x) | `fix(compress): handle empty input` |
| `perf` | Patch | `perf(db): reduce query allocations` |
| `refactor` | Patch | `refactor(otel): simplify provider setup` |
| Breaking change | Major (vX.0.0) | `feat!: remove deprecated API` or footer `BREAKING CHANGE:` |

### What does NOT trigger a release

`docs`, `test`, `ci`, `chore`, `style`, `build` commits are excluded.

### Workflow

1. Merge PR to `main` with conventional commit title
2. CI runs tests
3. semantic-release analyzes commits since last tag
4. If a release is warranted, it creates a GitHub release with notes
5. Go module proxy is warmed automatically

## CI Pipelines

- **`ci.yml`** - Runs on PRs: test (with race detector) + lint
- **`release.yml`** - Runs on push to `main`: test + semantic-release

## Conventional Commits

All PR titles must follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

### Best Practices for PR Authors
- Write detailed PR descriptions (they become release notes when squash-merged)
- Use conventional commit format in PR title
- Include scope when the change targets a specific package

## Import Paths

```go
import "github.com/jasoet/pkg/v2/compress"
import "github.com/jasoet/pkg/v2/server"
```

```bash
go get github.com/jasoet/pkg/v2@latest
```

## Testing

Before any release:

1. **Unit Tests:** `task test`
2. **Integration Tests:** `task test:integration`
3. **Linting:** `task lint`

Or run everything:
```bash
task test:complete  # Runs all tests with coverage
```
