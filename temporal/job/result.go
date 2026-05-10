package job

import (
	"context"
	"time"

	"go.temporal.io/sdk/client"
)

// RunHandle is a lightweight handle to one workflow run. Returned by
// Definition.Execute and Definition.GetRun.
type RunHandle struct {
	WorkflowID string
	RunID      string
	raw        client.WorkflowRun
}

// Get blocks until the workflow completes and unmarshals its result into
// valuePtr (must be a non-nil pointer). Returns the workflow's error if it
// failed. Returns nil if the handle has no underlying run (e.g., constructed
// from GetRun on an unknown ID).
func (h RunHandle) Get(ctx context.Context, valuePtr any) error {
	if h.raw == nil {
		return nil
	}
	return h.raw.Get(ctx, valuePtr)
}

// RunDetail is the full description of one workflow run.
type RunDetail struct {
	WorkflowID       string
	RunID            string
	Type             string
	TaskQueue        string
	Status           Status
	StartTime        time.Time
	CloseTime        *time.Time // nil if still running
	ExecutionTime    time.Duration
	HistoryLength    int64
	Memo             map[string]any
	SearchAttributes map[string]any
}

// RunHistory is the activity-event extraction of one run's history, bounded
// by HistoryOpts.MaxEvents.
type RunHistory struct {
	WorkflowID string
	RunID      string
	Activities []ActivityEvent
	Truncated  bool
}

// ActivityEvent describes one activity attempt within a workflow run.
type ActivityEvent struct {
	Name      string
	Status    ActivityStatus
	Attempt   int32
	StartTime time.Time
	CloseTime time.Time
	Duration  time.Duration
	Input     []byte // raw payload; caller deserializes
	Result    []byte // raw payload; nil on failure
	Error     string // empty on success
}

// RunPage is one page of ListRuns results.
type RunPage struct {
	Runs          []RunSummary
	NextPageToken []byte
}

// RunSummary is one row in a list of runs.
type RunSummary struct {
	WorkflowID string
	RunID      string
	Type       string
	Status     Status
	StartTime  time.Time
	CloseTime  *time.Time
	TaskQueue  string
}

// DefinitionStats is per-Definition aggregate counters.
type DefinitionStats struct {
	Running        int64
	CompletedToday int64
	FailedToday    int64
	AsOf           time.Time
}

// ScheduleSummary is a lightweight schedule summary.
type ScheduleSummary struct {
	ID           string
	WorkflowType string
	Paused       bool
	NextRunTime  *time.Time
	LastRunTime  *time.Time
	Note         string
}

// ScheduleDetail is the full schedule description.
type ScheduleDetail struct {
	ScheduleSummary
	Spec       any // wrapped ScheduleSpec; typed in Task 3
	RecentRuns []RunSummary
}

// TaskQueueInfo describes a task queue's pollers and reachability.
type TaskQueueInfo struct {
	Name        string
	WorkerCount int
	// Future: PollerDetails, Reachability, etc.
}
