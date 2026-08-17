package temporal

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/mocks"
)

func newScheduleManagerForTest(t *testing.T) *ScheduleManager {
	t.Helper()
	sm, err := NewScheduleManager(mocks.NewClient(t))
	require.NoError(t, err)
	return sm
}

// TestDeleteSchedules_AllSucceed verifies every tracked schedule is deleted and
// dropped from tracking.
func TestDeleteSchedules_AllSucceed(t *testing.T) {
	sm := newScheduleManagerForTest(t)

	ha := mocks.NewScheduleHandle(t)
	hb := mocks.NewScheduleHandle(t)
	ha.On("Delete", mock.Anything).Return(nil)
	hb.On("Delete", mock.Anything).Return(nil)

	sm.scheduleHandlers = map[string]client.ScheduleHandle{"a": ha, "b": hb}

	require.NoError(t, sm.DeleteSchedules(context.Background()))
	assert.Empty(t, sm.GetScheduleHandlers(), "all schedules must be removed from tracking")
}

// TestDeleteSchedules_NotFoundIsSuccess verifies a NotFound response is treated
// as success (the schedule is already gone) and the entry is dropped so a retry
// converges instead of re-hitting NotFound forever.
func TestDeleteSchedules_NotFoundIsSuccess(t *testing.T) {
	sm := newScheduleManagerForTest(t)

	gone := mocks.NewScheduleHandle(t)
	ok := mocks.NewScheduleHandle(t)
	gone.On("Delete", mock.Anything).Return(serviceerror.NewNotFound("already gone"))
	ok.On("Delete", mock.Anything).Return(nil)

	sm.scheduleHandlers = map[string]client.ScheduleHandle{"gone": gone, "ok": ok}

	require.NoError(t, sm.DeleteSchedules(context.Background()), "NotFound must not fail the batch")
	assert.Empty(t, sm.GetScheduleHandlers(), "NotFound entries must be dropped from tracking")
}

// TestDeleteSchedules_PartialFailureConverges verifies that on a genuine failure
// the failed schedule stays tracked and its error is returned, while successful
// ones are dropped — and a subsequent retry converges to empty.
func TestDeleteSchedules_PartialFailureConverges(t *testing.T) {
	sm := newScheduleManagerForTest(t)

	bad := mocks.NewScheduleHandle(t)
	good := mocks.NewScheduleHandle(t)
	// First attempt fails for "bad", succeeds for "good".
	bad.On("Delete", mock.Anything).Return(errors.New("transient boom")).Once()
	good.On("Delete", mock.Anything).Return(nil).Once()

	sm.scheduleHandlers = map[string]client.ScheduleHandle{"bad": bad, "good": good}

	err := sm.DeleteSchedules(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transient boom")

	remaining := sm.GetScheduleHandlers()
	assert.Len(t, remaining, 1, "only the failed schedule must remain")
	assert.Contains(t, remaining, "bad")
	assert.NotContains(t, remaining, "good", "the successfully-deleted schedule must be dropped")

	// Retry: "bad" now deletes cleanly, so the manager converges to empty.
	bad.On("Delete", mock.Anything).Return(nil).Once()
	require.NoError(t, sm.DeleteSchedules(context.Background()))
	assert.Empty(t, sm.GetScheduleHandlers(), "retry must converge to empty")
}

// TestDeleteSchedules_Empty is a no-op success.
func TestDeleteSchedules_Empty(t *testing.T) {
	sm := newScheduleManagerForTest(t)
	require.NoError(t, sm.DeleteSchedules(context.Background()))
}
