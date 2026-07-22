package archtest

import (
	"github.com/jasoet/pkg/v2/docker"
	"github.com/jasoet/pkg/v2/grpc"
	"github.com/jasoet/pkg/v2/rest"
	"github.com/jasoet/pkg/v2/server"
)

// Compile-time contract: each compliant package exposes WithOTelConfig.
// Add a package here when it is unified onto the v3 conventions.
var (
	_ = docker.WithOTelConfig
	_ = grpc.WithOTelConfig
	_ = rest.WithOTelConfig
	_ = server.WithOTelConfig
)
