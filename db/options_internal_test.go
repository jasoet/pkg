package db

import (
	"testing"

	"github.com/stretchr/testify/assert"

	pkgotel "github.com/jasoet/pkg/v3/otel"
)

func TestWithConnectionConfig_AppliesAllFields(t *testing.T) {
	src := ConnectionConfig{
		DBType:       Postgresql,
		Host:         "localhost",
		Port:         5432,
		Username:     "user",
		Password:     "secret",
		DBName:       "mydb",
		MaxIdleConns: 5,
		MaxOpenConns: 10,
		OTelConfig:   pkgotel.NewConfig("original"),
	}

	got := ConnectionConfig{}
	WithConnectionConfig(src)(&got)

	assert.Equal(t, src, got)
}

func TestWithOTelConfig_Overrides(t *testing.T) {
	custom := pkgotel.NewConfig("custom")
	got := ConnectionConfig{OTelConfig: pkgotel.NewConfig("original")}

	WithOTelConfig(custom)(&got)

	assert.Same(t, custom, got.OTelConfig)
	assert.Equal(t, "custom", got.OTelConfig.ServiceName)
}

func TestWithOTelConfig_NilKeepsExisting(t *testing.T) {
	original := pkgotel.NewConfig("original")
	got := ConnectionConfig{OTelConfig: original}

	WithOTelConfig(nil)(&got)

	assert.Same(t, original, got.OTelConfig)
}
