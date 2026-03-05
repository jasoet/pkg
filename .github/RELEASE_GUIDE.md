# Release Guide

This guide explains how to ensure PR details appear in release notes when using semantic-release.

## How It Works

1. PRs are squash-merged to `main` with conventional commit titles
2. `semantic-release` analyzes commit messages and determines version bump
3. GitHub release is created with categorized notes
4. Go module proxy is warmed for immediate availability

## Getting Good Release Notes

### Configure Squash Merge (Repository Setting)

1. Go to: `https://github.com/jasoet/pkg/settings`
2. Scroll to **"Pull Requests"** section
3. Under **"Allow squash merging"**, click **"Configure"**
4. Select: **"Default to pull request title and description"**
5. Save changes

This automatically includes PR body in squash commits, which becomes the release note content.

### What Triggers a Release

| Commit Type | Release | Hidden from Notes |
|---|---|---|
| `feat` | Minor | No |
| `fix` | Patch | No |
| `perf` | Patch | No |
| `refactor` | Patch | No |
| `docs`, `test`, `ci`, `chore`, `style`, `build` | None | Yes |
| `BREAKING CHANGE` footer or `!` suffix | Major | No |

### PR Title Format

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(argo): add Argo Workflows client library
fix(compress): handle empty input gracefully
perf(db): reduce query allocations
refactor(otel): simplify provider setup
feat!: remove deprecated Config struct
```

### PR Body Best Practices

Write detailed PR descriptions — they become the commit body on squash merge:

```markdown
## Summary
Brief description of changes and motivation

## Changes
- Added feature X
- Updated component Y
- Fixed issue Z

## Testing
How changes were tested

## Breaking Changes
List any breaking changes (or "None")
```

## Configuration

- **`.releaserc.json`** - semantic-release config (commit analyzer, release notes, GitHub plugin)
- **`.github/workflows/release.yml`** - Release pipeline (test + semantic-release + Go proxy warm-up)
- **`.github/workflows/ci.yml`** - PR pipeline (test + lint)

## Troubleshooting

### Release notes missing details

**Check:** Did the squash commit include the PR body?
```bash
git show HEAD --format=fuller
```
**Fix:** Verify repository squash merge settings (see above).

### Unwanted releases from non-code changes

Only `feat`, `fix`, `perf`, and `refactor` commits trigger releases. Use `chore`, `ci`, `docs`, `test`, or `style` types for non-library changes.

### Go proxy not updated

The release workflow warms the proxy automatically. If it still shows stale data:
```bash
GOPROXY=https://proxy.golang.org go list -m github.com/jasoet/pkg/v2@v2.x.x
```
