# v3 Phase 4: config + retry Unification

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring `config` and `retry` onto the v3 conventions: functional options, no leaked third-party types in public APIs, OTelConfig tag contract (retry), truthful compile-checked docs.

**Architecture:** config drops viper from its public surface (options replace the `func(*viper.Viper)` callback and the exported `NestedEnvVars`); retry converts its builder methods into package-level options and moves validation from panic-at-set-time to error-at-Do-time.

**Tech Stack:** Go 1.26, viper (internal only after this phase), cenkalti/backoff/v4, testify.

## Global Constraints

- Work on `next`, module `github.com/jasoet/pkg/v3`. Conventional Commits; NEVER AI attribution. Breaking commits carry `!` + `BREAKING CHANGE:` footer.
- Verification per task: `nix develop -c go build ./... && nix develop -c go build -tags=example,integration ./...` plus focused `go test`. `task check` green at phase end.
- Package READMEs must match the new API and each snippet must have a compile-checked Example test.
- Backlog of record: `docs/plans/2026-07-22-v3-audit-backlog.md` (config and retry sections).

---

### Task 1: config — de-leak viper, options API

**Files:**
- Modify: `config/config.go`
- Test: `config/options_test.go` (new)
- Modify: `config/config_test.go` (update callers of removed APIs)

**Interfaces:**
- Produces:
  - `type Option func(*viper.Viper)` — unexported use only; consumers never name viper.
  - `func WithEnvPrefix(prefix string) Option`
  - `func WithDefaults(defaults map[string]any) Option`
  - `func WithNestedEnvVars(prefix string, keyDepth int, configPath string) Option`
  - `func LoadStringWithOptions[T any](configString string, opts ...Option) (*T, error)`
  - REMOVED: `LoadStringWithConfig`, `NestedEnvVars` (exported).
  - KEPT unchanged: `LoadString[T](configString string, envPrefix ...string)` (simple path; doc: only the first envPrefix value is used).

- [ ] **Step 1: Write the failing test**

Create `config/options_test.go`:
```go
package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jasoet/pkg/v3/config"
)

type appCfg struct {
	Debug  bool              `yaml:"debug"`
	Server struct{ Port int } `yaml:"server"`
	Users  map[string]map[string]string `yaml:"users"`
}

func TestLoadStringWithOptions_DefaultsAndPrefix(t *testing.T) {
	cfg, err := config.LoadStringWithOptions[appCfg](`server: {port: 8080}`,
		config.WithDefaults(map[string]any{"debug": true}),
		config.WithEnvPrefix("APP"),
	)
	require.NoError(t, err)
	assert.True(t, cfg.Debug)
	assert.Equal(t, 8080, cfg.Server.Port)
}

func TestLoadStringWithOptions_NestedEnvVars(t *testing.T) {
	t.Setenv("APP_USERS_ADMIN_NAME", "alice")
	cfg, err := config.LoadStringWithOptions[appCfg](``,
		config.WithNestedEnvVars("APP", 1, "users"),
	)
	require.NoError(t, err)
	assert.Equal(t, "alice", cfg.Users["admin"]["name"])
}

func TestLoadStringWithOptions_NestedDoesNotOverrideYAML(t *testing.T) {
	// Precedence contract: nested env vars fill only keys absent from YAML.
	t.Setenv("APP_USERS_ADMIN_NAME", "alice")
	cfg, err := config.LoadStringWithOptions[appCfg](`users: {admin: {name: bob}}`,
		config.WithNestedEnvVars("APP", 1, "users"),
	)
	require.NoError(t, err)
	assert.Equal(t, "bob", cfg.Users["admin"]["name"])
}
```

Run: `nix develop -c go test ./config/ -run TestLoadStringWithOptions -count=1`
Expected: FAIL — undefined symbols.

- [ ] **Step 2: Implement**

