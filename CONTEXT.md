# pkg

A Go utility library of independent packages, each wrapping one concern (HTTP, gRPC,
database, containers, workflows) with OpenTelemetry instrumentation built in. Consumers
import only the packages they need.

## Language

### Package kinds

**Utility package**:
A package whose wrapped dependency is incidental to why a consumer imports it — `rest`,
`config`, `db`, `docker`, `server`, `grpc`, `compress`, `concurrent`, `retry`, `ssh`,
`base32`. Third-party types are hidden from public signatures.
_Avoid_: wrapper package, helper package

**SDK-integration package**:
A package whose whole purpose is to make a specific vendor SDK easier to use — `temporal`
and `argo`. Vendor types appear in public signatures deliberately, because the consumer is
writing against that SDK anyway.
_Avoid_: leaky package, thin wrapper

**Selective de-leak**:
The decision to hide a third-party type behind a library-owned one in a utility package
while deliberately keeping it visible in an SDK-integration package. See
[ADR 0004](docs/adr/0004-selective-de-leak-of-third-party-types.md).
_Avoid_: abstraction, encapsulation

**Escape hatch**:
A public method that deliberately returns a third-party type inside an otherwise de-leaked
package, because no library-owned shape would carry the same information. Documented as
such, never an oversight.
_Avoid_: leak, loophole, backdoor

### Conventions

**Convention contract**:
The set of rules every configurable package must satisfy — functional options constructor,
`OTelConfig` field with the non-serialized tags, `WithOTelConfig` option, instrumentation
through `otel.Layers`, and `Example*` tests behind README snippets.
_Avoid_: standard, style guide

**Convention test**:
A test in `internal/archtest` that enforces the convention contract mechanically, by
reflection over registered config structs and by compile-time assignment of each package's
`WithOTelConfig`. Adding a package means extending the registry.
_Avoid_: architecture test, lint rule

**OTel injection point**:
`WithOTelConfig(*otel.Config)` — the single supported way to give a package its telemetry
providers. See [ADR 0002](docs/adr/0002-otel-config-is-injected-never-serialized.md).
_Avoid_: otel setup, telemetry config

**Docs-of-record**:
An `Example*` test that a README snippet is copied from, so documentation cannot drift from
a compiling API. A README code block without one is not trusted.
_Avoid_: sample, snippet test

### Release lines

**v3 line**:
Work on the `next` branch, published as `v3.0.0-next.N` prereleases, merged to `main` as
`v3.0.0`. The only line receiving features. See
[ADR 0001](docs/adr/0001-freeze-v2-and-ship-v3-as-one-big-bang.md).

**Frozen line**:
`release/v2`, pinned at v2.13.1, open to emergency patches only. `release/v1` is closed
entirely at v1.6.0.
_Avoid_: legacy, deprecated, maintenance branch
