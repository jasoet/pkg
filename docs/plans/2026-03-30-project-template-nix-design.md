# PROJECT_TEMPLATE Nix Development Environment Update

**Date:** 2026-03-30
**Status:** Approved

## Goal

Update PROJECT_TEMPLATE.md to make Nix the default dev environment for consumer projects, matching the three-layer tool strategy established in the `pkg` library itself.

## Design Decisions

- **Nix is the default** with a brief note that tools can be installed globally if Nix is skipped
- **New Section 0** before directory layout — foundational context that informs everything else
- **Moderate detail** — enough for AI agents to scaffold correctly and understand rationale, but no installation/troubleshooting (that's user-facing, not scaffolding)

## Changes

### 1. New Section 0: Dev Environment Setup

- Update prerequisite line: Nix + go-task global, Go provided by flake
- Three-layer strategy table (Homebrew / Nix / Podman)
- Why Nix: reproducible, per-project, cross-platform
- Why `go-task` stays global: chicken-and-egg problem
- Pattern A (`nix develop -c` prefix) as default Taskfile integration
- `flake.nix` Go service template
- `.envrc` content and `.gitignore` additions

### 2. Section 1: Directory Layout

- Add `flake.nix`, `flake.lock`, `.envrc` to both layout trees
- Add rows to directory purpose table
- Update `docker/compose.yml` and `Taskfile.yml` descriptions to align with three-layer terminology

### 3. Section 13: Taskfile Configuration

- Add `vars: N: "nix develop -c"` block
- Prefix all tool commands with `{{.N}}`
- Infrastructure commands stay bare
- Add `nix:check` and `nix:update` standard tasks
- Brief note explaining the prefix pattern

### 4. Section 14: Architecture Rules

- New "Dev Environment" subsection at top of checklist
- Rules: flake.nix exists, go-task not in flake, Taskfile uses {{.N}}, .direnv/ gitignored
- Existing items shift numbering