In `config/config.go`:
- Add `type Option func(*viper.Viper)` and the three options. `WithEnvPrefix` sets `v.SetEnvPrefix(prefix)`; `WithDefaults` loops `v.SetDefault(k, val)`; `WithNestedEnvVars` calls the existing (now unexported) `nestedEnvVars(prefix, keyDepth, configPath, v)`.
- Add `LoadStringWithOptions[T]`: same body as today's `LoadStringWithConfig` but: default prefix `ENV`, apply opts AFTER `AutomaticEnv()`/replacer setup and BEFORE `ReadConfig`... — NOTE: match current semantics exactly: viper.New → prefix/replacer/AutomaticEnv → ReadConfig → then options (today configFn runs after ReadConfig). Keep that order: options run after ReadConfig, before Unmarshal. Document: options that must precede parsing are not supported.
- DELETE `LoadStringWithConfig` and exported `NestedEnvVars` (rename the helper to unexported `nestedEnvVars` with identical body, including the not-goroutine-safe doc note).
- Update `config/config_test.go` callers: `LoadStringWithConfig[T](s, fn)` → `LoadStringWithOptions[T](s, ...)`; direct `NestedEnvVars(...)` test callers → `LoadStringWithOptions` with `WithNestedEnvVars` (behavior identical).

- [ ] **Step 3: Verify**

```bash
nix develop -c go build ./...
nix develop -c go build -tags=example,integration ./...
nix develop -c go test ./config/ -count=1
```
Expected: all green.

- [ ] **Step 4: Commit**

```bash
git add config/
git commit -m "feat(config)!: replace viper-leaking APIs with functional options

BREAKING CHANGE: LoadStringWithConfig and NestedEnvVars removed; use LoadStringWithOptions with WithEnvPrefix/WithDefaults/WithNestedEnvVars."
```

---

### Task 2: config — README + Example tests

**Files:**
- Modify: `config/README.md`
- Test: `config/example_test.go` (new, package config_test)

**Interfaces:**
- Produces: every README snippet backed by an Example test in `config/example_test.go`.

- [ ] **Step 1: Write Example tests**

Create `config/example_test.go` with `ExampleLoadString`, `ExampleLoadStringWithOptions`, `ExampleWithNestedEnvVars` (use t.Setenv-free approach: set real env via os.Setenv + defer Unsetenv since Examples have no *testing.T — use os.Setenv/Unsetenv directly). Add `// Output:` blocks where deterministic.

- [ ] **Step 2: Rewrite config/README.md**

Fix per backlog: correct import path (`/v3`), remove broken example links (point at `examples/config/`), remove fabricated benchmark and stale Go-version claim, document the options API and the nested-env precedence contract (env fills only YAML-absent keys), remove contradictory YAML naming guidance.

- [ ] **Step 3: Verify**

`nix develop -c go test ./config/ -count=1 -v | grep -E 'Example|ok'` — all examples execute and pass.

- [ ] **Step 4: Commit**

```bash
git add config/
git commit -m "docs(config): align README with v3 options API; add compile-checked examples"
```

---

### Task 3: retry — functional options, validation at Do-time

**Files:**
- Modify: `retry/retry.go`
- Test: `retry/options_test.go` (new)
- Modify: `retry/retry_test.go` (update builder-style callers)

**Interfaces:**
- Produces:
  - `type Option func(*Config)`; `func New(opts ...Option) Config`
  - Options: `WithName(string)`, `WithOTelConfig(*otel.Config)` (renamed from `WithOTel`), `WithMaxRetries(uint64)`, `WithInitialInterval(time.Duration)`, `WithMaxInterval(time.Duration)`, `WithMultiplier(float64)`, `WithRandomizationFactor(float64)`
  - `Config` fields gain yaml/mapstructure tags; `OTelConfig *otel.Config` tagged `yaml:"-" mapstructure:"-"`
  - REMOVED: all builder methods on Config (`WithName`, `WithOTel`, `WithMaxRetries`, `WithInitialInterval`, `WithMaxInterval`, `WithMultiplier`, `WithRandomizationFactor`)
  - Validation: invalid values (Multiplier <= 1, InitialInterval <= 0, MaxInterval < InitialInterval, RandomizationFactor outside [0,1]) make `Do`/`DoWithNotify` return an error before the first attempt — never panic. (Current panics in setters removed with the setters.)
  - KEPT: `DefaultConfig() Config`, `Do`, `DoWithNotify`, `Permanent` signatures.

