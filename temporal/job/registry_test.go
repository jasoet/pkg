package job

import (
	"context"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func newTestDef(t *testing.T, name string) *Definition {
	t.Helper()
	d, err := New(name, "tq-"+name,
		WithRegister(func(worker.Worker) {}),
		WithExecute(func(context.Context, client.Client, client.StartWorkflowOptions, any) (client.WorkflowRun, error) {
			return nil, nil
		}),
		WithNewInput(func() any { return nil }),
	)
	require.NoError(t, err)
	return d
}

func TestRegistry_AddAndGet(t *testing.T) {
	r := NewRegistry()
	d := newTestDef(t, "alpha")
	require.NoError(t, r.Add(d))
	got, ok := r.Get("alpha")
	assert.True(t, ok)
	assert.Same(t, d, got)
}

func TestRegistry_AddDuplicate(t *testing.T) {
	r := NewRegistry(newTestDef(t, "a"))
	err := r.Add(newTestDef(t, "a"))
	assert.ErrorIs(t, err, ErrDuplicateName)
}

func TestRegistry_AddNilOrInvalid(t *testing.T) {
	r := NewRegistry()
	assert.ErrorIs(t, r.Add(nil), ErrInvalidDefinition)
	assert.ErrorIs(t, r.Add(&Definition{}), ErrInvalidDefinition)
}

func TestRegistry_MustGet_Missing(t *testing.T) {
	r := NewRegistry()
	assert.Panics(t, func() { _ = r.MustGet("missing") })
}

func TestRegistry_List_Sorted(t *testing.T) {
	r := NewRegistry(newTestDef(t, "charlie"), newTestDef(t, "alpha"), newTestDef(t, "bravo"))
	got := r.Names()
	assert.True(t, sort.StringsAreSorted(got))
	assert.Equal(t, []string{"alpha", "bravo", "charlie"}, got)

	list := r.List()
	require.Len(t, list, 3)
	for i, d := range list {
		assert.Equal(t, got[i], d.Name)
	}
}

func TestRegistry_NewWithSeed(t *testing.T) {
	r := NewRegistry(newTestDef(t, "a"), newTestDef(t, "b"))
	assert.Len(t, r.List(), 2)
}
