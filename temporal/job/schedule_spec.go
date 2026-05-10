package job

import (
	"errors"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

// OverlapPolicy controls how the scheduler handles a new trigger when a
// previous run is still in flight. Values mirror Temporal's ScheduleOverlapPolicy.
type OverlapPolicy int

const (
	OverlapSkip           OverlapPolicy = iota // default — drop new trigger if previous still running
	OverlapBufferOne                           // queue one trigger; drop further
	OverlapBufferAll                           // queue all triggers
	OverlapCancelOther                         // cancel running, start new
	OverlapTerminateOther                      // terminate running, start new
	OverlapAllowAll                            // start in parallel
)

// ToSDK converts the OverlapPolicy to the equivalent Temporal SDK enum value.
func (p OverlapPolicy) ToSDK() enumspb.ScheduleOverlapPolicy {
	switch p {
	case OverlapBufferOne:
		return enumspb.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE
	case OverlapBufferAll:
		return enumspb.SCHEDULE_OVERLAP_POLICY_BUFFER_ALL
	case OverlapCancelOther:
		return enumspb.SCHEDULE_OVERLAP_POLICY_CANCEL_OTHER
	case OverlapTerminateOther:
		return enumspb.SCHEDULE_OVERLAP_POLICY_TERMINATE_OTHER
	case OverlapAllowAll:
		return enumspb.SCHEDULE_OVERLAP_POLICY_ALLOW_ALL
	default:
		return enumspb.SCHEDULE_OVERLAP_POLICY_SKIP
	}
}

// ScheduleRange mirrors Temporal's ScheduleRange (Start/End/Step int).
type ScheduleRange struct {
	Start int
	End   int
	Step  int
}

// CalendarSpec mirrors Temporal's ScheduleCalendarSpec.
type CalendarSpec struct {
	Second     []ScheduleRange
	Minute     []ScheduleRange
	Hour       []ScheduleRange
	DayOfMonth []ScheduleRange
	Month      []ScheduleRange
	Year       []ScheduleRange
	DayOfWeek  []ScheduleRange
	Comment    string
}

// ScheduleSpec describes when a Definition's workflow should fire automatically.
// Exactly one of Interval / Cron / Calendar must be set.
type ScheduleSpec struct {
	Interval time.Duration
	Cron     string
	Calendar []CalendarSpec

	Overlap       OverlapPolicy
	Jitter        time.Duration
	Paused        bool
	Note          string
	CatchupWindow time.Duration
}

func (s *ScheduleSpec) validate() error {
	if s == nil {
		return errors.New("job: ScheduleSpec is nil")
	}
	set := 0
	if s.Interval > 0 {
		set++
	}
	if s.Cron != "" {
		set++
	}
	if len(s.Calendar) > 0 {
		set++
	}
	if set == 0 {
		return errors.New("job: ScheduleSpec requires one of Interval, Cron, or Calendar")
	}
	if set > 1 {
		return errors.New("job: ScheduleSpec must set exactly one of Interval, Cron, or Calendar")
	}
	return nil
}

func (s *ScheduleSpec) toSDKSpec() (client.ScheduleSpec, error) {
	if err := s.validate(); err != nil {
		return client.ScheduleSpec{}, err
	}
	spec := client.ScheduleSpec{
		Jitter: s.Jitter,
	}
	switch {
	case s.Interval > 0:
		spec.Intervals = []client.ScheduleIntervalSpec{{Every: s.Interval}}
	case s.Cron != "":
		spec.CronExpressions = []string{s.Cron}
	case len(s.Calendar) > 0:
		for _, c := range s.Calendar {
			spec.Calendars = append(spec.Calendars, calendarToSDK(c))
		}
	}
	return spec, nil
}

func calendarToSDK(c CalendarSpec) client.ScheduleCalendarSpec {
	return client.ScheduleCalendarSpec{
		Second:     rangesToSDK(c.Second),
		Minute:     rangesToSDK(c.Minute),
		Hour:       rangesToSDK(c.Hour),
		DayOfMonth: rangesToSDK(c.DayOfMonth),
		Month:      rangesToSDK(c.Month),
		Year:       rangesToSDK(c.Year),
		DayOfWeek:  rangesToSDK(c.DayOfWeek),
		Comment:    c.Comment,
	}
}

func rangesToSDK(rs []ScheduleRange) []client.ScheduleRange {
	if len(rs) == 0 {
		return nil
	}
	out := make([]client.ScheduleRange, len(rs))
	for i, r := range rs {
		out[i] = client.ScheduleRange{Start: r.Start, End: r.End, Step: r.Step}
	}
	return out
}
