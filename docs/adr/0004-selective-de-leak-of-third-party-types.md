# Third-party types are de-leaked selectively, not universally

Utility packages hide the library they wrap: `rest` returns `*rest.Response` rather than
`*resty.Response`, `config` takes options rather than `*viper.Viper`, and `docker`'s
`WaitStrategy` takes a `docker.ContainerTarget` rather than a `*client.Client`.
SDK-integration packages do the opposite: `temporal` exposes `client.Client` and `argo`
exposes `apiclient.Client` and `v1alpha1.Workflow` in public signatures, by design.

**Why the split:** the test is what the consumer came for. Nobody chooses this library to
get resty or viper — those are implementation details, and leaking them makes the
dependency un-swappable and the godoc unreadable. But consumers of `temporal` and `argo`
*are* writing Temporal and Argo code; wrapping those SDKs would mean shadowing a huge,
fast-moving API surface, and every gap in the wrapper becomes a wall the consumer cannot
climb. The wrapper's cost scales with the wrapped API's size, and the benefit scales with
how incidental the dependency is — those point in opposite directions here.

**Explicit exceptions inside de-leaked packages.** `docker.Executor.Inspect()` and
`GetStats()` still return `docker/docker` types. This is a deliberate escape hatch: the
information has no natural library-owned shape and wrapping it would mean tracking the
docker API for no gain. They are documented as escape hatches — not a de-leak that was
missed.

**Consequences:** `temporal` and `argo` consumers take a direct dependency on those SDKs
and their version churn. That is the honest cost of an integration package and is stated in
their READMEs rather than hidden behind a leaky abstraction. `config.Option` still exposes
viper indirectly through godoc; sealing it behind an interface with an unexported `apply`
is deferred to v3.x rather than rushed into the major.
