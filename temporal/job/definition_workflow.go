package job

import (
	"context"
	"fmt"
	"strings"
	"time"

	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
)

// Describe returns the current state of one workflow run. runID "" = latest.
func (d *Definition) Describe(ctx context.Context, c client.Client, wfID, runID string) (RunDetail, error) {
	resp, err := c.DescribeWorkflowExecution(ctx, wfID, runID)
	if err != nil {
		return RunDetail{}, translateSDKError("describe", err)
	}
	info := resp.GetWorkflowExecutionInfo()
	if info == nil {
		return RunDetail{}, fmt.Errorf("describe: empty info from server")
	}
	det := RunDetail{
		WorkflowID:    info.GetExecution().GetWorkflowId(),
		RunID:         info.GetExecution().GetRunId(),
		Type:          info.GetType().GetName(),
		TaskQueue:     info.GetTaskQueue(),
		Status:        StatusFromSDK(info.GetStatus()),
		StartTime:     info.GetStartTime().AsTime(),
		HistoryLength: info.GetHistoryLength(),
	}
	if info.GetCloseTime() != nil {
		ct := info.GetCloseTime().AsTime()
		det.CloseTime = &ct
		det.ExecutionTime = ct.Sub(det.StartTime)
	} else {
		det.ExecutionTime = time.Since(det.StartTime)
	}
	det.Memo = decodeMemo(info.GetMemo())
	det.SearchAttributes = decodeSearchAttributes(info.GetSearchAttributes())
	return det, nil
}

