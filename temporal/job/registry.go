package job

import (
	"context"
	"fmt"
	"sort"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// Registry maps logical job names to Definitions. Construction does not
// validate seed Definitions twice — they were validated by New already.
type Registry struct {
	defs map[string]*Definition
}

// NewRegistry creates a Registry, optionally seeded. Duplicates among seeds
// silently use the later value (validate input upstream if that matters).
func NewRegistry(defs ...*Definition) *Registry {
	r := &Registry{defs: make(map[string]*Definition, len(defs))}
	for _, d := range defs {
		if d == nil || d.Name == "" {
			continue
		}
		r.defs[d.Name] = d
	}
	return r
}

// Add inserts a Definition. Returns ErrDuplicateName on conflict,
// ErrInvalidDefinition if the Definition is nil or missing required fields.
func (r *Registry) Add(d *Definition) error {
	if d == nil {
		return fmt.Errorf("%w: nil definition", ErrInvalidDefinition)
	}
	if d.Name == "" || d.TaskQueue == "" || d.register == nil || d.execute == nil || d.newInput == nil {
		return fmt.Errorf("%w: definition fields incomplete", ErrInvalidDefinition)
	}
	if _, exists := r.defs[d.Name]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateName, d.Name)
	}
	r.defs[d.Name] = d
	return nil
}

// Get returns the Definition with the given name and a boolean indicating
// whether it was found.
func (r *Registry) Get(name string) (*Definition, bool) {
	d, ok := r.defs[name]
	return d, ok
}

// MustGet returns the Definition with the given name. Panics with
// fmt.Errorf("%w: %s", ErrNotFound, name) if absent.
func (r *Registry) MustGet(name string) *Definition {
	d, ok := r.defs[name]
	if !ok {
		panic(fmt.Errorf("%w: %s", ErrNotFound, name))
	}
	return d
}

// List returns all Definitions, sorted by Name.
func (r *Registry) List() []*Definition {
	names := r.Names()
	out := make([]*Definition, len(names))
	for i, n := range names {
		out[i] = r.defs[n]
	}
	return out
}

// Names returns all registered names, sorted alphabetically.
func (r *Registry) Names() []string {
	out := make([]string, 0, len(r.defs))
	for name := range r.defs {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// RegisterAll registers every Definition on the given worker. Idempotent:
// underlying workflow/activity types are deduplicated via
// RegisterWorkflowOnce / RegisterActivityOnce in builder closures.
func (r *Registry) RegisterAll(w worker.Worker) {
	for _, d := range r.List() {
		d.Register(w)
	}
}

// ApplySchedules creates or updates Temporal schedules for every Definition
// that has one. Returns the first error encountered (does not roll back).
func (r *Registry) ApplySchedules(ctx context.Context, c client.Client) error {
	for _, d := range r.List() {
		if d.Schedule == nil {
			continue
		}
		if err := d.ApplySchedule(ctx, c); err != nil {
			return fmt.Errorf("apply schedule for %q: %w", d.Name, err)
		}
	}
	return nil
}
