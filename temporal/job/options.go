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
	TodayOnly bool           // default false — set true for "running + closed today"
	Location  *time.Location // if nil and TodayOnly: UTC; otherwise this zone's calendar day
}

// HistoryOpts configures Definition.History.
type HistoryOpts struct {
	MaxEvents int // default 500 in the method; 0 = no cap (caller takes responsibility)
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
	searchAttrs map[string]any
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

// WithSearchAttributes attaches search attributes to the workflow execution.
func WithSearchAttributes(sa map[string]any) ExecuteOption {
	return func(c *executeConfig) { c.searchAttrs = sa }
}

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
	if c.searchAttrs != nil {
		opts.SearchAttributes = c.searchAttrs
	}
	return opts
}
