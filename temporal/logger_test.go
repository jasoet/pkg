package temporal

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newBufferedAdapter(buf *bytes.Buffer) *ZerologAdapter {
	return NewZerologAdapter(zerolog.New(buf))
}

func parseLogLine(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry), "log output must be a single JSON object")
	return entry
}

func TestZerologAdapterLevels(t *testing.T) {
	tests := []struct {
		name      string
		log       func(z *ZerologAdapter, msg string, keyvals ...any)
		wantLevel string
	}{
		{name: "Debug", log: (*ZerologAdapter).Debug, wantLevel: "debug"},
		{name: "Info", log: (*ZerologAdapter).Info, wantLevel: "info"},
		{name: "Warn", log: (*ZerologAdapter).Warn, wantLevel: "warn"},
		{name: "Error", log: (*ZerologAdapter).Error, wantLevel: "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			adapter := newBufferedAdapter(&buf)

			tt.log(adapter, "test message", "workflow_id", "wf-123", "attempt", 3)

			entry := parseLogLine(t, &buf)
			assert.Equal(t, tt.wantLevel, entry["level"])
			assert.Equal(t, "test message", entry["message"])
			assert.Equal(t, "wf-123", entry["workflow_id"])
			assert.InDelta(t, 3, entry["attempt"], 0)
		})
	}
}

func TestZerologAdapterOddKeyvals(t *testing.T) {
	var buf bytes.Buffer
	adapter := newBufferedAdapter(&buf)

	adapter.Info("odd keyvals", "key1", "value1", "dangling")

	entry := parseLogLine(t, &buf)
	assert.Equal(t, "value1", entry["key1"])
	assert.Equal(t, "dangling", entry["unknown"])
}

func TestZerologAdapterNonStringKey(t *testing.T) {
	var buf bytes.Buffer
	adapter := newBufferedAdapter(&buf)

	adapter.Info("non-string key", 42, "value")

	entry := parseLogLine(t, &buf)
	assert.Equal(t, "value", entry["unknown"])
}

func TestZerologAdapterWith(t *testing.T) {
	var buf bytes.Buffer
	adapter := newBufferedAdapter(&buf)

	child := adapter.With("namespace", "test-ns", "dangling")

	childLogger, ok := child.(*ZerologAdapter)
	require.True(t, ok, "With must return a *ZerologAdapter")
	childLogger.Info("with context", "extra", "field")

	entry := parseLogLine(t, &buf)
	assert.Equal(t, "test-ns", entry["namespace"])
	assert.Equal(t, "dangling", entry["unknown"])
	assert.Equal(t, "field", entry["extra"])
	assert.Equal(t, "with context", entry["message"])
}

func TestZerologAdapterWithCallerSkip(t *testing.T) {
	var buf bytes.Buffer
	adapter := newBufferedAdapter(&buf)

	child := adapter.WithCallerSkip(0)
	require.NotNil(t, child)

	child.Info("caller message")

	entry := parseLogLine(t, &buf)
	assert.Equal(t, "caller message", entry["message"])
	assert.Contains(t, entry, "caller")
}
