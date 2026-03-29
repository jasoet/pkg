# Project Instructions

<!-- AI: Read this file at the start of every session. Update it when conventions, -->
<!-- architecture, or key paths change. Also keep README.md in sync. -->

## Project Overview

Production-ready Go utility library (v2) with OpenTelemetry instrumentation. 15 packages: otel, config, logging, db, docker, server, grpc, rest, concurrent, temporal, ssh, compress, argo, retry, base32.

**Module Path:** `github.com/jasoet/pkg/v2`
**Go Version:** 1.26+ (uses generics)
**Test Coverage:** 85%
**v1 Branch:** [`release/v1`](https://github.com/jasoet/pkg/tree/release/v1) — final v1 release (v1.6.0), no longer maintained. Use `go get github.com/jasoet/pkg@v1.6.0` for projects that don't need OpenTelemetry.

## ABSOLUTE RULE — Git Authorship

**NEVER add AI (Claude, Copilot, or any AI) as co-author, committer, or contributor in git commits.**
Only the user's registered email may appear in commits. This is company policy — commits with AI
authorship WILL BE REJECTED. Do not use `--author`, `Co-authored-by`, or any other mechanism to
attribute commits to AI. This applies to ALL commits, including those made by tools and subagents.

## Conventions

- **Node.js**: Always use `bun`/`bunx` (never node, npm, npx).
- **Commands**: Always use `task <name>` to run commands. Run `task --list` to discover available tasks. If a command is important or repeated but has no task, suggest adding it to `Taskfile.yml`.
- **Brainstorming**: New topics or planning always start with brainstorming skill first. If unsure, ask the user.
- **Superpowers**: Ensure superpowers skills are installed. Use TDD for implementation, systematic-debugging for bugs.
- **Commits**: Use Conventional Commits. Format: `<type>(<scope>): <description>`. Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `perf`, `ci`.
- **Branching**: New branch for each feature/fix (`feat/...`, `fix/...`). PR with squash merge. Use `gh` for PR status and CI checks.
- **Nix**: All dev tools provided via `flake.nix`. Use `task <name>` which wraps commands with `nix develop -c` (via `{{.N}}` Taskfile variable). Prerequisites: Nix (with flakes), go-task (global via Homebrew). direnv/`.envrc` is optional convenience — tasks work without it.
- **Containers**: Dual Docker/Podman support. Container CLIs (`podman`, `docker-client`, `podman-compose`) are in the flake for version consistency; daemons are system-level. This is a shared library — consumers use either runtime.
- **Patterns**: Functional options for configuration. OTelConfig always injected via `With*()` options, never serialized (`yaml:"-" mapstructure:"-"`). Use `otel.Layers.Start*()` for instrumentation.
- **Self-maintaining docs**: Update `INSTRUCTION.md`, `README.md`, and `AI_PATTERN.md` when making significant changes.
- **AI_PATTERN.md**: For AI working on projects that USE this library. Keep it as an index pointing to module READMEs and examples.
- **PROJECT_TEMPLATE.md**: For AI scaffolding new Go projects with this library.

## Key Paths

| Path | Purpose |
|------|---------|
| `<module>/` | Package source (15 packages at root level) |
| `<module>/README.md` | Per-package documentation |
| `<module>/examples/` | Per-package runnable examples (`//go:build example`) |
| `<module>/*_test.go` | Unit tests (no build tag) |
| `<module>/*_integration_test.go` | Integration tests (`//go:build integration`) |
| `docs/plans/` | Design docs and implementation plans |
| `.claude/` | Claude Code hooks and settings |
| `flake.nix` | Nix flake — dev tool declarations |
| `.envrc` | direnv auto-activation (optional) |
| `Taskfile.yml` | All project commands |
| `INSTRUCTION.md` | AI dev context (this file) |
| `AI_PATTERN.md` | AI library consumer patterns index |
| `PROJECT_TEMPLATE.md` | New project scaffolding guide |
| `README.md` | Human documentation |

## Taskfile Commands

| Task | Description |
|------|-------------|
| `task test` | Unit tests with coverage |
| `task test:integration` | Integration tests (Docker/Podman required) |
| `task test:argo` | Argo tests (k8s cluster required) |
| `task test:complete` | All tests with comprehensive coverage |
| `task lint` | Run golangci-lint |
| `task fmt` | Format with gofumpt |
| `task vendor` | go mod tidy + vendor |
| `task check` | test + lint |
| `task clean` | Remove build artifacts |
| `task nix:check` | Verify Nix environment and tool availability |
| `task nix:update` | Update flake inputs (bump tool versions) |
| `task docker:check` | Verify Docker/Podman availability |
| `task k8s:check` | Verify kubectl and cluster |
| `task argo:check` | Verify Argo Workflows |
| `task ci:test` | Unit tests for CI (no coverage HTML) |
| `task ci:lint` | Lint for CI |
| `task ci:check` | CI test + lint |
| `task release` | Run semantic-release (CI only) |
| `task release:proxy-warmup` | Warm Go module proxy with latest tag |

## Testing Strategy

**Build Tags:** Unit (none), Integration (`integration`), Argo (`argo`), Examples (`example`)
**Assertions:** `github.com/stretchr/testify/assert` and `require`
**Integration:** Uses testcontainers (Docker/Podman). 15min timeout. Cleanup: `defer container.Terminate(ctx)`
**Coverage:** Generated in `output/coverage*.html`

## Adding a New Package

1. Create: `newpkg/`, `newpkg/README.md`, `newpkg/newpkg.go`, `newpkg/newpkg_test.go`
2. Follow: functional options, `OTelConfig *otel.Config` with `yaml:"-" mapstructure:"-"`, `otel.Layers.Start*()`, testify
3. Update: README.md package table, AI_PATTERN.md package reference, PROJECT_TEMPLATE.md if relevant
