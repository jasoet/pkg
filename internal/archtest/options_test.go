package archtest

import (
	"github.com/jasoet/pkg/v3/argo"
	"github.com/jasoet/pkg/v3/db"
	"github.com/jasoet/pkg/v3/docker"
	"github.com/jasoet/pkg/v3/grpc"
	"github.com/jasoet/pkg/v3/otel"
	"github.com/jasoet/pkg/v3/rest"
	"github.com/jasoet/pkg/v3/retry"
	"github.com/jasoet/pkg/v3/server"
	"github.com/jasoet/pkg/v3/ssh"
	"github.com/jasoet/pkg/v3/temporal"
)

// Compile-time contract: each compliant package exposes WithOTelConfig.
// Add a package here when it is unified onto the v3 conventions.
//
// Signature contract: WithOTelConfig takes *otel.Config and returns the
// package's option type.
//
// Sanctioned deviations:
//   - rest's option type is named ClientOption rather than Option. The v3 rest
//     phase completed without renaming it, so ClientOption is the accepted,
//     permanent name for the v3 series (not a pending unification).
var (
	_ func(*otel.Config) argo.Option       = argo.WithOTelConfig
	_ func(*otel.Config) db.Option         = db.WithOTelConfig
	_ func(*otel.Config) docker.Option     = docker.WithOTelConfig
	_ func(*otel.Config) grpc.Option       = grpc.WithOTelConfig
	_ func(*otel.Config) rest.ClientOption = rest.WithOTelConfig
	_ func(*otel.Config) retry.Option      = retry.WithOTelConfig
	_ func(*otel.Config) server.Option     = server.WithOTelConfig
	_ func(*otel.Config) ssh.Option        = ssh.WithOTelConfig
	_ func(*otel.Config) temporal.Option   = temporal.WithOTelConfig
)
