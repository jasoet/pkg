# Project Instructions

<!-- AI: Read this file at the start of every session. Update it when conventions, -->
<!-- architecture, or key paths change. Also keep README.md in sync. -->

## Project Overview

Production-ready Go utility library (v2) with OpenTelemetry instrumentation. 15 packages: otel, config, logging, db, docker, server, grpc, rest, concurrent, temporal, ssh, compress, argo, retry, base32.

**Module Path:** `github.com/jasoet/pkg/v2`
**Go Version:** 1.24+ (uses generics)
**Test Coverage:** 85%

## Conventions

- **Commands**: Always use `task <name>` to run commands. Run `task --list` to discover available tasks. If a command is important or repeated but has no task, suggest adding it to `Taskfile.yml`.
- **Brainstorming**: New topics or planning always start with brainstorming skill first. If unsure, ask the user.
- **Superpowers**: Ensure superpowers skills are installed. Use TDD for implementation, systematic-debugging for bugs.
- **Commits**: Use Conventional Commits. Format: `<type>(<scope>): <description>`. Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `perf`, `ci`. Never add AI as co-author or committer.
- **Branching**: New branch for each feature/fix (`feat/...`, `fix/...`). PR with squash merge. Use `gh` for PR status and CI checks.
- **Containers**: Dual Docker/Podman support. This is a shared library — consumers use either runtime.
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
| `task tools` | Install dev tools |
| `task docker:check` | Verify Docker/Podman availability |
| `task k8s:check` | Verify kubectl and cluster |
| `task argo:check` | Verify Argo Workflows |

## Testing Strategy

**Build Tags:** Unit (none), Integration (`integration`), Argo (`argo`), Examples (`example`)
**Assertions:** `github.com/stretchr/testify/assert` and `require`
**Integration:** Uses testcontainers (Docker/Podman). 15min timeout. Cleanup: `defer container.Terminate(ctx)`
**Coverage:** Generated in `output/coverage*.html`

## Adding a New Package

1. Create: `newpkg/`, `newpkg/README.md`, `newpkg/newpkg.go`, `newpkg/newpkg_test.go`
2. Follow: functional options, `OTelConfig *otel.Config` with `yaml:"-" mapstructure:"-"`, `otel.Layers.Start*()`, testify
3. Update: README.md package table, AI_PATTERN.md package reference
