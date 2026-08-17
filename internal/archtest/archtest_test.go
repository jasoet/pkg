package archtest

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jasoet/pkg/v3/argo"
	"github.com/jasoet/pkg/v3/db"
	"github.com/jasoet/pkg/v3/docker"
	"github.com/jasoet/pkg/v3/otel"
	"github.com/jasoet/pkg/v3/rest"
	"github.com/jasoet/pkg/v3/retry"
	"github.com/jasoet/pkg/v3/server"
	"github.com/jasoet/pkg/v3/ssh"
	"github.com/jasoet/pkg/v3/temporal"
)

// compliantConfigs registers exported config structs that must carry an
// OTelConfig *otel.Config field tagged `yaml:"-" mapstructure:"-"`.
// Add a package here when it is unified onto the v3 conventions.
//
// Sanctioned deviations (packages intentionally absent from this registry):
//   - grpc: its config struct is unexported (grpc/config.go `type config
//     struct`), so it cannot be registered here and is exempt from the
//     config-struct contract. grpc still participates in the WithOTelConfig
//     options contract (see options_test.go).
//
// wantCompliantConfigs is the authoritative expected key set, asserted by
// TestCompliantConfigsRegistryComplete so a dropped or silently-omitted entry
// fails the build instead of vacuously passing the loop below.
var wantCompliantConfigs = []string{
	"argo", "db", "docker", "rest", "retry", "server", "ssh", "temporal",
}

var compliantConfigs = map[string]reflect.Type{
	"argo":     reflect.TypeOf(argo.Config{}),
	"db":       reflect.TypeOf(db.ConnectionConfig{}),
	"docker":   reflect.TypeOf(docker.ContainerRequest{}),
	"rest":     reflect.TypeOf(rest.Config{}),
	"retry":    reflect.TypeOf(retry.Config{}),
	"server":   reflect.TypeOf(server.Config{}),
	"ssh":      reflect.TypeOf(ssh.Config{}),
	"temporal": reflect.TypeOf(temporal.Config{}),
}

// TestCompliantConfigsRegistryComplete guards against the enrollment registry
// being emptied or silently losing entries (e.g. a bad merge), which would make
// TestConfigStructsCarryOTelConfig pass vacuously. It asserts the registry
// contains exactly the expected package set.
func TestCompliantConfigsRegistryComplete(t *testing.T) {
	require.Len(t, compliantConfigs, len(wantCompliantConfigs),
		"compliantConfigs size drifted from the expected package set")
	for _, pkg := range wantCompliantConfigs {
		_, ok := compliantConfigs[pkg]
		assert.Truef(t, ok, "compliantConfigs missing expected package %q", pkg)
	}
}

func TestConfigStructsCarryOTelConfig(t *testing.T) {
	otelPtrType := reflect.TypeOf(&otel.Config{})

	for pkg, typ := range compliantConfigs {
		t.Run(pkg, func(t *testing.T) {
			field, ok := typ.FieldByName("OTelConfig")
			if !ok {
				t.Fatalf("%s: missing OTelConfig field", pkg)
			}
			if field.Type != otelPtrType {
				t.Errorf("%s: OTelConfig is %s, want *otel.Config", pkg, field.Type)
			}
			if got := field.Tag.Get("yaml"); got != "-" {
				t.Errorf("%s: OTelConfig yaml tag = %q, want %q", pkg, got, "-")
			}
			if got := field.Tag.Get("mapstructure"); got != "-" {
				t.Errorf("%s: OTelConfig mapstructure tag = %q, want %q", pkg, got, "-")
			}
		})
	}
}
