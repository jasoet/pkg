# v3 Phase 1: Foundation — Branch Mechanics, Process Teeth, Archtest

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Set up v3 development infrastructure — `release/v2` maintenance branch, `next` prerelease branch, semantic-release branch roles, integration-test release gate, gorelease API-diff CI gate, and the `internal/archtest` convention-enforcement test package — so all later v3 work lands on a ratchet that makes convention drift a red test.

**Architecture:** semantic-release three-branch model (`main` = release line, `next` = v3 dev with prereleases, `release/v2` = emergency patches). All v3 development happens on `next`. CI gates run on the self-hosted macOS runner via nix.

**Tech Stack:** Go 1.26, semantic-release (bunx), GitHub Actions (self-hosted), golang.org/x/exp/cmd/gorelease, testify.

## Global Constraints

- Work happens on the `next` branch after Task 2. Only Tasks 1–2 run from `main`.
- Conventional Commits: `<type>(<scope>): <description>`. NEVER add AI attribution anywhere.
- Run commands via `task <name>` where one exists; raw `nix develop -c`/`go` is acceptable where no task covers it.
- `docs/plans/2026-07-22-v3-audit-backlog.md` is the backlog of record — do not duplicate it.
- Do NOT tag anything manually. semantic-release owns all tags.

---

### Task 1: Create `release/v2` maintenance branch

**Files:** none (git only)

**Interfaces:**
- Produces: remote branch `release/v2` at tag `v2.13.1`, used by Task 3's `.releaserc.json`.

- [ ] **Step 1: Create and push the branch from the v2.13.1 tag**

```bash
git fetch --tags origin
git branch release/v2 v2.13.1
git push -u origin release/v2
```

- [ ] **Step 2: Verify**

Run: `git ls-remote --heads origin release/v2`
Expected: one line, SHA equals `git rev-parse v2.13.1^{commit}`.

---

### Task 2: Create `next` branch and switch to it

**Files:** none (git only)

**Interfaces:**
- Consumes: `main` at `dd01284` (or later).
- Produces: remote branch `next`; all subsequent tasks commit here.

- [ ] **Step 1: Create, switch, push**

```bash
git checkout main && git pull
git checkout -b next
git push -u origin next
```

- [ ] **Step 2: Verify**

Run: `git branch --show-current && git ls-remote --heads origin next`
Expected: `next`, and one remote line.

---

### Task 3: semantic-release branch roles

**Files:**
- Modify: `.releaserc.json` (only the `"branches"` key — keep all plugin config identical)

**Interfaces:**
- Consumes: `release/v2` and `next` remote branches (Tasks 1–2).
- Produces: releases from `main` = normal, from `next` = `X.Y.Z-next.N` prereleases, from `release/v2` = patches in the `2.13.x` range.

- [ ] **Step 1: Write the failing verification (dry-run on current config)**

Run: `nix develop -c bunx semantic-release --dry-run --no-ci 2>&1 | head -20`
Expected: config loads, but there is no `next` channel — proving the change is needed.

- [ ] **Step 2: Replace the branches array in `.releaserc.json`**

Change:
```json
  "branches": [
    "main"
  ],
```
to:
```json
  "branches": [
    "main",
    {
      "name": "next",
      "prerelease": true
    },
    {
      "name": "release/v2",
      "range": "2.13.x"
    }
  ],
```

- [ ] **Step 3: Verify JSON validity and channel recognition**

```bash
bunx js-yaml .releaserc.json > /dev/null && echo JSON-OK
nix develop -c bunx semantic-release --dry-run --no-ci 2>&1 | head -30
```
Expected: JSON-OK; dry-run output lists `next` as a configured branch (note: it may report "no release" — that is fine, we only verify the config loads and the branch is recognized).

- [ ] **Step 4: Extend release workflow triggers**

In `.github/workflows/release.yml`, change:
```yaml
on:
  push:
    branches: [main]
```
to:
```yaml
on:
  push:
    branches: [main, next, 'release/v2']
```

- [ ] **Step 5: Commit**

```bash
git add .releaserc.json .github/workflows/release.yml
git commit -m "ci(release): add next prerelease and release/v2 maintenance branch roles"
```

---

### Task 4: Integration-test gate in the Release workflow

