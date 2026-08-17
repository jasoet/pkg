# v3 Phase 13: Utilities — compress, concurrent, base32

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Finish the package sweep: fix compress's error-contract inconsistencies and silently-ignored option, concurrent's flipped parameter order and duplicate wrapper, base32's systematically wrong docs — with compile-checked examples everywhere.

**Architecture:** Three independent tracks in one phase. compress keeps its signatures (documented asymmetry) but gets consistent sentinel errors and a working option; concurrent gets a breaking parameter-order fix; base32 gets golden-value tests and input normalization.

**Tech Stack:** Go 1.26, testify.

## Global Constraints

- Work on `next`, module `github.com/jasoet/pkg/v3`. Conventional Commits; NEVER AI attribution. Breaking commits carry `!` + `BREAKING CHANGE:` footer.
- Verification per task: `nix develop -c go build ./... && nix develop -c go build -tags=example,integration ./...` plus focused tests; `task check` green at phase end.
- Backlog: `docs/plans/2026-07-22-v3-audit-backlog.md` (compress, concurrent, base32 sections). No OTel for these packages — concurrent/compress/base32 are pure stateless utilities, exempt from the OTelConfig convention (recorded decision).

---

### Task 1: compress — error contract + ignored option + docs

**Files:**
- Modify: `compress/gz.go`, `compress/tar.go`, `compress/errors.go` (as needed)
- Test: `compress/errors_test.go` (extend), `compress/example_test.go` (new)
- Modify: `compress/README.md`

**Interfaces:**
- Produces: every guard-rail failure maps to a documented sentinel (errors.Is-able); the option `UnGz` silently ignores (audit claim — find it in gz.go) either works or is removed with a footer; README documents ALL options, correct error-matching against real sentinels, no fabricated benchmarks/file-mode claims, /v3 paths.
- First read gz.go/tar.go/errors.go fully; the audit claims: absolute-path required for `UnGz` but not `UnTar` (align or document), sentinel errors on only half the guard rails (complete the set), one option silently ignored by `UnGz`.

- [ ] **Step 1: Write the failing tests**

1. Table test: every guard-rail rejection (path traversal, oversized, bad mode, etc.) matches a sentinel via `errors.Is` — list each current failure and the sentinel it SHOULD map to (per errors.go).
2. Test for the silently-ignored option (identify it first from gz.go): asserting it takes effect.

Run: FAIL on the gaps found.

- [ ] **Step 2: Implement**

Complete sentinel mapping; make the ignored option work (or delete it with BREAKING footer if it's meaningless); align or explicitly document the UnGz/UnTar path asymmetry.

- [ ] **Step 3: Rewrite compress/README.md** + Example tests

Fix per backlog: all options documented, quick-start that runs, real error-matching examples, no fabrications. `compress/example_test.go`: `ExampleGz`, `ExampleUnGz` (deterministic with `// Output:` where possible — gzip of fixed content).

- [ ] **Step 4: Verify + Commit**

```bash
nix develop -c go test ./compress/ -count=1
nix develop -c go build -tags=example,integration ./...
git add compress/
git commit -m "feat(compress)!: complete sentinel error contract; fix ignored UnGz option; rewrite docs

BREAKING CHANGE: <only if an option is removed or behavior changed — else drop the !>"
```

---

### Task 2: concurrent — parameter order + duplicate wrapper + docs

**Files:**
- Modify: `concurrent/execution.go`
- Modify callers: `concurrent/execution_test.go`, examples using `ExecuteConcurrentlyTyped`
- Modify: `concurrent/README.md`
- Test: `concurrent/example_test.go` (new)

**Interfaces:**
- Produces: `ExecuteConcurrentlyTyped[R, T](ctx, funcs map[string]Func[T]) (map[string]R, error)` — wait, read the current signature first (execution.go:105): `ExecuteConcurrentlyTyped[T any, R any](ctx, funcs map[string]Func[T]) (map[string]R, error)` — the audit says "flipped parameter order" (T before R while the return is R). Fix: swap type params to `[R, T]` for call-site readability `ExecuteConcurrentlyTyped[Result, Input]`, or align with ExecuteConcurrently's `[T]` shape. Decide on reading the code: whichever makes `ExecuteConcurrentlyTyped[Output, Input]` read correctly.
- REMOVED: the thin duplicate wrapper function the audit flagged (identify it — a wrapper that just calls ExecuteConcurrently).
- No OTel (exempt — recorded).

- [ ] **Step 1: Write the failing test**

Compile-level: `ExecuteConcurrentlyTyped[string, int](ctx, funcs)` returns `map[string]string` given `Func[int]` — assert the intended reading compiles and runs (today it's flipped).

Run: FAIL to compile/run as intended.

- [ ] **Step 2: Implement**

Swap type params; delete the duplicate wrapper (grep its callers first); update all callers.

- [ ] **Step 3: Rewrite concurrent/README.md** + Example tests

Remove fabricated benchmarks and the false 100%-coverage claim, fix broken links and import path, fix run instructions (`-tags=example`). `concurrent/example_test.go`: `ExampleExecuteConcurrently` (deterministic `// Output:`), `ExampleExecuteConcurrentlyTyped`.

- [ ] **Step 4: Verify + Commit**

```bash
nix develop -c go test ./concurrent/ -count=1
git add concurrent/ examples/concurrent/
git commit -m "feat(concurrent)!: fix ExecuteConcurrentlyTyped type-parameter order; drop duplicate wrapper

BREAKING CHANGE: ExecuteConcurrentlyTyped type parameters are now [R, T] (result first); the duplicate wrapper is removed."
```

---

### Task 3: base32 — golden tests + normalization + docs

**Files:**
- Modify: `base32/base32.go`, `base32/checksum.go` (normalization)
- Test: `base32/golden_test.go` (new), `base32/example_test.go` (new or extend)
- Modify: `base32/README.md`, doc comments with wrong values

**Interfaces:**
- Produces: golden regression tests pinning EncodeBase32/AppendChecksum/ValidateChecksum/DecodeBase32 outputs for known vectors; `AppendChecksum` and `ValidateChecksum` normalize their input (via NormalizeBase32) so dashed/lowercase input works; every doc-comment and README encoded value verified against actual function output (generate the values BY RUNNING the functions, not from the old docs).

- [ ] **Step 1: Write the failing tests**

1. `golden_test.go`: known vectors — capture CURRENT correct outputs by running the functions first (encode 12345→8 chars, checksum round-trips, normalize cases), then pin them.
2. `TestAppendChecksum_Normalizes`: `AppendChecksum("0000c1p9")` (lowercase) and dashed input succeed.

Run: normalization tests FAIL on current code.

- [ ] **Step 2: Implement**

`AppendChecksum`/`ValidateChecksum` call `NormalizeBase32` on input first (document that clean input is unaffected). Fix every wrong encoded value in README + doc comments using the golden values from Step 1. Fix the examples that produce empty output (dashed input rejected — now works via normalization). Fix run instructions.

- [ ] **Step 3: Verify + Commit**

```bash
nix develop -c go test ./base32/ -count=1
nix develop -c go run -tags=example ./examples/base32  # must produce non-empty expected output
git add base32/ examples/base32/
git commit -m "fix(base32): normalize input in AppendChecksum/ValidateChecksum; golden tests; correct doc values"
```

---

### Task 4: Phase verification + final review + push

- [ ] **Step 1: Full gate**

```bash
task check
nix develop -c go build -tags=example,integration ./...
```

- [ ] **Step 2: Push** — `git push origin next`
