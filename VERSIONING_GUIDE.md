# Versioning Guide

This document describes the versioning strategy and release workflow for the `pkg` library.

## Overview

This project uses **Semantic Versioning** (SemVer) with a hybrid release approach:
- **v2 Beta Period** (current): Manual releases for deliberate milestone control
- **v2 GA and beyond**: Automated releases using semantic-release

## Current Status: v2 Beta Period

During the v2 beta phase, we use **manual releases** to maintain control over:
- Release timing and messaging
- Breaking changes communication
- Feature completeness milestones

### Creating Manual Beta Releases

```bash
# 1. Ensure all changes are committed and pushed
git checkout main
git pull

# 2. Create annotated tag
git tag -a v2.0.0-beta.X -m "Release notes here"

# 3. Push tag
git push origin v2.0.0-beta.X

# 4. Create GitHub release
gh release create v2.0.0-beta.X --prerelease --title "v2.0.0-beta.X" --notes "Release notes"
```

## Future: Automated Semantic Release Workflow

After v2.0.0 GA, this project will use **semantic-release** for automated versioning and releases.

### Workflow Overview

1. **Branch-based development** with free commit messages
2. **PR titles follow Conventional Commits** format
3. **Squash merge** PRs to main (PR title becomes commit message)
4. **Automated releases** triggered by semantic-release

### Conventional Commits Format

PR titles must follow this format:

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Types:**
- `feat`: New feature (minor version bump)
- `fix`: Bug fix (patch version bump)
- `docs`: Documentation only
- `refactor`: Code refactoring (patch version bump)
- `test`: Adding/updating tests (patch version bump)
- `chore`: Maintenance tasks (patch version bump)
- `perf`: Performance improvements (patch version bump)
- `ci`: CI/CD changes (patch version bump)

**Breaking Changes:**
Add `!` after type or `BREAKING CHANGE:` in footer for major version bump:
```
feat!: redesign API interface

BREAKING CHANGE: The API has been redesigned
```

### Branch Protection Rules

Configure these settings in GitHub repository settings:

1. **Require pull request before merging**
   - Require approvals: 1
   - Dismiss stale reviews when new commits are pushed

2. **Require status checks to pass**
   - Require branches to be up to date
   - Status checks: `test`, `lint`, `build`

3. **Do not allow bypassing the above settings**

4. **Squash merging only**
   - Disable merge commits
   - Disable rebase merging
   - Enable squash merging only

### Semantic Release Configuration

The project uses `.releaserc.json` for semantic-release configuration:

```json
{
  "branches": ["main", "master"],
  "plugins": [
    ["@semantic-release/commit-analyzer", {
      "preset": "conventionalcommits",
      "releaseRules": [
        {"type": "docs", "scope": "README", "release": "patch"},
        {"type": "refactor", "release": "patch"},
        {"type": "chore", "release": "patch"},
        {"type": "test", "release": "patch"}
      ]
    }],
    "@semantic-release/release-notes-generator",
    ["@semantic-release/changelog", {"changelogFile": "CHANGELOG.md"}],
    "@semantic-release/github",
    ["@semantic-release/git", {
      "assets": ["CHANGELOG.md", "go.mod"],
      "message": "chore(release): ${nextRelease.version} [skip ci]\n\n${nextRelease.notes}"
    }]
  ]
}
```

### Release Process (Automated)

Once semantic-release is re-enabled:

1. **Merge PR to main** with semantic commit message (from PR title)
2. **CI automatically runs** semantic-release
3. **Version is determined** from commit messages since last release
4. **CHANGELOG.md is updated** automatically
5. **Git tag is created** and pushed
6. **GitHub release is created** with release notes

### Skip CI for Non-Release Commits

Add `[skip ci]` to commit messages for documentation-only changes:
```
docs: update README [skip ci]
```

### Release Workflow File

After v2 GA, remove the v2 check in `.github/workflows/release.yml`:

```yaml
# Remove this check:
- name: Check for v2 tags
  id: check_v2
  run: |
    if git tag -l "v2.*" | grep -q .; then
      echo "v2_exists=true" >> $GITHUB_OUTPUT
    fi

# And this condition:
- name: Release
  if: steps.check_v2.outputs.v2_exists != 'true'  # Remove this
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
  run: npx semantic-release
```

