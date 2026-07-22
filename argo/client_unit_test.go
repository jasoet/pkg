package argo

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jasoet/pkg/v3/otel"
)

func TestNamespaceTrimsNewline(t *testing.T) {
	namespaceFile := filepath.Join(t.TempDir(), "namespace")
	require.NoError(t, os.WriteFile(namespaceFile, []byte("production\n"), 0o600))

	icc := &inClusterClientConfig{namespaceFile: namespaceFile}

	namespace, overridden, err := icc.Namespace()

	require.NoError(t, err)
	assert.True(t, overridden)
	assert.Equal(t, "production", namespace, "namespace must be trimmed of the trailing newline")
}

// TestArchtestArgo mirrors the internal/archtest registry check for the argo
// package: Config must carry OTelConfig *otel.Config tagged `yaml:"-" mapstructure:"-"`.
func TestArchtestArgo(t *testing.T) {
	field, ok := reflect.TypeOf(Config{}).FieldByName("OTelConfig")
	require.True(t, ok, "argo.Config: missing OTelConfig field")

	assert.Equal(t, reflect.TypeOf(&otel.Config{}), field.Type, "OTelConfig must be *otel.Config")
	assert.Equal(t, "-", field.Tag.Get("yaml"), "OTelConfig yaml tag")
	assert.Equal(t, "-", field.Tag.Get("mapstructure"), "OTelConfig mapstructure tag")
}
