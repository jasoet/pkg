package archtest

import (
	"reflect"
	"testing"

	"github.com/jasoet/pkg/v3/db"
	"github.com/jasoet/pkg/v3/otel"
	"github.com/jasoet/pkg/v3/rest"
	"github.com/jasoet/pkg/v3/retry"
	"github.com/jasoet/pkg/v3/server"
	"github.com/jasoet/pkg/v3/temporal"
)

// compliantConfigs registers exported config structs that must carry an
// OTelConfig *otel.Config field tagged `yaml:"-" mapstructure:"-"`.
// Add a package here when it is unified onto the v3 conventions.
var compliantConfigs = map[string]reflect.Type{
	"db":       reflect.TypeOf(db.ConnectionConfig{}),
	"rest":     reflect.TypeOf(rest.Config{}),
	"retry":    reflect.TypeOf(retry.Config{}),
	"server":   reflect.TypeOf(server.Config{}),
	"temporal": reflect.TypeOf(temporal.Config{}),
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
