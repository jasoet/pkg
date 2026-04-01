# Nix Integration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add Nix flake-based reproducible dev environment where all Taskfile commands execute tools through `nix develop -c`.

**Architecture:** `flake.nix` declares all dev tools (Go, linters, formatters, bun). Taskfile uses a `{{.N}}` variable (`nix develop -c`) to prefix every tool invocation. Non-tool shell commands (`mkdir`, `echo`, `rm`) run directly.

**Tech Stack:** Nix flakes, go-task, direnv (optional)

---

### Task 1: Create Feature Branch

**Step 1: Create and switch to feature branch**

Run: `git checkout -b feat/nix-integration`

**Step 2: Verify branch**

Run: `git branch --show-current`
Expected: `feat/nix-integration`

---

### Task 2: Create `flake.nix`

**Files:**
- Create: `flake.nix`

**Step 1: Create the flake**

```nix
{
  description = "Go pkg/v2 development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in {
        devShells.default = pkgs.mkShell {
          packages = [
            # Go toolchain
            pkgs.go
            pkgs.golangci-lint
            pkgs.gofumpt
            pkgs.gosec

            # Container CLIs (daemon is system-level)
            pkgs.podman
            pkgs.docker-client
            pkgs.podman-compose

            # Other tools
            pkgs.bun
            pkgs.jq
          ];

          shellHook = ''
            echo "pkg/v2 dev environment ready — Go $(go version | awk '{print $3}')"
          '';
        };
      });
}
```

**Step 2: Generate `flake.lock`**

Run: `nix flake lock`
Expected: `flake.lock` created in project root.

**Step 3: Verify the devShell builds**

Run: `nix develop -c go version`
Expected: Go version output (e.g., `go version go1.26.0 linux/amd64`)

**Step 4: Verify all tools are accessible**

Run:
```bash
nix develop -c golangci-lint --version
nix develop -c gofumpt --version
nix develop -c gosec --version
nix develop -c podman --version
nix develop -c docker --version
nix develop -c podman-compose --version
nix develop -c bun --version
nix develop -c jq --version
```
Expected: Version output for each tool, no errors.

**Step 5: Commit**

```bash
git add flake.nix flake.lock
git commit -m "chore(nix): add flake.nix with dev tool declarations"
```

---

### Task 3: Create `.envrc` and Update `.gitignore`

**Files:**
- Create: `.envrc`
- Modify: `.gitignore`

**Step 1: Create `.envrc`**

```bash
use flake
```

**Step 2: Add `.direnv/` to `.gitignore`**

Add after the last line of `.gitignore`:

```gitignore

# Nix / direnv
.direnv/
```

**Step 3: Commit**

```bash
git add .envrc .gitignore
git commit -m "chore(nix): add .envrc and gitignore .direnv/"
```

---

### Task 4: Update `Taskfile.yml` — Add Nix Variable and Prefix Commands

**Files:**
- Modify: `Taskfile.yml`

This is the largest task. Transform every tool command to use `{{.N}}` prefix. Keep shell-only commands (`mkdir`, `echo`, `rm`, `task:` references) as-is.

**Step 1: Add `N` variable to the `vars` section**

Add at the top of the `vars:` block (before `PODMAN_SOCKET`):

```yaml
vars:
  N: "nix develop -c"
```

**Step 2: Prefix `vendor` task**

```yaml
  vendor:
    desc: Run go mod vendor
    silent: true
    cmds:
      - '{{.N}} go mod tidy'
      - '{{.N}} go mod vendor'
```

**Step 3: Prefix `test` task**

```yaml
  test:
    desc: Run unit tests with coverage
    silent: true
    cmds:
      - mkdir -p output
      - '{{.N}} go test -race -count=1 -coverprofile=output/coverage.out -covermode=atomic ./... -tags=!examples'
      - '{{.N}} go tool cover -html=output/coverage.out -o output/coverage.html'
      - 'echo "✓ Coverage: output/coverage.html"'
```

**Step 4: Prefix `test:integration` task**

```yaml
  test:integration:
    desc: Run integration tests including temporal (testcontainers, Docker or Podman required)
    silent: true
    deps: [docker:check]
    env:
      DOCKER_HOST: '{{.CONTAINER_HOST}}'
      TESTCONTAINERS_RYUK_DISABLED: '{{if .CONTAINER_HOST}}true{{end}}'
    cmds:
      - mkdir -p output
      - '{{.N}} bash -c "go list ./... | grep -v examples | xargs go test -race -count=1 -coverprofile=output/coverage-integration.out -covermode=atomic -tags=integration -timeout=15m"'
      - '{{.N}} go tool cover -html=output/coverage-integration.out -o output/coverage-integration.html'
      - 'echo "✓ Integration coverage: output/coverage-integration.html"'
```

Note: The piped `go list | xargs go test` command needs `bash -c` wrapping because `nix develop -c` executes a single command — the pipe would be interpreted by the outer shell otherwise.

**Step 5: Prefix `test:argo` task**

```yaml
  test:argo:
    desc: Run Argo integration tests (requires k8s cluster with Argo Workflows)
    silent: true
    cmds:
      - task: argo:check
      - mkdir -p output
      - '{{.N}} go test -race -count=1 -coverprofile=output/coverage-argo.out -covermode=atomic -tags=argo -timeout=15m ./argo/...'
      - '{{.N}} go tool cover -html=output/coverage-argo.out -o output/coverage-argo.html'
      - 'echo "✓ Argo coverage: output/coverage-argo.html"'
      - '{{.N}} bash -c "go tool cover -func=output/coverage-argo.out | grep total"'
```