- [ ] **Step 1: Write the failing test**

Create `retry/options_test.go`:
```go
package retry_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/jasoet/pkg/v3/retry"
)

func TestNew_AppliesOptions(t *testing.T) {
	cfg := retry.New(
		retry.WithName("db.connect"),
		retry.WithMaxRetries(3),
		retry.WithInitialInterval(100*time.Millisecond),
	)
	assert.Equal(t, "db.connect", cfg.Name)
	assert.Equal(t, uint64(3), cfg.MaxRetries)
	assert.Equal(t, 100*time.Millisecond, cfg.InitialInterval)
}

func TestDo_InvalidConfigReturnsErrorNotPanic(t *testing.T) {
	cfg := retry.New(retry.WithMultiplier(0.5))
	err := retry.Do(context.Background(), cfg, func(ctx context.Context) error {
		return errors.New("boom")
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "multiplier")
}
```

Run: `nix develop -c go test ./retry/ -run 'TestNew_|TestDo_Invalid' -count=1`
Expected: FAIL — undefined `retry.New` / options.

- [ ] **Step 2: Implement**

In `retry/retry.go`:
- Add `OTelConfig` tags (`yaml:"-" mapstructure:"-"`); add yaml+mapstructure tags to exported fields matching existing names where trivially derivable.
- Add `Option`, `New(opts ...Option) Config` (start from `DefaultConfig()`, apply opts), and the seven options (bodies = today's setter bodies MINUS panics).
- Delete the seven builder methods.
- Add unexported `func (c Config) validate() error` (checks listed in Interfaces); call it first in `Do` and `DoWithNotify`.
- Update `retry/retry_test.go` builder-style calls to `retry.New(...)`.

- [ ] **Step 3: Register in archtest**

In `internal/archtest/archtest_test.go` add `"retry": reflect.TypeOf(retry.Config{}),` to `compliantConfigs` (import retry). In `internal/archtest/options_test.go` add `_ func(*otel.Config) retry.Option = retry.WithOTelConfig` (import otel + retry).

- [ ] **Step 4: Verify**

```bash
nix develop -c go build ./... && nix develop -c go build -tags=example,integration ./...
nix develop -c go test ./retry/ ./internal/archtest/ -count=1
grep -rn 'retry\.DefaultConfig()\.\|\.WithOTel(' --include='*.go' . | grep -v vendor | grep -v '_test.go' || echo CLEAN
```
Expected: green; CLEAN (also fix non-test callers, e.g. examples, db, temporal if any use the old builder style — convert to `retry.New(...)`).

- [ ] **Step 5: Commit**

```bash
git add retry/ internal/archtest/
git commit -m "feat(retry)!: functional options, OTelConfig tags, Do-time validation

BREAKING CHANGE: Config builder methods removed (use retry.New with options); WithOTel renamed WithOTelConfig; invalid configs now error at Do time instead of panicking in setters."
```

---

### Task 4: retry — README + Example tests

**Files:**
- Modify: `retry/README.md`, `examples/retry/` (README + example.go if stale)
- Test: `retry/example_test.go` (new)

- [ ] **Step 1: Example tests**

Create `retry/example_test.go`: `ExampleNew`, `ExampleDo`, `ExamplePermanent`. Use deterministic `// Output:` where possible (e.g., an operation failing twice then succeeding with tiny intervals).

- [ ] **Step 2: Rewrite retry/README.md**

Per backlog: document ALL Config fields (one is currently undocumented), the options API, Do-time validation behavior; fix the example README's "expected output" to be reproducible.

- [ ] **Step 3: Verify** — `nix develop -c go test ./retry/ -count=1 -v | grep -E 'Example|ok'`

- [ ] **Step 4: Commit**

```bash
git add retry/ examples/retry/
git commit -m "docs(retry): align README with v3 options API; add compile-checked examples"
```

---

### Task 5: Phase verification and push

- [ ] **Step 1: Full gate**

```bash
task check
nix develop -c go build -tags=example,integration ./...
```
Expected: green.

- [ ] **Step 2: Push** — `git push origin next`
