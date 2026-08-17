# Freeze v2 and ship v3 as one big-bang release

A 15-package audit at v2.13.1 found convention violations, leaked third-party types and
inaccurate docs spread across every package, most of which could only be fixed by breaking
the API. Rather than drip 20+ breaking releases through the v2 line, v2 is frozen at
**v2.13.1** on `release/v2` (emergency patches only) and all the breaks land together in
**v3.0.0**, developed on the `next` branch as `v3.0.0-next.N` prereleases.

**Why this and not incremental deprecation:** the library is a product for external users,
and a stream of majors is worse for them than one migration with a written guide. Go's
module-path versioning makes the big bang cheap to consume — `/v2` and `/v3` can be
imported side by side, so consumers migrate package by package on their own schedule rather
than on ours. That property is what makes the trade-off work; without it the incremental
path would have been correct.

**Consequences:** the migration surface is large and concentrated. `MIGRATION.md` is the
mitigation and is a release blocker, not a nice-to-have — several breaks shipped in
`fix:`-typed commits without `BREAKING CHANGE` footers and will never appear in generated
release notes.