// History returns the activity-event extraction of one run's history.
func (d *Definition) History(ctx context.Context, c client.Client, wfID, runID string, opts HistoryOpts) (RunHistory, error) {
	limit := opts.MaxEvents
	if limit == 0 {
		limit = 500
	}
	iter := c.GetWorkflowHistory(ctx, wfID, runID, false, enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
	hist := RunHistory{WorkflowID: wfID, RunID: runID}
	scheduled := map[int64]*historypb.HistoryEvent{} // event ID -> scheduled event
	count := 0
	for iter.HasNext() {
		if count >= limit {
			hist.Truncated = true
			break
		}
		ev, err := iter.Next()
		if err != nil {
			return hist, translateSDKError("history", err)
		}
		count++
		switch attrs := ev.Attributes.(type) {
		case *historypb.HistoryEvent_ActivityTaskScheduledEventAttributes:
			scheduled[ev.EventId] = ev
		case *historypb.HistoryEvent_ActivityTaskStartedEventAttributes:
			sched := scheduled[attrs.ActivityTaskStartedEventAttributes.GetScheduledEventId()]
			if sched != nil {
				hist.Activities = append(hist.Activities, buildActivityEventFromStarted(sched, ev))
			}
		case *historypb.HistoryEvent_ActivityTaskCompletedEventAttributes:
			updateLatestRunning(&hist, ev.GetEventTime().AsTime(), ActivityCompleted, payloadToBytes(attrs.ActivityTaskCompletedEventAttributes.GetResult()), "")
		case *historypb.HistoryEvent_ActivityTaskFailedEventAttributes:
			updateLatestRunning(&hist, ev.GetEventTime().AsTime(), ActivityFailed, nil, attrs.ActivityTaskFailedEventAttributes.GetFailure().GetMessage())
		case *historypb.HistoryEvent_ActivityTaskTimedOutEventAttributes:
			updateLatestRunning(&hist, ev.GetEventTime().AsTime(), ActivityTimedOut, nil, attrs.ActivityTaskTimedOutEventAttributes.GetFailure().GetMessage())
		case *historypb.HistoryEvent_ActivityTaskCanceledEventAttributes:
			updateLatestRunning(&hist, ev.GetEventTime().AsTime(), ActivityCanceled, nil, "")
		}
	}
	return hist, nil
}

func buildActivityEventFromStarted(scheduled, started *historypb.HistoryEvent) ActivityEvent {
	a := scheduled.GetActivityTaskScheduledEventAttributes()
	return ActivityEvent{
		Name:      a.GetActivityType().GetName(),
		Status:    ActivityStarted,
		Attempt:   started.GetActivityTaskStartedEventAttributes().GetAttempt(),
		StartTime: started.GetEventTime().AsTime(),
		Input:     payloadToBytes(a.GetInput()),
	}
}

// updateLatestRunning closes out the latest ActivityEvent in hist that is still
// in ActivityStarted state. Temporal emits events in monotonic EventID order so
// the latest still-running activity is the one being closed.
func updateLatestRunning(hist *RunHistory, closeTime time.Time, status ActivityStatus, result []byte, errMsg string) {
	for i := len(hist.Activities) - 1; i >= 0; i-- {
		a := &hist.Activities[i]
		if a.Status == ActivityStarted {
			a.Status = status
			a.CloseTime = closeTime
			a.Duration = closeTime.Sub(a.StartTime)
			a.Result = result
			a.Error = errMsg
			return
		}
	}
}

func payloadToBytes(p *commonpb.Payloads) []byte {
	if p == nil || len(p.GetPayloads()) == 0 {
		return nil
	}
	var out []byte
	for _, pl := range p.GetPayloads() {
		out = append(out, pl.GetData()...)
	}
	return out
}

func decodeMemo(p *commonpb.Memo) map[string]any {
	if p == nil || len(p.GetFields()) == 0 {
		return nil
	}
	out := make(map[string]any, len(p.GetFields()))
	for k, v := range p.GetFields() {
		out[k] = string(v.GetData())
	}
	return out
}

func decodeSearchAttributes(p *commonpb.SearchAttributes) map[string]any {
	if p == nil || len(p.GetIndexedFields()) == 0 {
		return nil
	}
	out := make(map[string]any, len(p.GetIndexedFields()))
	for k, v := range p.GetIndexedFields() {
		out[k] = string(v.GetData())
	}
	return out
}

// Cancel requests cancellation of a workflow run.
func (d *Definition) Cancel(ctx context.Context, c client.Client, wfID, runID string) error {
	return translateSDKError("cancel", c.CancelWorkflow(ctx, wfID, runID))
}

// Terminate hard-stops a workflow run.
func (d *Definition) Terminate(ctx context.Context, c client.Client, wfID, runID, reason string) error {
	return translateSDKError("terminate", c.TerminateWorkflow(ctx, wfID, runID, reason))
}

// Signal sends a signal to a workflow run.
func (d *Definition) Signal(ctx context.Context, c client.Client, wfID, runID, signalName string, payload any) error {
	return translateSDKError("signal", c.SignalWorkflow(ctx, wfID, runID, signalName, payload))
}

// Query invokes a synchronous query against a workflow run.
func (d *Definition) Query(ctx context.Context, c client.Client, wfID, runID, queryType string, args ...any) (any, error) {
	resp, err := c.QueryWorkflow(ctx, wfID, runID, queryType, args...)
	if err != nil {
		return nil, translateSDKError("query", err)
	}
	var out any
	if err := resp.Get(&out); err != nil {
		return nil, translateSDKError("query-decode", err)
	}
	return out, nil
}

// ListRuns lists workflow executions of this Definition, scoped by
// "WorkflowId STARTS_WITH '<Name>-'".
func (d *Definition) ListRuns(ctx context.Context, c client.Client, opts ListOpts) (RunPage, error) {
	pageSize := opts.PageSize
	if pageSize <= 0 {
		pageSize = 100
	}
	if pageSize > 1000 {
		pageSize = 1000
	}
	q := d.scopedQuery(opts.Status, opts.TimeRange)
	resp, err := c.ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
		PageSize:      int32(pageSize),
		NextPageToken: opts.PageToken,
		Query:         q,
	})
	if err != nil {
		return RunPage{}, translateSDKError("list-runs", err)
	}
	page := RunPage{NextPageToken: resp.GetNextPageToken()}
	for _, info := range resp.GetExecutions() {
		s := RunSummary{
			WorkflowID: info.GetExecution().GetWorkflowId(),
			RunID:      info.GetExecution().GetRunId(),
			Type:       info.GetType().GetName(),
			Status:     StatusFromSDK(info.GetStatus()),
			StartTime:  info.GetStartTime().AsTime(),
			TaskQueue:  info.GetTaskQueue(),
		}
		if info.GetCloseTime() != nil {
			ct := info.GetCloseTime().AsTime()
			s.CloseTime = &ct
		}
		page.Runs = append(page.Runs, s)
	}
	return page, nil
}

