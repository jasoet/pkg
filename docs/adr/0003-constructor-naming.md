# Constructor naming: `New` for the primary type, `New<Thing>` when there are several

Packages that construct one main thing export `New` (`docker.New`, `grpc.New`,
`server.New`, `ssh.New`, `retry.New`). Packages that construct several name each one
(`db.NewPool`, `rest.NewClient`, `temporal.NewClient`, `argo.NewClient`,
`otel.NewConfig`, `server.NewConfig`).

**Why record this:** the v3 audit flagged `otel.NewConfig`/`server.NewConfig` versus
`retry.New`/`grpc.New` as an inconsistency to resolve. It is not one. `retry.New` returns a
`Config` because a `Config` handed to `retry.Do` *is* the package's primary artifact —
there is no retry object. `server` exports both because a `Server` is built from a `Config`
and consumers legitimately want each. The apparent split is the rule working, not a
deviation from it.

The alternative — renaming everything to one literal prefix — would have added breaking
changes that make call sites worse, since `New` in a package that builds four different
things says nothing about what you get back.
