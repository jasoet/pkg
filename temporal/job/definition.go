package job

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// Definition is a type-focused description of one registered Temporal workflow.
// All per-job operations hang off the type as methods.
type Definition struct {
	Name        string
	TaskQueue   string
	Description string
	Tags        []string
	Schedule    *ScheduleSpec

	// Private wiring set only by New via Option closures.
	register func(worker.Worker)
	execute  func(ctx context.Context, c client.Client, opts client.StartWorkflowOptions, input any) (client.WorkflowRun, error)
	newInput func() any
}

// Option configures a Definition during construction.
type Option func(*Definition)

// WithRegister sets the worker-registration closure.
func WithRegister(fn func(worker.Worker)) Option {
	return func(d *Definition) { d.register = fn }
}

// WithExecute sets the workflow-execution closure. The closure receives a
// pre-built client.StartWorkflowOptions (ID + TaskQueue + caller overrides)
// and the typed input value.
func WithExecute(fn func(ctx context.Context, c client.Client, opts client.StartWorkflowOptions, input any) (client.WorkflowRun, error)) Option {
	return func(d *Definition) { d.execute = fn }
}

// WithNewInput sets the factory that returns a typed zero value of the
// workflow input. Callers fill the value before calling Execute.
func WithNewInput(fn func() any) Option {
	return func(d *Definition) { d.newInput = fn }
}

// WithSchedule attaches an optional schedule specification.
func WithSchedule(spec *ScheduleSpec) Option {
	return func(d *Definition) { d.Schedule = spec }
}

// WithDescription attaches a human-readable description.
func WithDescription(desc string) Option {
	return func(d *Definition) { d.Description = desc }
}

// WithTags attaches user-defined tags.
func WithTags(tags ...string) Option {
	return func(d *Definition) { d.Tags = tags }
}

// New constructs a Definition. Validates name, task queue, all closures, and
// the optional schedule. Returns ErrInvalidDefinition if anything is missing
// or inconsistent.
func New(name, taskQueue string, opts ...Option) (*Definition, error) {
	if name == "" {
		return nil, fmt.Errorf("%w: name required", ErrInvalidDefinition)
	}
	if taskQueue == "" {
		return nil, fmt.Errorf("%w: task queue required", ErrInvalidDefinition)
	}
	d := &Definition{Name: name, TaskQueue: taskQueue}
	for _, opt := range opts {
		opt(d)
	}
	if d.register == nil || d.execute == nil || d.newInput == nil {
		return nil, fmt.Errorf("%w: WithRegister, WithExecute, and WithNewInput are all required", ErrInvalidDefinition)
	}
	if d.Schedule != nil {
		if err := d.Schedule.validate(); err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidDefinition, err)
		}
	}
	return d, nil
}

// NewInput returns a fresh typed zero value for this Definition's workflow
// input. Callers fill it before calling Execute (e.g., via json.Unmarshal).
func (d *Definition) NewInput() any {
	return d.newInput()
}

// Register wires the workflow and its activities onto a worker. Safe to call
// concurrently and multiple times — the builder-supplied register closure is
// expected to use RegisterWorkflowOnce / RegisterActivityOnce for idempotency
// when the underlying workflow type may be shared across Definitions.
func (d *Definition) Register(w worker.Worker) {
	if d.register == nil {
		return
	}
	d.register(w)
}

// Execute starts a workflow run. The workflow ID defaults to "<Name>-<uuid>"
// unless overridden via WithWorkflowID(...).
func (d *Definition) Execute(ctx context.Context, c client.Client, input any, opts ...ExecuteOption) (RunHandle, error) {
	if d.execute == nil {
		return RunHandle{}, ErrNotRegistered
	}
	var cfg executeConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	defaultID := d.Name + "-" + uuid.NewString()
	sdkOpts := cfg.apply(defaultID, d.TaskQueue)
	run, err := d.execute(ctx, c, sdkOpts, input)
	if err != nil {
		return RunHandle{}, translateSDKError("execute", err)
	}
	return RunHandle{
		WorkflowID: run.GetID(),
		RunID:      run.GetRunID(),
		raw:        run,
	}, nil
}

// GetRun returns a RunHandle for an existing workflow run identified by
// wfID and runID (runID "" = latest). Useful when reattaching to a run
// triggered elsewhere.
func (d *Definition) GetRun(c client.Client, wfID, runID string) RunHandle {
	if c == nil {
		return RunHandle{WorkflowID: wfID, RunID: runID}
	}
	run := c.GetWorkflow(context.Background(), wfID, runID)
	return RunHandle{WorkflowID: wfID, RunID: runID, raw: run}
}

// --- Dedup helpers used by builders' Register closures ---

type registrarKey struct {
	worker   worker.Worker
	typeName string
}

var (
	registeredWorkflows  sync.Map
	registeredActivities sync.Map
)

// RegisterWorkflowOnce registers a workflow on a worker, returning silently
// if the (worker, typeName) pair has already been registered. Used by
// builder packages to make their RegisterAll-style helpers idempotent.
func RegisterWorkflowOnce(w worker.Worker, typeName string, wf any, opts workflow.RegisterOptions) {
	key := registrarKey{w, typeName}
	if _, loaded := registeredWorkflows.LoadOrStore(key, struct{}{}); loaded {
		return
	}
	w.RegisterWorkflowWithOptions(wf, opts)
}

// RegisterActivityOnce registers an activity on a worker idempotently.
// Activity name comes from opts.Name; pass empty Name only for typed-function
// activities (rare in this codebase).
func RegisterActivityOnce(w worker.Worker, typeName string, fn any, opts activity.RegisterOptions) {
	if typeName == "" {
		typeName = opts.Name
	}
	if typeName == "" {
		// Fallback: this should not happen in this codebase but provides safety.
		typeName = fmt.Sprintf("%T", fn)
	}
	key := registrarKey{w, typeName}
	if _, loaded := registeredActivities.LoadOrStore(key, struct{}{}); loaded {
		return
	}
	w.RegisterActivityWithOptions(fn, opts)
}