// Stats returns running/completed-today/failed-today counts scoped to this
// Definition's workflow IDs.
func (d *Definition) Stats(ctx context.Context, c client.Client, opts StatsOpts) (DefinitionStats, error) {
	now := time.Now()
	loc := opts.Location
	if loc == nil {
		loc = time.UTC
	}
	startOfDay := time.Date(now.In(loc).Year(), now.In(loc).Month(), now.In(loc).Day(), 0, 0, 0, 0, loc)
	prefix := fmt.Sprintf("WorkflowId STARTS_WITH %q", d.Name+"-")

	countQ := func(q string) (int64, error) {
		resp, err := c.CountWorkflow(ctx, &workflowservice.CountWorkflowExecutionsRequest{Query: q})
		if err != nil {
			return 0, translateSDKError("count", err)
		}
		return resp.GetCount(), nil
	}
	running, err := countQ(prefix + " AND ExecutionStatus = \"Running\"")
	if err != nil {
		return DefinitionStats{}, err
	}
	completed, err := countQ(prefix + fmt.Sprintf(" AND ExecutionStatus = \"Completed\" AND CloseTime >= %q", startOfDay.Format(time.RFC3339)))
	if err != nil {
		return DefinitionStats{}, err
	}
	failed, err := countQ(prefix + fmt.Sprintf(" AND ExecutionStatus = \"Failed\" AND CloseTime >= %q", startOfDay.Format(time.RFC3339)))
	if err != nil {
		return DefinitionStats{}, err
	}
	return DefinitionStats{Running: running, CompletedToday: completed, FailedToday: failed, AsOf: now}, nil
}

// scopedQuery builds a visibility query that scopes by WorkflowId prefix and
// optionally filters by status / time range.
func (d *Definition) scopedQuery(statuses []Status, tr *TimeRange) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("WorkflowId STARTS_WITH %q", d.Name+"-"))
	if len(statuses) > 0 {
		statusStrs := make([]string, 0, len(statuses))
		for _, s := range statuses {
			statusStrs = append(statusStrs, fmt.Sprintf("%q", titleCase(s.String())))
		}
		parts = append(parts, fmt.Sprintf("ExecutionStatus IN (%s)", strings.Join(statusStrs, ", ")))
	}
	if tr != nil {
		if !tr.Start.IsZero() {
			parts = append(parts, fmt.Sprintf("StartTime >= %q", tr.Start.Format(time.RFC3339)))
		}
		if !tr.End.IsZero() {
			parts = append(parts, fmt.Sprintf("StartTime <= %q", tr.End.Format(time.RFC3339)))
		}
	}
	return strings.Join(parts, " AND ")
}

// titleCase upper-cases the first rune and converts underscore-separated words
// to CamelCase. Used to map status strings like "continued_as_new" to
// "ContinuedAsNew" for Temporal visibility-query status values. ASCII-only
// since status names are fixed and ASCII.
func titleCase(s string) string {
	if s == "" {
		return s
	}
	b := []byte(s)
	var out []byte
	upperNext := true
	for _, c := range b {
		if c == '_' {
			upperNext = true
			continue
		}
		if upperNext && c >= 'a' && c <= 'z' {
			out = append(out, c-('a'-'A'))
			upperNext = false
			continue
		}
		out = append(out, c)
	}
	return string(out)
}
