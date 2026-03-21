# Code Review: `config` Package

**Date:** 2026-03-21

## Package Summary

The `config` package provides a thin generic wrapper around Viper for loading YAML configuration into typed Go structs, with automatic environment variable override support. Three public functions: `LoadString[T]`, `LoadStringWithConfig[T]`, and `NestedEnvVars`.

---

## Issues Found

### Critical

**C1 — `NestedEnvVars` panics on negative `keyDepth`** (`config.go:77`)

```go
entityName := strings.ToLower(keyParts[keyDepth])    // panics if keyDepth < 0
fieldName := strings.ToLower(keyParts[keyDepth+1])
```

No lower-bound validation. Passing a negative value causes an immediate `index out of range` panic.

**Fix:** Add `if keyDepth < 0 { return }` at the top.

### High

**H1 — Race condition on shared `*viper.Viper` in `NestedEnvVars`** (`config.go:94-95`)

`viperConfig.IsSet()` (read) immediately followed by `viperConfig.Set()` (write) with no synchronization. Viper is not goroutine-safe for concurrent reads and writes. Confirmed by `-race` detector.

**H2 — Silent data loss for multi-segment env var keys** (`config.go:77-83`)

Function extracts exactly two segments from the key. Additional underscore-separated segments are silently dropped. `PREFIX_DB_CONNECTION_TIMEOUT=30` with `keyDepth=1` stores under `db.connection` — the word `TIMEOUT` is lost.

**Fix:** Join trailing segments: `fieldName = strings.ToLower(strings.Join(keyParts[keyDepth+1:], "_"))`.

### Medium

- M1: `keyDepth` semantics not obvious; prefix not stripped before split
- M2: Variadic `envPrefix` silently ignores extra values — API anti-pattern
- M3: Errors returned bare without wrapping or sentinel types

### Low

- L1: Stale `configFn` mention in `LoadString` doc comment
- L2: Tests use `os.Setenv` instead of `t.Setenv`
- L3: No tests for any error/failure path
- L4: `TestConfig` struct contradicts package's own `mapstructure` tag advice

### Recommendations

1. Add negative `keyDepth` guard immediately
2. Document thread-safety constraints on `NestedEnvVars`
3. Fix field-name truncation by joining trailing segments
4. Redesign `keyDepth`/prefix coupling — strip prefix first
5. Replace variadic `envPrefix` with functional options
6. Wrap errors with `fmt.Errorf`