**Files:**
- Modify: `.github/workflows/release.yml` (the `test` job)

**Interfaces:**
- Produces: every release (main/next/release-v2) gates on unit AND integration tests.

- [ ] **Step 1: Add the integration step to the `test` job**

In `.github/workflows/release.yml`, after the existing `Test` step, add:
```yaml
      - name: Integration tests
        run: nix develop --command bash -c "go list ./... | grep -v examples | xargs go test -count=1 -tags=integration -timeout=20m"
```

(Keep the existing unit step unchanged. No `-race` on the integration step — it doubles runtime against containers.)

- [ ] **Step 2: Verify locally (same command as CI)**

Run: `nix develop -c bash -c "go list ./... | grep -v examples | xargs go test -count=1 -tags=integration -timeout=20m"`
Expected: all packages `ok` (db, ssh, temporal, docker included). This is the exact gate command; it must be green before commit.

- [ ] **Step 3: Validate workflow YAML**

Run: `bunx js-yaml .github/workflows/release.yml > /dev/null && echo YAML-OK`
Expected: YAML-OK

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "ci(release): gate releases on integration tests"
```

---

### Task 5: gorelease API-diff gate in CI

**Files:**
- Modify: `.github/workflows/ci.yml`

**Interfaces:**
- Produces: blocking API-compat check on `main`, `release/v2`, and PRs; informational on `next` (breaking changes are intended there).

- [ ] **Step 1: Full history for tag comparison**

In `.github/workflows/ci.yml`, the checkout step becomes:
```yaml
      - name: Checkout
        uses: actions/checkout@v6
        with:
          fetch-depth: 0
```

- [ ] **Step 2: Add the gorelease step after the `Test` step**

```yaml
      - name: API compatibility check
        # Breaking API changes are intended on `next` (v3 line) — informational there.
        if: github.ref_name != 'next'
        run: nix develop --command go run golang.org/x/exp/cmd/gorelease@latest

      - name: API compatibility report (informational on next)
        if: github.ref_name == 'next'
        continue-on-error: true
        run: nix develop --command go run golang.org/x/exp/cmd/gorelease@latest
```

- [ ] **Step 3: Verify locally**

Run: `nix develop -c go run golang.org/x/exp/cmd/gorelease@latest`
Expected: exits 0 on a clean tree (no API change vs v2.13.1). First run downloads the tool — slow is fine.

- [ ] **Step 4: Validate workflow YAML and commit**

```bash
bunx js-yaml .github/workflows/ci.yml > /dev/null && echo YAML-OK
git add .github/workflows/ci.yml
git commit -m "ci: add gorelease API-compatibility gate (blocking on main/release-v2, informational on next)"
```

---

### Task 6: `internal/archtest` convention tests (TDD)

**Files:**
- Create: `internal/archtest/archtest_test.go`
- Create: `internal/archtest/doc.go`

**Interfaces:**
- Produces: `go test ./internal/archtest/` — the ratchet later phases extend. Registry map `compliantConfigs` (name → reflect.Type) and compile-time `WithOTelConfig` assignments are the extension points.

- [ ] **Step 1: Write the failing test**

Create `internal/archtest/archtest_test.go`:
```go
package archtest

import (
	"reflect"
	"testing"

	"github.com/jasoet/pkg/v2/db"
	"github.com/jasoet/pkg/v2/otel"
	"github.com/jasoet/pkg/v2/rest"
	"github.com/jasoet/pkg/v2/server"
	"github.com/jasoet/pkg/v2/temporal"
)

// compliantConfigs registers exported config structs that must carry an
// OTelConfig *otel.Config field tagged `yaml:"-" mapstructure:"-"`.
// Add a package here when it is unified onto the v3 conventions.
var compliantConfigs = map[string]reflect.Type{
	"db":       reflect.TypeOf(db.ConnectionConfig{}),
	"rest":     reflect.TypeOf(rest.Config{}),
	"server":   reflect.TypeOf(server.Config{}),
	"temporal": reflect.TypeOf(temporal.Config{}),
}

