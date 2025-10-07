# Release Guide

This guide explains how to ensure PR details appear in release notes when using semantic-release with a protected main branch.

## Problem

When merging PRs, only the PR title appears in releases:
- ❌ Release shows: "add Argo Workflows client library"
- ✅ Want: Full PR description with features, testing details, examples

## Root Cause

`semantic-release` generates release notes from **commit messages**, not PR data. When you squash-merge, by default only the PR title becomes the commit message.

## Solution: Configure Squash Merge to Include PR Body

### Option 1: Repository Settings (Recommended)

1. Go to: `https://github.com/jasoet/pkg/settings`
2. Scroll to **"Pull Requests"** section
3. Under **"Allow squash merging"**, click **"Configure"**
4. Select: **"Default to pull request title and description"**
5. Save changes

This automatically includes PR body in squash commits.

### Option 2: Manual Edit When Merging

When clicking "Squash and merge":
1. GitHub shows the commit message editor
2. Copy your PR body content
3. Paste it below the title in the commit message
4. Complete the merge

**Example commit message:**
```
feat(argo): add Argo Workflows client library (#8)

## Summary

Adds a production-ready Argo Workflows client library to the pkg repository.

## Features

- **Multiple Connection Modes**: Kubernetes API, in-cluster, and Argo Server HTTP
- **Flexible Configuration**: Config structs and functional options pattern
- **OpenTelemetry Integration**: Built-in tracing and observability support
- **Production-Ready**: Proper error handling without fatal errors

## Testing

All unit tests pass:
\`\`\`bash
$ go test ./argo
PASS
ok      github.com/jasoet/pkg/v2/argo   1.174s
\`\`\`
```

### Option 3: Use gh CLI with --body

```bash
# Merge PR with body included
gh pr merge 123 --squash --body
```

This prompts you to edit the commit message including the PR body.

## How semantic-release Uses This

1. **Commit analyzer** reads the commit messages (now with PR body)
2. **Release notes generator** creates sections based on commit type
3. **GitHub plugin** publishes the detailed release notes

## Verification

After merging with PR body included:

1. Check commit log: `git log --format=fuller`
2. Verify commit body contains PR details
3. Wait for release workflow to complete
4. Check release notes at `https://github.com/jasoet/pkg/releases`

## Best Practices

### For PR Authors
- Write detailed PR descriptions with:
  - Summary section
  - Feature lists
  - Testing details
  - Migration examples
  - Breaking changes

### For Reviewers/Mergers
- Always include PR body when squash-merging
- Use conventional commit format in PR title
- Review the commit message before confirming merge

## Example PR Template

```markdown
## Summary
Brief description of changes and motivation

## Changes
- ✅ Added feature X
- ✅ Updated component Y
- ✅ Fixed issue Z

## Testing
How changes were tested

## Breaking Changes
List any breaking changes (or "None")

## Migration Guide
How users should update their code (if needed)
```

## Why Not Use @semantic-release/changelog?

The `@semantic-release/changelog` plugin writes CHANGELOG.md and commits it back to the repo. This **fails with protected branches** because:
- Protected branches require PRs for all commits
- semantic-release runs after merge, can't create another PR
- Workflow fails: `refusing to allow a bot to create or update workflow`

Instead, GitHub releases serve as your changelog - they're automatically created with full details when commits include PR bodies.

## Troubleshooting

### Release notes still missing details

**Check:** Did the squash commit include PR body?
```bash
git show HEAD --format=fuller
```

**Solution:** Verify repo settings or manually edit commit messages when merging.

### semantic-release fails with "protected branch"

**Check:** Do you have `@semantic-release/changelog` or `@semantic-release/git` in `.releaserc.json`?

**Solution:** Remove these plugins - they try to commit to the repo, which fails with protected branches.

## Current Configuration

Your `.releaserc.json` is correctly configured:
- ✅ Analyzes commits with conventional commits
- ✅ Generates detailed release notes
- ✅ Publishes to GitHub releases
- ✅ No plugins that commit back to repo

The only requirement is **including PR bodies in squash commits**.
