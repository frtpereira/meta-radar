package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNextOccurrence(t *testing.T) {
	tests := []struct {
		name   string
		now    time.Time
		hour   int
		minute int
		want   time.Time
	}{
		{
			name:   "later today",
			now:    time.Date(2026, 3, 10, 4, 0, 0, 0, time.UTC),
			hour:   6,
			minute: 0,
			want:   time.Date(2026, 3, 10, 6, 0, 0, 0, time.UTC),
		},
		{
			name:   "already passed today rolls to tomorrow",
			now:    time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC),
			hour:   6,
			minute: 0,
			want:   time.Date(2026, 3, 11, 6, 0, 0, 0, time.UTC),
		},
		{
			name:   "exactly now rolls to tomorrow, not a zero-length sleep",
			now:    time.Date(2026, 3, 10, 6, 0, 0, 0, time.UTC),
			hour:   6,
			minute: 0,
			want:   time.Date(2026, 3, 11, 6, 0, 0, 0, time.UTC),
		},
		{
			name:   "minute component matters",
			now:    time.Date(2026, 3, 10, 6, 0, 0, 0, time.UTC),
			hour:   6,
			minute: 30,
			want:   time.Date(2026, 3, 10, 6, 30, 0, 0, time.UTC),
		},
		{
			name:   "rolls across a month boundary",
			now:    time.Date(2026, 3, 31, 7, 0, 0, 0, time.UTC),
			hour:   6,
			minute: 0,
			want:   time.Date(2026, 4, 1, 6, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, tt.want.Equal(nextOccurrence(tt.now, tt.hour, tt.minute)))
		})
	}
}

func TestParseHourMinute(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		hour, minute, err := parseHourMinute("06:30")
		require.NoError(t, err)
		assert.Equal(t, 6, hour)
		assert.Equal(t, 30, minute)
	})

	t.Run("invalid", func(t *testing.T) {
		_, _, err := parseHourMinute("not-a-time")
		assert.Error(t, err)
	})
}