func TestConfigStructsCarryOTelConfig(t *testing.T) {
	otelPtrType := reflect.TypeOf(&otel.Config{})

	for pkg, typ := range compliantConfigs {
		t.Run(pkg, func(t *testing.T) {
			field, ok := typ.FieldByName("OTelConfig")
			if !ok {
				t.Fatalf("%s: missing OTelConfig field", pkg)
			}
			if field.Type != otelPtrType {
				t.Errorf("%s: OTelConfig is %s, want *otel.Config", pkg, field.Type)
			}
			if got := field.Tag.Get("yaml"); got != "-" {
				t.Errorf("%s: OTelConfig yaml tag = %q, want %q", pkg, got, "-")
			}
			if got := field.Tag.Get("mapstructure"); got != "-" {
				t.Errorf("%s: OTelConfig mapstructure tag = %q, want %q", pkg, got, "-")
			}
		})
	}
}
```

Create `internal/archtest/doc.go`:
```go
// Package archtest mechanically enforces the library's v3 conventions.
// Tests here fail when a package's config struct loses its OTelConfig
// contract or a package drops its WithOTelConfig option. Extend the
// registries as packages are unified onto the conventions.
package archtest
```

Create `internal/archtest/options_test.go`:
```go
package archtest

import (
	"github.com/jasoet/pkg/v2/docker"
	"github.com/jasoet/pkg/v2/grpc"
	"github.com/jasoet/pkg/v2/rest"
	"github.com/jasoet/pkg/v2/server"
)

// Compile-time contract: each compliant package exposes WithOTelConfig.
// Add a package here when it is unified onto the v3 conventions.
var (
	_ = docker.WithOTelConfig
	_ = grpc.WithOTelConfig
	_ = rest.WithOTelConfig
	_ = server.WithOTelConfig
)
```

- [ ] **Step 2: Run to verify failure mode (sanity)**

Run: `nix develop -c go test ./internal/archtest/ -v 2>&1 | head -20`
Expected: PASS for all five registered structs (they are today's compliant set). To prove the test has teeth, also run:
`nix develop -c go test ./internal/archtest/ -run TestConfigStructsCarryOTelConfig/db -v`
Expected: `ok` — db is compliant. (If any registered package fails, that package's tags regressed since the audit — fix the registry, not the test.)

- [ ] **Step 3: Negative check — test actually detects violations (do not commit this)**

Temporarily add `"argo": reflect.TypeOf(argo.Config{}),` to the registry plus the argo import, run:
`nix develop -c go test ./internal/archtest/ -run TestConfigStructsCarryOTelConfig/argo -v`
Expected: FAIL — argo.Config lacks `mapstructure:"-"` (audit finding). Then revert the temporary lines.

- [ ] **Step 4: Commit**

```bash
git add internal/archtest/
git commit -m "test(archtest): add convention-enforcement test package (OTelConfig tags, WithOTelConfig presence)"
```

---

### Task 7: Update INSTRUCTION.md for the v3 branch model

**Files:**
- Modify: `INSTRUCTION.md` (Conventions section + Project Overview)

**Interfaces:**
- Produces: agent-facing docs matching the new branch reality.

- [ ] **Step 1: Add branch-model note**

In `INSTRUCTION.md`, in the `## Project Overview` section after the `**v1 Branch:**` line, add:
```markdown
**v3 Development:** v2 is frozen at v2.13.1 (`release/v2` branch, emergency patches only). v3 work happens on the `next` branch (prereleases `v3.0.0-next.N`). Backlog: `docs/plans/2026-07-22-v3-audit-backlog.md`.
```

- [ ] **Step 2: Add archtest to the Key Paths table**

After the `docs/plans/` row, add:
```markdown
| `internal/archtest/` | Convention-enforcement tests — extend registry when unifying a package |
```

- [ ] **Step 3: Commit**

```bash
git add INSTRUCTION.md
git commit -m "docs(instruction): document v3 branch model and archtest"
```

---

### Task 8: Final verification and push

- [ ] **Step 1: Full local gate**

Run: `task check` (unit tests + lint)
Expected: green.

- [ ] **Step 2: Push next**

```bash
git push origin next
```

- [ ] **Step 3: Verify the Release workflow on next**

Run: `gh run list --branch next --limit 3`
Expected: a Release run triggered by the push; it may fail at the semantic-release step if there is no release to cut (`feat`/`fix` commits on next WILL cut `2.14.0-next.1` or similar — either outcome is acceptable as long as the test jobs are green). Watch with `gh run watch` if needed.
