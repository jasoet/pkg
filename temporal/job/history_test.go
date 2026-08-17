package job

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	failurepb "go.temporal.io/api/failure/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/sdk/mocks"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// fakeHistoryIter is a minimal client.HistoryEventIterator over a fixed slice.
type fakeHistoryIter struct {
	events []*historypb.HistoryEvent
	idx    int
}

func (f *fakeHistoryIter) HasNext() bool { return f.idx < len(f.events) }

func (f *fakeHistoryIter) Next() (*historypb.HistoryEvent, error) {
	ev := f.events[f.idx]
	f.idx++
	return ev, nil
}

func schedEvent(eventID int64, actName string, at time.Time) *historypb.HistoryEvent {
	return &historypb.HistoryEvent{
		EventId:   eventID,
		EventTime: timestamppb.New(at),
		Attributes: &historypb.HistoryEvent_ActivityTaskScheduledEventAttributes{
			ActivityTaskScheduledEventAttributes: &historypb.ActivityTaskScheduledEventAttributes{
				ActivityType: &commonpb.ActivityType{Name: actName},
			},
		},
	}
}

func startEvent(eventID, schedID int64, at time.Time) *historypb.HistoryEvent {
	return &historypb.HistoryEvent{
		EventId:   eventID,
		EventTime: timestamppb.New(at),
		Attributes: &historypb.HistoryEvent_ActivityTaskStartedEventAttributes{
			ActivityTaskStartedEventAttributes: &historypb.ActivityTaskStartedEventAttributes{
				ScheduledEventId: schedID,
				Attempt:          1,
			},
		},
	}
}

func completeEvent(eventID, schedID int64, result string, at time.Time) *historypb.HistoryEvent {
	return &historypb.HistoryEvent{
		EventId:   eventID,
		EventTime: timestamppb.New(at),
		Attributes: &historypb.HistoryEvent_ActivityTaskCompletedEventAttributes{
			ActivityTaskCompletedEventAttributes: &historypb.ActivityTaskCompletedEventAttributes{
				ScheduledEventId: schedID,
				Result:           &commonpb.Payloads{Payloads: []*commonpb.Payload{{Data: []byte(result)}}},
			},
		},
	}
}

func failEvent(eventID, schedID int64, msg string, at time.Time) *historypb.HistoryEvent {
	return &historypb.HistoryEvent{
		EventId:   eventID,
		EventTime: timestamppb.New(at),
		Attributes: &historypb.HistoryEvent_ActivityTaskFailedEventAttributes{
			ActivityTaskFailedEventAttributes: &historypb.ActivityTaskFailedEventAttributes{
				ScheduledEventId: schedID,
				Failure:          &failurepb.Failure{Message: msg},
			},
		},
	}
}

func activityByName(hist RunHistory, name string) (ActivityEvent, bool) {
	for _, a := range hist.Activities {
		if a.Name == name {
			return a, true
		}
	}
	return ActivityEvent{}, false
}

// TestHistory_ConcurrentActivitiesAttribution feeds a synthetic history with two
// parallel activities A and B where A completes before B. The old
// "close the latest still-running activity" logic misattributed A's completion
// to B; matching by ScheduledEventId must attribute each result correctly.
func TestHistory_ConcurrentActivitiesAttribution(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	events := []*historypb.HistoryEvent{
		schedEvent(5, "ActivityA", base),
		schedEvent(6, "ActivityB", base.Add(1*time.Millisecond)),
		startEvent(7, 5, base.Add(2*time.Millisecond)),                 // A started (index 0)
		startEvent(8, 6, base.Add(3*time.Millisecond)),                 // B started (index 1)
		completeEvent(9, 5, "resultA", base.Add(10*time.Millisecond)),  // A completes first
		completeEvent(10, 6, "resultB", base.Add(20*time.Millisecond)), // B completes later
	}

	c := mocks.NewClient(t)
	c.On("GetWorkflowHistory", mock.Anything, "wf", "run", false, mock.Anything).
		Return(&fakeHistoryIter{events: events})

	d := &Definition{Name: "x", TaskQueue: "tq"}
	hist, err := d.History(context.Background(), c, "wf", "run", HistoryOpts{})
	require.NoError(t, err)
	require.Len(t, hist.Activities, 2)

	a, ok := activityByName(hist, "ActivityA")
	require.True(t, ok, "ActivityA must be present")
	assert.Equal(t, ActivityCompleted, a.Status)
	assert.Equal(t, "resultA", string(a.Result), "ActivityA must carry its own result, not B's")

	b, ok := activityByName(hist, "ActivityB")
	require.True(t, ok, "ActivityB must be present")
	assert.Equal(t, ActivityCompleted, b.Status)
	assert.Equal(t, "resultB", string(b.Result), "ActivityB must carry its own result, not A's")
}

