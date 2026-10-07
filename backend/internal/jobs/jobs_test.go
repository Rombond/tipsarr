package jobs

import (
	"testing"
	"time"
)

func TestNextWait(t *testing.T) {
	cases := []struct {
		every    time.Duration
		failures int
		want     time.Duration
	}{
		{6 * time.Hour, 0, 6 * time.Hour},
		{6 * time.Hour, 1, 30 * time.Second},
		{6 * time.Hour, 2, time.Minute},
		{6 * time.Hour, 3, 2 * time.Minute},
		{6 * time.Hour, 9, 10 * time.Minute},    // capped
		{15 * time.Second, 4, 15 * time.Second}, // never longer than the normal interval
	}
	for _, c := range cases {
		if got := nextWait(c.every, c.failures); got != c.want {
			t.Errorf("nextWait(%v, %d) = %v, want %v", c.every, c.failures, got, c.want)
		}
	}
}
