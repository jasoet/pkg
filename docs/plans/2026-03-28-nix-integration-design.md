# Nix Integration Design

## Goal

Replace ad-hoc tool management with Nix flake-based reproducible development environment. All project
commands execute tools through `nix develop -c` to guarantee consistent, pinned versions.

## Prerequisites

| Tool | Install Method | Purpose |
|------|---------------|---------|
| Nix (flakes enabled) | [Determinate installer](https://install.determinate.systems/nix) | Package management |
| go-task | `nix profile install nixpkgs#go-task` | Task runner (global) |
| gh (optional) | Global install | GitHub PR management |

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Nix required | Yes, no fallback | Consistency — assume Linux VM + Nix + Task |
| Command wrapping | Always `nix develop -c` via `{{.N}}` | No direnv detection, works everywhere |
| `task` location | Global, not in flake | Needed to run tasks in the first place |
| `gh` location | Global, not in flake | PR management, not project tooling |
| `bun` | In flake | Needed for `task release` locally |
| direnv / `.envrc` | Included as optional convenience | Not required for task execution |

## Flake Packages

```nix
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
```

## Taskfile Changes

### Nix prefix variable

```yaml
vars:
  N: "nix develop -c"
```

All tool commands use `{{.N}}` prefix. Non-tool shell commands (`mkdir`, `echo`, `rm`) run directly.

### Transformation examples

```yaml
# Before
lint:
  cmds:
    - golangci-lint run ./...

# After
lint:
  cmds:
    - '{{.N}} golangci-lint run ./...'
```

### New tasks

| Task | Command | Purpose |
|------|---------|---------|
| `nix:update` | `nix flake update` | Bump pinned tool versions |
| `nix:check` | Verify nix + flake availability | Smoke test |

### Removed tasks

| Task | Reason |
|------|--------|
| `task tools` | Nix provides all tools — no `go install` needed |

## Files

| File | Action | Notes |
|------|--------|-------|
| `flake.nix` | Create | devShell with all project tools |
| `.envrc` | Create | `use flake` — optional direnv convenience |
| `flake.lock` | Auto-generated | Commit — pins exact versions |
| `Taskfile.yml` | Modify | Add `N` var, prefix commands, add `nix:*` tasks, remove `tools` |
| `.gitignore` | Modify | Add `.direnv/` |

## What Doesn't Change

- `compose.yml` / Podman services
- CI/CD workflows (GitHub Actions provides its own tools)
- `go.mod`, package structure, test patterns
- Git branching strategy

## Implementation Strategy

Create a feature branch (`feat/nix-integration`) to validate all changes don't break CI/CD before merging.
