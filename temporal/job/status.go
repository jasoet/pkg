package job

import (
	enumspb "go.temporal.io/api/enums/v1"
)

// Status represents a workflow execution status, mirrored from Temporal's
// WorkflowExecutionStatus enum for use without leaking SDK enum types.
type Status int

const (
	StatusUnknown Status = iota
	StatusRunning
	StatusCompleted
	StatusFailed
	StatusCanceled
	StatusTerminated
	StatusContinuedAsNew
	StatusTimedOut
)

// String returns the lowercase snake_case name of the status.
func (s Status) String() string {
	switch s {
	case StatusRunning:
		return "running"
	case StatusCompleted:
		return "completed"
	case StatusFailed:
		return "failed"
	case StatusCanceled:
		return "canceled"
	case StatusTerminated:
		return "terminated"
	case StatusContinuedAsNew:
		return "continued_as_new"
	case StatusTimedOut:
		return "timed_out"
	default:
		return "unknown"
	}
}

// IsTerminal reports whether the status represents a closed (finished) workflow.
func (s Status) IsTerminal() bool {
	switch s {
	case StatusCompleted, StatusFailed, StatusCanceled, StatusTerminated, StatusContinuedAsNew, StatusTimedOut:
		return true
	default:
		return false
	}
}

// StatusFromSDK maps a Temporal SDK WorkflowExecutionStatus to a job.Status.
func StatusFromSDK(s enumspb.WorkflowExecutionStatus) Status {
	switch s {
	case enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING:
		return StatusRunning
	case enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED:
		return StatusCompleted
	case enumspb.WORKFLOW_EXECUTION_STATUS_FAILED:
		return StatusFailed
	case enumspb.WORKFLOW_EXECUTION_STATUS_CANCELED:
		return StatusCanceled
	case enumspb.WORKFLOW_EXECUTION_STATUS_TERMINATED:
		return StatusTerminated
	case enumspb.WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW:
		return StatusContinuedAsNew
	case enumspb.WORKFLOW_EXECUTION_STATUS_TIMED_OUT:
		return StatusTimedOut
	default:
		return StatusUnknown
	}
}

// ActivityStatus mirrors Temporal's per-activity outcome.
type ActivityStatus int

const (
	ActivityScheduled ActivityStatus = iota
	ActivityStarted
	ActivityCompleted
	ActivityFailed
	ActivityTimedOut
	ActivityCanceled
)

func (s ActivityStatus) String() string {
	switch s {
	case ActivityScheduled:
		return "scheduled"
	case ActivityStarted:
		return "started"
	case ActivityCompleted:
		return "completed"
	case ActivityFailed:
		return "failed"
	case ActivityTimedOut:
		return "timed_out"
	case ActivityCanceled:
		return "canceled"
	default:
		return "unknown"
	}
}
