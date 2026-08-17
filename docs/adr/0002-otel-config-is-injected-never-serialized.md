# OTel configuration is injected, never serialized

Every package config struct carries `OTelConfig *otel.Config` tagged exactly
`yaml:"-" mapstructure:"-"`, and `WithOTelConfig(*otel.Config)` is the single injection
point. Telemetry is wired in code; it is never loaded from a config file.

**Why:** `otel.Config` holds live provider handles (tracer, meter, logger). Those cannot be
meaningfully expressed in YAML, and a partially-deserialized one is worse than none — it
produces a config that looks configured and silently exports nothing. Excluding it from
serialization makes that state unrepresentable.

**Consequences:** consumers cannot enable telemetry purely through configuration; they must
construct providers and pass them. Accepted deliberately. The tags are enforced by
`internal/archtest` so the convention cannot drift back one package at a time — the failure
mode being prevented is silent, so a mechanical check is worth more than a documented rule.
