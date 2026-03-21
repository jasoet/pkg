# Code Review: `db` Package

**Date:** 2026-03-21

## Package Summary

Multi-database connectivity library wrapping GORM. Supports PostgreSQL, MySQL, and MSSQL with connection pooling, schema migrations via `golang-migrate` with embedded SQL files, and optional OpenTelemetry tracing + metrics via `uptrace/otelgorm`.

---

## Issues Found

### High

**H1 — Credentials embedded in plaintext DSN returned by public method** (`pool.go:116-135`)

`Dsn()` is exported and returns a fully assembled connection string with the plaintext password. Any caller that logs or stores the returned string exposes the password.

**Fix:** Make `Dsn()` unexported (`dsn()`). Provide a `RedactedDsn()` for debugging.

**H2 — Ping error leaks the full DSN (including password)** (`pool.go:182-184`)

Database drivers embed connection parameters in error messages. Returning `err` unwrapped passes this through to callers who may log it.

### Medium

- M1: SSLMode defaults to `"disable"` for all database types — should default to `"require"`
- M2: `Validate()` does not validate `SSLMode` values per database type
- M3: `collectPoolMetrics` silently ignores gauge creation errors with `//nolint:errcheck`
- M4: `SQLDB()` creates and discards a full GORM pool on every call — resource leak
- M5: `Ping()` is not context-aware — can block indefinitely
- M6: Validation tag/implementation mismatch for `Port` field
- M7: `MaxIdleConns > MaxOpenConns` not cross-validated

### Low

- L1: `migrations.go` is PostgreSQL-only but helper is named generically
- L2: `setupMigration` returns a `zerolog.Logger` as a return value — unusual pattern
- L3: `DatabaseType` has inconsistent naming (`Mysql` vs `MSSQL`)
- L4: Pool connection leak on OTel plugin installation failure
- L5: `time.Sleep` in integration test setup
- L6: README shows passwords in examples

### Security

- No SQL injection vectors found — all queries use parameterized form
- No hardcoded production credentials found

### Recommendations

1. Make `Dsn()` unexported; add `RedactedDsn()`
2. Wrap bare `return nil, err` paths with sanitized messages
3. Change SSL default to `"require"`; validate `SSLMode` values
4. Fix connection leak when OTel plugin fails
5. Accept `context.Context` in `Pool()`; use `PingContext`
