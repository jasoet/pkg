# v3 Phase 2: Module Path Bump to /v3

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Change the Go module path from `github.com/jasoet/pkg/v2` to `github.com/jasoet/pkg/v3` on the `next` branch, before any package-refactor phases begin, so later phases never write `/v2` imports that would need a second rewrite.

**Architecture:** Purely mechanical: module line in go.mod, all `/v2` import strings in non-vendor `.go` files, vendor re-sync, release-notes template fix. Package READMEs keep stale `/v2` import snippets intentionally — each package's own phase rewrites its docs.

**Tech Stack:** Go 1.26 modules, vendored deps, semantic-release.

## Global Constraints

- Work on `next`. Conventional Commits; NEVER add AI attribution. Commit with `!` and a `BREAKING CHANGE:` footer — this is the first intentionally breaking v3 commit.
- Run commands via `task <name>` where one exists.
- Do NOT touch package README import examples in this phase (per-package phases own docs).
- Do NOT tag manually.

---

### Task 1: Rewrite module path and imports

**Files:**
- Modify: `go.mod` (module line)
- Modify: all non-vendor `*.go` files containing `github.com/jasoet/pkg/v2`
- Regenerate: `vendor/`, `go.sum`

**Interfaces:**
- Produces: building `/v3` module; archtest and all tests compile against `/v3` imports.

- [ ] **Step 1: Record the failing state**

Run: `grep -rl 'github.com/jasoet/pkg/v2' --include='*.go' . | grep -v '^./vendor' | wc -l`
Expected: a large count (the files to rewrite).

- [ ] **Step 2: Rewrite go.mod and all import strings**

```bash
sed -i '' 's|^module github.com/jasoet/pkg/v2$|module github.com/jasoet/pkg/v3|' go.mod
grep -rl 'github.com/jasoet/pkg/v2' --include='*.go' . | grep -v '^./vendor' | xargs sed -i '' 's|github.com/jasoet/pkg/v2|github.com/jasoet/pkg/v3|g'
```

- [ ] **Step 3: Re-sync modules and vendor**

Run: `task vendor`
Expected: `go mod tidy` + `go mod vendor` complete without errors.

- [ ] **Step 4: Verify build and tests**

```bash
nix develop -c go build ./...
nix develop -c go test ./internal/archtest/ -count=1
task test
```
Expected: build clean; archtest green; unit suite green (coverage totals may shift slightly — fine).

- [ ] **Step 5: Verify no /v2 imports remain in code**

Run: `grep -rl 'github.com/jasoet/pkg/v2' --include='*.go' . | grep -v '^./vendor' | wc -l`
Expected: `0`

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum vendor $(git ls-files -m '*.go')
git commit -m "feat!: change module path to github.com/jasoet/pkg/v3

BREAKING CHANGE: module path is now github.com/jasoet/pkg/v3; consumers must update imports."
```

---

### Task 2: Release-notes template and docs pointers

**Files:**
- Modify: `.releaserc.json` (headerPartial `go get` line)
- Modify: `INSTRUCTION.md` (module path + v3 note)

**Interfaces:**
- Produces: correct `go get github.com/jasoet/pkg/v3@...` in future release notes; agent docs pointing at /v3.

- [ ] **Step 1: Fix the headerPartial go-get line**

In `.releaserc.json`, change `go get github.com/jasoet/pkg/v2@{{currentTag}}` to `go get github.com/jasoet/pkg/v3@{{currentTag}}`.

- [ ] **Step 2: Update INSTRUCTION.md**

Change `**Module Path:** `github.com/jasoet/pkg/v2`` to `**Module Path:** `github.com/jasoet/pkg/v3`` (on `next`; `release/v2` keeps `/v2`).

- [ ] **Step 3: Verify**

```bash
bunx js-yaml .releaserc.json > /dev/null && echo JSON-OK
grep -n 'pkg/v3' INSTRUCTION.md .releaserc.json | head -5
```
Expected: JSON-OK; both files reference /v3.

- [ ] **Step 4: Commit**

```bash
git add .releaserc.json INSTRUCTION.md
git commit -m "ci(release): point release notes and INSTRUCTION.md at /v3 module path"
```
