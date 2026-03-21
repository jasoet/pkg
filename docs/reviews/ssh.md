# Code Review: `ssh` Package

**Date:** 2026-03-21

## Package Summary

Implements a local-to-remote SSH port forwarding tunnel. Wraps `golang.org/x/crypto/ssh` to establish an SSH client connection, opens a local TCP listener, and for each inbound connection spawns a goroutine that bidirectionally proxies data through the SSH session to a fixed remote endpoint.

---

## Issues Found

### Critical

**C1 — `t.client` written outside lock in `Start()` — data race** (`tunnel.go:145`)

`t.client` assigned without holding `t.mu`. `Close()` reads `t.client` without mutex. `forward()` also reads `t.client` without synchronization. If `Close()` and `Start()` are called concurrently, unprotected read/write pair on `t.client`.

**C2 — Accept loop continues after `Close()` — no shutdown signal** (`tunnel.go:168-177`)

Accept loop exits only when `listener.Accept()` returns error. After `Close()`, `t.client` is already closed, but newly-accepted connections call `t.client.Dial()` on a closed SSH client. No channel, context, or atomic flag to signal stop.

### High

- H1: `Start()` can be called on already-started tunnel — leaks previous SSH client and listener
- H2: `Close()` does not close SSH client under lock; `t.client` not zeroed
- H3: `Password` field carries `yaml:"password"` tag — serializes to YAML
- H4: Listener binds to `localhost` only, no configurable bind address
- H5: `forward()` goroutines not tracked — `Close()` doesn't wait for in-flight goroutines

### Medium

- M1: `Start()` uses `context.Background()` — doesn't accept caller context
- M2: No input validation in `New()` or `Start()` (port, host, user)
- M3: `io.Copy` errors silently discarded
- M4: `#nosec G106` suppresses `InsecureIgnoreHostKey` warning globally; no log warning when active
- M5: `getHostKeyCallback` error message opaque to end-users
- M6: README claims "Password Only" in limitations — inaccurate; key-based auth is implemented

### Low

- L1: No `CloseWrite()` half-close for streaming protocols
- L2: Error message capitalization inconsistency
- L3: Test file embeds a real Ed25519 private key
- L4: Integration tests use `time.Sleep` instead of retry/poll
- L5: `Start()` doesn't return actual local port when `LocalPort: 0`
- L6: Both `InsecureIgnoreHostKey` and `KnownHostsFile` can be set simultaneously

### Security

| Finding | Severity |
|---------|----------|
| `t.client` race: written/read without mutex | Critical |
| `Password` field serializes to YAML | High |
| No warning when `InsecureIgnoreHostKey=true` | Medium |
| Real Ed25519 private key in test file | Low |

### Recommendations

1. Protect `t.client` with `t.mu` everywhere; set `nil` in `Close()` under lock
2. Add shutdown channel and `sync.WaitGroup` for accept/forward goroutines
3. Add `yaml:"-"` to `Password` field
4. Add `Start(ctx context.Context)` signature
5. Add input validation for `Port`, `LocalPort`, `Host`, `RemoteHost`, `User`
6. Log warning when `InsecureIgnoreHostKey=true`
7. Replace committed test key with programmatically-generated key
