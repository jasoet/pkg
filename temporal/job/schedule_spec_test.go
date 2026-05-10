package job

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScheduleSpec_Validate(t *testing.T) {
	t.Run("interval is valid", func(t *testing.T) {
		s := &ScheduleSpec{Interval: time.Hour}
		assert.NoError(t, s.validate())
	})
	t.Run("cron is valid", func(t *testing.T) {
		s := &ScheduleSpec{Cron: "0 0 * * *"}
		assert.NoError(t, s.validate())
	})
	t.Run("calendar is valid", func(t *testing.T) {
		s := &ScheduleSpec{Calendar: []CalendarSpec{{Hour: []ScheduleRange{{Start: 0, End: 23, Step: 1}}}}}
		assert.NoError(t, s.validate())
	})
	t.Run("nothing set is invalid", func(t *testing.T) {
		s := &ScheduleSpec{}
		assert.Error(t, s.validate())
	})
	t.Run("two set is invalid", func(t *testing.T) {
		s := &ScheduleSpec{Interval: time.Hour, Cron: "0 * * * *"}
		assert.Error(t, s.validate())
	})
}

func TestScheduleSpec_ToSDKSpec_Interval(t *testing.T) {
	s := &ScheduleSpec{Interval: 15 * time.Minute}
	sdk, err := s.toSDKSpec()
	require.NoError(t, err)
	require.Len(t, sdk.Intervals, 1)
	assert.Equal(t, 15*time.Minute, sdk.Intervals[0].Every)
}

func TestScheduleSpec_ToSDKSpec_Cron(t *testing.T) {
	s := &ScheduleSpec{Cron: "0 */6 * * *"}
	sdk, err := s.toSDKSpec()
	require.NoError(t, err)
	require.Len(t, sdk.CronExpressions, 1)
	assert.Equal(t, "0 */6 * * *", sdk.CronExpressions[0])
}

func TestScheduleSpec_ToSDKSpec_Calendar(t *testing.T) {
	s := &ScheduleSpec{
		Calendar: []CalendarSpec{{
			Hour:    []ScheduleRange{{Start: 9, End: 17, Step: 1}},
			Minute:  []ScheduleRange{{Start: 0, End: 0, Step: 1}},
			Comment: "business hours",
		}},
	}
	sdk, err := s.toSDKSpec()
	require.NoError(t, err)
	require.Len(t, sdk.Calendars, 1)
	assert.Equal(t, "business hours", sdk.Calendars[0].Comment)
	assert.Len(t, sdk.Calendars[0].Hour, 1)
	assert.Equal(t, 9, sdk.Calendars[0].Hour[0].Start)
	assert.Equal(t, 17, sdk.Calendars[0].Hour[0].End)
}

func TestScheduleSpec_ToSDKSpec_Jitter(t *testing.T) {
	s := &ScheduleSpec{Interval: time.Hour, Jitter: 30 * time.Second}
	sdk, err := s.toSDKSpec()
	require.NoError(t, err)
	assert.Equal(t, 30*time.Second, sdk.Jitter)
}
