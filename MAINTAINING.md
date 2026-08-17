# Maintaining Guide

How this library is branched, released, and versioned. For contributing changes, see
[CONTRIBUTING.md](./CONTRIBUTING.md).

## Branch Strategy

| Branch | Module path | Status | Releases |
|---|---|---|---|
| `main` | `github.com/jasoet/pkg/v3` | Released v3 line | `v3.x.y` |
| `next` | `github.com/jasoet/pkg/v3` | v3 development | `v3.x.y-next.N` prereleases |
| `release/v2` | `github.com/jasoet/pkg/v2` | Frozen at v2.13.1 | `2.13.x` emergency patches only |
| `release/v1` | `github.com/jasoet/pkg` | Closed at v1.6.0 | none |

Day-to-day work targets **`next`**, not `main`. Feature and fix branches are cut from
`next` and squash-merged back into it; each merge publishes a `v3.x.y-next.N` prerelease.
`main` only moves when a prerelease line is promoted (see below).

Go's module-path versioning is what makes this work: `/v2` and `/v3` are different modules,
so consumers can import both while migrating. See
[ADR 0001](./docs/adr/0001-freeze-v2-and-ship-v3-as-one-big-bang.md).

## Releasing

Releases are automated by [semantic-release](https://github.com/semantic-release/semantic-release)
on every push to `main`, `next`, and `release/v2`.

### What triggers a release

| Commit type | Bump | Example |
|---|---|---|
| `feat` | minor | `feat(server): add gRPC interceptor` |
| `fix` | patch | `fix(compress): handle empty input` |
| `perf` | patch | `perf(db): reduce query allocations` |
| `refactor` | patch | `refactor(otel): simplify provider setup` |
| Breaking | major | `feat(api)!: remove deprecated method`, or a `BREAKING CHANGE:` footer |

`docs`, `test`, `ci`, `chore`, `style` and `build` never trigger a release.

### Normal changes: squash merge into `next`

The PR title becomes the commit message, so it must be a valid Conventional Commit. Write
the PR description carefully — it becomes the release notes.

**A breaking change needs both** a `BREAKING CHANGE:` footer *and* an entry in
[MIGRATION.md](./MIGRATION.md). Do not rely on the footer alone: several v3 breaks shipped
in `fix:`-typed commits without footers and never reached the generated notes. The
migration guide is the backstop.

### Promoting a line: merge commit into `main`, never squash

**When merging `next` into `main`, use a merge commit.**

```bash
gh pr merge <N> --merge      # correct
gh pr merge <N> --squash     # WRONG — silently produces the wrong version
```

Squashing collapses the whole line into a single commit and **destroys every
`BREAKING CHANGE` footer in it**. semantic-release then analyses one commit against the last
tag on `main` and computes a bump from that alone.

This is measured, not theoretical. Simulating both merges of the v3 line locally and running
`semantic-release --dry-run`:

| Merge strategy | Surviving `BREAKING CHANGE` footers | Computed version |
|---|---|---|
| `--merge` | 22 | **3.0.0** |
| `--squash` | **0** | **2.14.0** |

A `v2.14.0` tag on a module whose path is `/v3` is not installable — and the mistake is only
visible after the tag is published.

You can re-run that check before any promotion, without pushing anything:

```bash
git checkout main && git reset --hard origin/main
git merge --no-ff --no-edit origin/next
GITHUB_TOKEN=$(gh auth token) bunx semantic-release --dry-run --no-ci
git reset --hard origin/main    # discard the simulation
```

A merge commit also produces complete release notes, since every `feat`/`fix` on the line
stays individually attributed.

### Pre-promotion checklist

1. `task ci:check` and `go vet -tags='example integration argo' ./...` clean on `next`.
2. Integration suite green: `task test:integration`.
3. `MIGRATION.md` covers every break on the line, including any that shipped without a
   footer.
4. Package coverage figures in `README.md` regenerated.
5. Open the PR `next` → `main` and confirm the **API compatibility check reports
   informationally, not blocking** — `ci.yml` keys that off `head_ref == 'next'`.
   gorelease reporting `Inferred base version: none` is expected until the first
   non-prerelease tag exists on the new major.
6. Merge with `--merge`, then confirm the published tag is what you expected before
   announcing anything.

## CI Pipelines

| Workflow | Triggers | Does |
|---|---|---|
| `ci.yml` | push to `main`/`next`/`release/v2` and tags; PRs targeting them | lint, race tests, `go vet` over build-tagged code, gorelease API check |
| `release.yml` | push to `main`/`next`/`release/v2`; manual dispatch | race tests, integration tests, semantic-release, Go proxy warmup |

Both run on the self-hosted `[self-hosted, local, macOS, ARM64]` runner.

The gorelease API check is **blocking** everywhere except where `next` is involved
(`ref_name`, `base_ref`, or `head_ref` equal to `next`), because breaking changes are the
point of the v3 line. It is pinned to the `golang.org/x/exp` pseudo-version in `go.mod`;
bump both together.

`ci.yml` runs `go vet` with `-tags='example integration argo'` as a separate step. `task
check` compiles only untagged code, and example- or integration-only breakage has slipped
through that gap before.

## Import Paths

```go
import "github.com/jasoet/pkg/v3/server"
```

```bash
go get github.com/jasoet/pkg/v3@latest
```

## Testing Before a Release

```bash
task test              # unit
task test:integration  # integration (Docker or Podman required)
task lint
task test:complete     # everything, including argo (needs a k8s cluster)
```
