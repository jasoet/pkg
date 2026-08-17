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
	// ScheduleArgs are the positional arguments passed to the workflow each time
	// the schedule fires. Required for scheduled workflows that take input;
	// leave empty for workflows with no parameters. Set via WithScheduleArgs.
	ScheduleArgs []any

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

// WithScheduleArgs sets the positional arguments passed to the workflow on each
// scheduled run. Use this for scheduled workflows that require input; without
// it the scheduled action carries no arguments and such workflows fail to
// decode their parameters every run.
func WithScheduleArgs(args ...any) Option {
	return func(d *Definition) { d.ScheduleArgs = args }
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

// Registrar deduplicates workflow/activity registrations for a single worker,
// preventing the SDK's "already registered" panic when the same type is shared
// across multiple Definitions. It holds no package-global state: once the
// Registrar (and its worker) go out of scope everything it tracks is reclaimed
// by the GC, so churned workers do not leak. Prefer a Registrar over the
// package-level RegisterWorkflowOnce / RegisterActivityOnce helpers when
// workers are created and discarded frequently.
type Registrar struct {
	w    worker.Worker
	mu   sync.Mutex
	seen map[string]struct{}
}

// NewRegistrar returns a Registrar bound to w.
func NewRegistrar(w worker.Worker) *Registrar {
	return &Registrar{w: w, seen: make(map[string]struct{})}
}

// RegisterWorkflowOnce registers a workflow on the Registrar's worker, returning
// silently if a workflow with the same typeName has already been registered on it.
func (r *Registrar) RegisterWorkflowOnce(typeName string, wf any, opts workflow.RegisterOptions) {
	if r.markSeen("wf:" + typeName) {
		return
	}
	r.w.RegisterWorkflowWithOptions(wf, opts)
}

// RegisterActivityOnce registers an activity on the Registrar's worker idempotently.
// Activity name comes from typeName, falling back to opts.Name then the fn type.
func (r *Registrar) RegisterActivityOnce(typeName string, fn any, opts activity.RegisterOptions) {
	typeName = activityTypeName(typeName, fn, opts)
	if r.markSeen("act:" + typeName) {
		return
	}
	r.w.RegisterActivityWithOptions(fn, opts)
}

// markSeen records key and reports whether it was already present.
func (r *Registrar) markSeen(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.seen[key]; ok {
		return true
	}
	r.seen[key] = struct{}{}
	return false
}

func activityTypeName(typeName string, fn any, opts activity.RegisterOptions) string {
	if typeName == "" {
		typeName = opts.Name
	}
	if typeName == "" {
		// Fallback: this should not happen in this codebase but provides safety.
		typeName = fmt.Sprintf("%T", fn)
	}
	return typeName
}

// registrars caches one Registrar per worker for the package-level helpers.
// Unlike a map keyed by (worker, typeName), a single entry per worker can be
// released via ForgetWorker when the worker is discarded.
var registrars sync.Map // worker.Worker -> *Registrar

func registrarFor(w worker.Worker) *Registrar {
	if r, ok := registrars.Load(w); ok {
		return r.(*Registrar)
	}
	r, _ := registrars.LoadOrStore(w, NewRegistrar(w))
	return r.(*Registrar)
}

// RegisterWorkflowOnce registers a workflow on a worker, returning silently
// if the (worker, typeName) pair has already been registered. Used by
// builder packages to make their RegisterAll-style helpers idempotent.
//
// The package retains one Registrar per worker; call ForgetWorker when a worker
// is discarded, or use a Registrar directly to avoid the global entirely.
func RegisterWorkflowOnce(w worker.Worker, typeName string, wf any, opts workflow.RegisterOptions) {
	registrarFor(w).RegisterWorkflowOnce(typeName, wf, opts)
}

// RegisterActivityOnce registers an activity on a worker idempotently.
// Activity name comes from opts.Name; pass empty Name only for typed-function
// activities (rare in this codebase).
func RegisterActivityOnce(w worker.Worker, typeName string, fn any, opts activity.RegisterOptions) {
	registrarFor(w).RegisterActivityOnce(typeName, fn, opts)
}

// ForgetWorker drops the internal dedup tracking for w. Call it when a worker
// is discarded so the package does not retain the worker (and its tracked type
// names) indefinitely. It is a no-op when using a Registrar directly.
func ForgetWorker(w worker.Worker) {
	registrars.Delete(w)
}
