package auth

import (
	"testing"
	"time"
)

func TestLimiter(t *testing.T) {
	l := NewLimiter(3, time.Minute)
	now := time.Now()
	l.now = func() time.Time { return now }
	k := UserKey("  Alice ")
	if k != UserKey("alice") {
		t.Fatal("keys must be case/space insensitive")
	}
	for i := 0; i < 2; i++ {
		l.Fail(k)
	}
	if l.Blocked(k) {
		t.Fatal("blocked too early")
	}
	l.Fail(k)
	if !l.Blocked(k) || !l.Blocked("other", k) {
		t.Fatal("should be blocked after 3 failures")
	}
	if l.Blocked(UserKey("bob")) {
		t.Fatal("other keys unaffected")
	}
	now = now.Add(61 * time.Second) // the window passes
	if l.Blocked(k) {
		t.Fatal("block must expire")
	}
	l.Fail(k)
	l.Reset(k)
	l.Fail(k)
	l.Fail(k)
	if l.Blocked(k) {
		t.Fatal("reset must clear the counter")
	}
}
