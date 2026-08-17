# `grpc` servers restart, `server` instances do not

`grpc.Server` supports `Start` → `Stop` → `Start` cycles: it rebuilds the underlying
`*grpc.Server` on each start, and `GetGRPCServer()` returns nil in between. `server.Server`
forbids restart — a shutdown instance is spent, and starting it again errors.

**Why they differ:** gRPC servers are commonly restarted in place to swap TLS material or
re-register services, so the rebuild-per-start cost buys something real. An Echo HTTP
server has no equivalent use case, and allowing restart there would mean carrying
resurrection logic whose only exercise is its own tests. Forbidding it makes the spent state
explicit instead of silently half-working.

**Consequences:** two packages in one library answer "can I restart this?" differently.
Recorded here because the natural assumption is that they match, and because a caller
caching a `GetGRPCServer()` reference across a stop will hold a stale pointer.

Relatedly, `server` and the `grpc` gateway both emit metrics named `http.server.*` with
**different attribute sets** — `server` uses method + status, the gateway adds `http.route`
and `active_requests`. Aligning them would mean either dropping the gateway's route
cardinality or inventing route attribution for unmatched Echo requests. Documented rather
than aligned; consumers aggregating across both must account for it.
