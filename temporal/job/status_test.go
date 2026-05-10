package job

import (
	"testing"

	"github.com/stretchr/testify/assert"
	enumspb "go.temporal.io/api/enums/v1"
)

func TestStatus_String(t *testing.T) {
	cases := map[Status]string{
		StatusUnknown:        "unknown",
		StatusRunning:        "running",
		StatusCompleted:      "completed",
		StatusFailed:         "failed",
		StatusCanceled:       "canceled",
		StatusTerminated:     "terminated",
		StatusContinuedAsNew: "continued_as_new",
		StatusTimedOut:       "timed_out",
	}
	for s, want := range cases {
		assert.Equal(t, want, s.String(), "Status(%d).String()", s)
	}
}

func TestStatus_IsTerminal(t *testing.T) {
	terminal := []Status{StatusCompleted, StatusFailed, StatusCanceled, StatusTerminated, StatusContinuedAsNew, StatusTimedOut}
	for _, s := range terminal {
		assert.True(t, s.IsTerminal(), "%s should be terminal", s)
	}
	assert.False(t, StatusRunning.IsTerminal())
	assert.False(t, StatusUnknown.IsTerminal())
}

func TestStatusFromSDK(t *testing.T) {
	cases := map[enumspb.WorkflowExecutionStatus]Status{
		enumspb.WORKFLOW_EXECUTION_STATUS_UNSPECIFIED:      StatusUnknown,
		enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING:          StatusRunning,
		enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED:        StatusCompleted,
		enumspb.WORKFLOW_EXECUTION_STATUS_FAILED:           StatusFailed,
		enumspb.WORKFLOW_EXECUTION_STATUS_CANCELED:         StatusCanceled,
		enumspb.WORKFLOW_EXECUTION_STATUS_TERMINATED:       StatusTerminated,
		enumspb.WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW: StatusContinuedAsNew,
		enumspb.WORKFLOW_EXECUTION_STATUS_TIMED_OUT:        StatusTimedOut,
	}
	for sdk, want := range cases {
		assert.Equal(t, want, StatusFromSDK(sdk), "sdk=%v", sdk)
	}
}