// TestHistory_ConcurrentMixedOutcomes verifies a failure and a completion running
// in parallel are attributed to the correct activity by ScheduledEventId.
func TestHistory_ConcurrentMixedOutcomes(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	events := []*historypb.HistoryEvent{
		schedEvent(5, "ActivityA", base),
		schedEvent(6, "ActivityB", base),
		startEvent(7, 5, base.Add(1*time.Millisecond)),
		startEvent(8, 6, base.Add(2*time.Millisecond)),
		failEvent(9, 5, "boom-A", base.Add(5*time.Millisecond)),       // A fails first
		completeEvent(10, 6, "resultB", base.Add(9*time.Millisecond)), // B completes
	}

	c := mocks.NewClient(t)
	c.On("GetWorkflowHistory", mock.Anything, "wf", "run", false, mock.Anything).
		Return(&fakeHistoryIter{events: events})

	d := &Definition{Name: "x", TaskQueue: "tq"}
	hist, err := d.History(context.Background(), c, "wf", "run", HistoryOpts{})
	require.NoError(t, err)
	require.Len(t, hist.Activities, 2)

	a, ok := activityByName(hist, "ActivityA")
	require.True(t, ok)
	assert.Equal(t, ActivityFailed, a.Status)
	assert.Equal(t, "boom-A", a.Error)
	assert.Nil(t, a.Result)

	b, ok := activityByName(hist, "ActivityB")
	require.True(t, ok)
	assert.Equal(t, ActivityCompleted, b.Status)
	assert.Equal(t, "resultB", string(b.Result))
	assert.Empty(t, b.Error)
}

// TestHistory_MaxEventsNoCap verifies MaxEvents == 0 means "no cap": the full
// history is scanned and Truncated stays false.
func TestHistory_MaxEventsNoCap(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	events := []*historypb.HistoryEvent{
		schedEvent(5, "ActivityA", base),
		startEvent(6, 5, base.Add(1*time.Millisecond)),
		completeEvent(7, 5, "resultA", base.Add(2*time.Millisecond)),
	}

	c := mocks.NewClient(t)
	c.On("GetWorkflowHistory", mock.Anything, "wf", "run", false, mock.Anything).
		Return(&fakeHistoryIter{events: events})

	d := &Definition{Name: "x", TaskQueue: "tq"}
	hist, err := d.History(context.Background(), c, "wf", "run", HistoryOpts{MaxEvents: 0})
	require.NoError(t, err)
	assert.False(t, hist.Truncated, "MaxEvents==0 must not truncate")
	require.Len(t, hist.Activities, 1)
	assert.Equal(t, ActivityCompleted, hist.Activities[0].Status)
}

// TestHistory_MaxEventsCapTruncates verifies a positive MaxEvents caps the scan.
func TestHistory_MaxEventsCapTruncates(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	events := []*historypb.HistoryEvent{
		schedEvent(5, "ActivityA", base),
		startEvent(6, 5, base.Add(1*time.Millisecond)),
		completeEvent(7, 5, "resultA", base.Add(2*time.Millisecond)),
	}

	c := mocks.NewClient(t)
	c.On("GetWorkflowHistory", mock.Anything, "wf", "run", false, mock.Anything).
		Return(&fakeHistoryIter{events: events})

	d := &Definition{Name: "x", TaskQueue: "tq"}
	// Cap at 2 events: the Completed event (3rd) is never scanned.
	hist, err := d.History(context.Background(), c, "wf", "run", HistoryOpts{MaxEvents: 2})
	require.NoError(t, err)
	assert.True(t, hist.Truncated, "positive MaxEvents below the event count must truncate")
	require.Len(t, hist.Activities, 1)
	assert.Equal(t, ActivityStarted, hist.Activities[0].Status, "activity stays open when its completion is truncated")
}
