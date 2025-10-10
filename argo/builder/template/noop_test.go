//go:build !integration && !argo

package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNoop(t *testing.T) {
	t.Run("creates basic noop template", func(t *testing.T) {
		noop := NewNoop()
		require.NotNil(t, noop)

		steps, err := noop.Steps()
		require.NoError(t, err)
		require.Len(t, steps, 1)
		assert.Equal(t, "noop", steps[0].Name)
		assert.Equal(t, "noop-template", steps[0].Template)

		templates, err := noop.Templates()
		require.NoError(t, err)
		require.Len(t, templates, 1)
		assert.Equal(t, "noop-template", templates[0].Name)
		assert.NotNil(t, templates[0].Container)
		assert.Equal(t, "alpine:3.19", templates[0].Container.Image)
		assert.Equal(t, []string{"sh", "-c"}, templates[0].Container.Command)
		assert.Equal(t, []string{"echo noop"}, templates[0].Container.Args)
	})
}

func TestNewNoopWithName(t *testing.T) {
	t.Run("creates noop with custom name", func(t *testing.T) {
		noop := NewNoopWithName("custom-noop")
		require.NotNil(t, noop)

		steps, err := noop.Steps()
		require.NoError(t, err)
		require.Len(t, steps, 1)
		assert.Equal(t, "custom-noop", steps[0].Name)
		assert.Equal(t, "custom-noop-template", steps[0].Template)

		templates, err := noop.Templates()
		require.NoError(t, err)
		require.Len(t, templates, 1)
		assert.Equal(t, "custom-noop-template", templates[0].Name)
	})

	t.Run("creates multiple noops with different names", func(t *testing.T) {
		noop1 := NewNoopWithName("placeholder-1")
		noop2 := NewNoopWithName("placeholder-2")

		steps1, err := noop1.Steps()
		require.NoError(t, err)
		steps2, err := noop2.Steps()
		require.NoError(t, err)

		assert.NotEqual(t, steps1[0].Name, steps2[0].Name)
		assert.Equal(t, "placeholder-1", steps1[0].Name)
		assert.Equal(t, "placeholder-2", steps2[0].Name)
	})
}

func TestNoopSteps(t *testing.T) {
	t.Run("returns correct workflow steps", func(t *testing.T) {
		noop := NewNoop()

		steps, err := noop.Steps()
		require.NoError(t, err)
		require.Len(t, steps, 1)
		assert.Equal(t, "noop", steps[0].Name)
		assert.Equal(t, "noop-template", steps[0].Template)
	})
}

func TestNoopTemplates(t *testing.T) {
	t.Run("returns correct templates", func(t *testing.T) {
		noop := NewNoop()

		templates, err := noop.Templates()
		require.NoError(t, err)
		require.Len(t, templates, 1)

		template := templates[0]
		assert.Equal(t, "noop-template", template.Name)
		require.NotNil(t, template.Container)
		assert.Equal(t, "alpine:3.19", template.Container.Image)
		assert.Equal(t, []string{"sh", "-c"}, template.Container.Command)
		assert.Equal(t, []string{"echo noop"}, template.Container.Args)
	})

	t.Run("custom named noop has correct template name", func(t *testing.T) {
		noop := NewNoopWithName("test-placeholder")

		templates, err := noop.Templates()
		require.NoError(t, err)
		require.Len(t, templates, 1)
		assert.Equal(t, "test-placeholder-template", templates[0].Name)
	})
}
