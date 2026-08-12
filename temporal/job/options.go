package job

import (
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
)

// TimeRange filters by a start-time inclusive range.
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// ListOpts configures Definition.ListRuns.
type ListOpts struct {
	Status    []Status   // empty = any
	TimeRange *TimeRange // by StartTime
	PageSize  int        // default 100, max 1000
	PageToken []byte
}

// StatsOpts configures Definition.Stats.
type StatsOpts struct {
	// TodayOnly, when true, restricts the CompletedToday/FailedToday counts to
	// runs that closed on the current calendar day (in Location). When false
	// (the default) those counts cover all completed/failed runs of the
	// Definition, all-time.
	TodayOnly bool
	// Location selects the calendar day for TodayOnly. Nil means UTC. Ignored
	// when TodayOnly is false.
	Location *time.Location
}

// HistoryOpts configures Definition.History.
type HistoryOpts struct {
	// MaxEvents caps how many history events are scanned. Zero or negative
	// means no cap — the full history is iterated (the caller takes
	// responsibility for potentially large histories).
	MaxEvents int
}

// ScheduleListOpts configures Registry.ListSchedules (future) and individual
// schedule paging.
type ScheduleListOpts struct {
	PageSize  int
	PageToken []byte
}

// executeConfig accumulates state across ExecuteOption calls.
type executeConfig struct {
	workflowID  string
	timeout     time.Duration
	taskTimeout time.Duration
	retryPolicy *temporal.RetryPolicy
	memo        map[string]any
}

// ExecuteOption customizes a single Definition.Execute call.
type ExecuteOption func(*executeConfig)

// WithWorkflowID overrides the default ID of "<Name>-<uuid>".
func WithWorkflowID(id string) ExecuteOption {
	return func(c *executeConfig) { c.workflowID = id }
}

// WithTimeout sets WorkflowExecutionTimeout.
func WithTimeout(d time.Duration) ExecuteOption {
	return func(c *executeConfig) { c.timeout = d }
}

// WithTaskTimeout sets WorkflowTaskTimeout.
func WithTaskTimeout(d time.Duration) ExecuteOption {
	return func(c *executeConfig) { c.taskTimeout = d }
}

// WithRetryPolicy sets the workflow-level retry policy.
func WithRetryPolicy(p *temporal.RetryPolicy) ExecuteOption {
	return func(c *executeConfig) { c.retryPolicy = p }
}

// WithMemo attaches a memo to the workflow execution.
func WithMemo(m map[string]any) ExecuteOption {
	return func(c *executeConfig) { c.memo = m }
}

// Note: search attributes are intentionally not exposed here yet. The Temporal
// SDK's legacy map[string]any path is deprecated in favor of TypedSearchAttributes,
// which requires per-attribute schema definitions. Callers that need search
// attributes today can construct their own client.StartWorkflowOptions and
// call client.ExecuteWorkflow directly. A future revision will add typed
// support.

// apply builds a client.StartWorkflowOptions from defaults + accumulated options.
func (c executeConfig) apply(defaultID, taskQueue string) client.StartWorkflowOptions {
	id := c.workflowID
	if id == "" {
		id = defaultID
	}
	opts := client.StartWorkflowOptions{
		ID:        id,
		TaskQueue: taskQueue,
	}
	if c.timeout > 0 {
		opts.WorkflowExecutionTimeout = c.timeout
	}
	if c.taskTimeout > 0 {
		opts.WorkflowTaskTimeout = c.taskTimeout
	}
	if c.retryPolicy != nil {
		opts.RetryPolicy = c.retryPolicy
	}
	if c.memo != nil {
		opts.Memo = c.memo
	}
	return opts
}