**Step 6: Prefix `test:complete` task**

```yaml
  test:complete:
    desc: Run ALL tests (unit + integration + argo) with single comprehensive coverage report
    silent: true
    deps: [docker:check]
    env:
      DOCKER_HOST: '{{.CONTAINER_HOST}}'
      TESTCONTAINERS_RYUK_DISABLED: '{{if .CONTAINER_HOST}}true{{end}}'
    cmds:
      - task: argo:check
      - mkdir -p output
      - echo "🧪 Running all tests (unit + integration + argo)..."
      - '{{.N}} bash -c "go list ./... | grep -v examples | xargs go test -race -count=1 -coverprofile=output/coverage-complete.out -covermode=atomic -tags=integration,argo -timeout=20m -v"'
      - '{{.N}} go tool cover -html=output/coverage-complete.out -o output/coverage-complete.html'
      - 'echo ""'
      - 'echo "✅ Complete test coverage report:"'
      - '{{.N}} bash -c "go tool cover -func=output/coverage-complete.out | grep total"'
      - 'echo "📄 HTML Report: output/coverage-complete.html"'
      - 'echo "📄 Coverage Data: output/coverage-complete.out"'
```

**Step 7: Prefix `lint` task**

```yaml
  lint:
    desc: Run golangci-lint
    silent: true
    cmds:
      - '{{.N}} golangci-lint run ./...'
```

**Step 8: Prefix `fmt` task**

```yaml
  fmt:
    desc: Format all Go files with gofumpt
    cmds:
      - '{{.N}} gofumpt -l -w .'
```

**Step 9: Prefix `ci:test` task**

```yaml
  ci:test:
    desc: Run unit tests for CI (no coverage HTML)
    silent: true
    cmds:
      - '{{.N}} go test -race -count=1 ./... -tags=!examples'
```

**Step 10: Prefix `ci:lint` task**

```yaml
  ci:lint:
    desc: Run golangci-lint for CI
    silent: true
    cmds:
      - '{{.N}} golangci-lint run ./...'
```

**Step 11: Prefix `release` task**

```yaml
  release:
    desc: Run semantic-release (CI only)
    silent: true
    cmds:
      - '{{.N}} bunx semantic-release'
```

**Step 12: Prefix `release:proxy-warmup` task**

```yaml
  release:proxy-warmup:
    desc: Warm Go module proxy with latest tag
    silent: true
    cmds:
      - |
        LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
        if [ -n "$LATEST_TAG" ]; then
          echo "Warming Go proxy for github.com/jasoet/pkg/v2@${LATEST_TAG}"
          {{.N}} bash -c "GOPROXY=https://proxy.golang.org GO111MODULE=on go list -m \"github.com/jasoet/pkg/v2@${LATEST_TAG}\"" || true
        fi
```

**Step 13: Replace `tools` task with `nix:check`**

Remove the `tools` task entirely. Add nix management tasks:

```yaml
  nix:check:
    desc: Verify Nix environment and tool availability
    silent: true
    cmds:
      - |
        if ! command -v nix &> /dev/null; then
          echo "❌ Nix not installed"
          echo "   Install: curl --proto '=https' --tlsv1.2 -sSf -L https://install.determinate.systems/nix | sh -s -- install"
          exit 1
        fi
        echo "✅ Nix $(nix --version)"
        echo "Checking devShell tools..."
        {{.N}} go version
        {{.N}} golangci-lint --version
        {{.N}} gofumpt --version
        {{.N}} gosec --version
        {{.N}} podman --version
        {{.N}} docker --version
        {{.N}} bun --version
        {{.N}} jq --version
        echo "✅ All tools available"

  nix:update:
    desc: Update flake inputs (bump tool versions)
    silent: true
    cmds:
      - nix flake update
      - echo "✅ Flake inputs updated. Run 'task nix:check' to verify."
```

**Step 14: Verify tasks still parse**

Run: `task --list`
Expected: All tasks listed without YAML errors.

**Step 15: Run unit tests through new Nix wrapper**

Run: `task test`
Expected: Tests pass, coverage generated in `output/`.

**Step 16: Run lint through new Nix wrapper**

Run: `task lint`
Expected: Lint passes.

**Step 17: Run nix:check**

Run: `task nix:check`
Expected: All tools listed with versions.

**Step 18: Commit**

```bash
git add Taskfile.yml
git commit -m "feat(nix): wrap all task commands with nix develop -c"
```

---

### Task 5: Update Documentation

**Files:**
- Modify: `INSTRUCTION.md`
- Modify: `README.md`

**Step 1: Update INSTRUCTION.md**

Add Nix prerequisites to the conventions section. Update the Taskfile Commands table (remove `tools`, add `nix:check` and `nix:update`).

**Step 2: Update README.md**

Add a "Development Setup" or "Prerequisites" section documenting:
- Nix installation
- go-task global install
- `task nix:check` to verify

**Step 3: Commit**

```bash
git add INSTRUCTION.md README.md
git commit -m "docs: add nix development environment setup instructions"
```

---

### Task 6: Push and Create PR

**Step 1: Push branch**

Run: `git push -u origin feat/nix-integration`

**Step 2: Create PR**

Run: `gh pr create --title "feat(nix): add reproducible dev environment with nix flakes" --body "..."`

**Step 3: Verify CI passes**

Run: `gh pr checks`
Expected: All CI checks pass (CI uses its own tools, not affected by Nix changes).