## Migration from v1 to v2

### Breaking Changes in v2

1. **OpenTelemetry Integration Added**
   - **v1 has NO OpenTelemetry support**
   - **v2 adds optional OpenTelemetry instrumentation** across all packages
   - Minimal API changes - most packages accept optional `OTelConfig` parameter
   - Automatic tracing, metrics, and logging integration when OTel is configured
   - Users can ignore OTel features and use packages without observability

2. **Database Package**
   - New `ConnectionConfig.OTelConfig` field (optional)
   - When provided, enables automatic query tracing and connection pool metrics
   - Backward compatible - works without OTel config

3. **Server/gRPC/REST Packages**
   - New `OTelConfig` field in configuration structs (optional)
   - When provided, enables automatic request tracing and metrics
   - Backward compatible - works without OTel config

4. **Logging Package**
   - New `NewLoggerProvider` function for OTel integration
   - Legacy `Initialize` and `ContextLogger` functions still work

### Migration Steps

1. **Update Dependencies**
   ```bash
   go get github.com/jasoet/pkg/v2@v2.0.0
   ```

2. **Update Import Paths**
   ```go
   // Old (v1)
   import "github.com/jasoet/pkg/db"

   // New (v2)
   import "github.com/jasoet/pkg/v2/db"
   ```

3. **(Optional) Add OpenTelemetry**
   - v2 supports OpenTelemetry instrumentation (optional)
   - To enable: create `otel.Config` and pass to package configs
   - Without OTel: packages work exactly like v1 (no observability)
   - With OTel: automatic tracing, metrics, and logging

4. **Test Thoroughly**
   - Run your test suite
   - Verify metrics collection
   - Verify distributed tracing
   - Check error handling

### v1 Maintenance

v1.x will receive **critical bug fixes only** for 6 months after v2.0.0 GA:
- Security patches
- Critical bug fixes
- No new features
- No dependency updates (except security)

After 6 months, v1.x enters maintenance mode (security patches only).

## Version Support Policy

- **Latest major version** (v2): Full support
- **Previous major version** (v1): Critical fixes for 6 months post-GA
- **Older versions**: No support

## Examples

### Example: Adding a New Feature

**Development:**
```bash
git checkout -b feature/add-redis-support
# Make changes with any commit messages
git commit -m "wip redis"
git commit -m "add tests"
git push origin feature/add-redis-support
```

**PR:**
- Title: `feat(cache): add Redis cache implementation`
- Body: Detailed description
- Merge: Squash merge to main

**Result:** Minor version bump (e.g., 2.0.0 → 2.1.0)

### Example: Bug Fix

**PR:**
- Title: `fix(db): handle connection timeout correctly`
- Merge: Squash merge to main

**Result:** Patch version bump (e.g., 2.1.0 → 2.1.1)

### Example: Breaking Change

**PR:**
- Title: `feat(http)!: redesign middleware interface`
- Body:
  ```
  BREAKING CHANGE: Middleware function signature changed from
  func(http.Handler) http.Handler to func(next http.HandlerFunc) http.HandlerFunc
  ```
- Merge: Squash merge to main

**Result:** Major version bump (e.g., 2.1.1 → 3.0.0)

## FAQ

### Q: Why manual releases for v2 beta?

Beta releases are deliberate milestones that benefit from manual control over timing and messaging. We want to carefully communicate breaking changes and feature completeness to early adopters.

### Q: When will semantic-release be re-enabled?

After v2.0.0 GA is released. At that point, we'll remove the v2 check in `.github/workflows/release.yml`.

### Q: Can I still use conventional commits during v2 beta?

Yes! We encourage it for consistency. It will make the transition to automated releases smoother.

### Q: What if I need to release a v1 patch?

Checkout the v1 branch, cherry-pick the fix, and create a tag manually. Semantic-release is disabled while v2 tags exist.

### Q: How do I know what version will be released?

Run `npx semantic-release --dry-run` locally to see what version would be released based on commits since the last tag.

## References

- [Semantic Versioning](https://semver.org/)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [Semantic Release](https://semantic-release.gitbook.io/)
- [OpenTelemetry Go Documentation](https://opentelemetry.io/docs/languages/go/)
