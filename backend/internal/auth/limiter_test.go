package auth

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/store"
)

func openStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(context.Background(), "sqlite:"+filepath.Join(t.TempDir(), "a.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func TestLimiterEscalatesAndPersists(t *testing.T) {
	ctx := context.Background()
	st := openStore(t)
	l := NewLimiter(st)
	l.FailWait = 0
	now := time.Now().Truncate(time.Second)
	l.now = func() time.Time { return now }
	k := UserKey("  Alice ")
	if k != UserKey("alice") {
		t.Fatal("keys must be case/space insensitive")
	}
	for i := 0; i < 7; i++ {
		l.Fail(ctx, k)
	}
	if l.Blocked(ctx, k) != 0 {
		t.Fatal("blocked too early")
	}
	l.Fail(ctx, k) // 8th: 1 min
	if w := l.Blocked(ctx, k); w != time.Minute {
		t.Fatalf("first lock = %v", w)
	}
	// a restart keeps the counter: a new limiter on the same DB
	l2 := NewLimiter(st)
	l2.FailWait = 0
	l2.now = l.now
	if l2.Blocked(ctx, k) == 0 {
		t.Fatal("lock lost across restart")
	}
	l2.Fail(ctx, k) // 9th: 2 min
	if w := l2.Blocked(ctx, k); w != 2*time.Minute {
		t.Fatalf("second lock = %v", w)
	}
	for i := 0; i < 20; i++ {
		l2.Fail(ctx, k)
	}
	if w := l2.Blocked(ctx, k); w != time.Hour {
		t.Fatalf("lock must be capped at 1 h, got %v", w)
	}
	// the lock ends after the cap
	now = now.Add(time.Hour + time.Second)
	if l2.Blocked(ctx, k) != 0 {
		t.Fatal("still locked after the cap")
	}
	// the address counter is separate, and a success only clears the username one
	ip := IPKey("10.0.0.1")
	l2.Fail(ctx, ip)
	l2.Reset(ctx, k)
	if tr, _ := st.GetLoginThrottle(ctx, k); tr.Failures != 0 {
		t.Fatal("reset failed")
	}
	if tr, _ := st.GetLoginThrottle(ctx, ip); tr.Failures != 1 {
		t.Fatal("ip counter must survive a username reset")
	}
	// forgotten after a quiet day
	now = now.Add(25 * time.Hour)
	l2.Fail(ctx, ip)
	if tr, _ := st.GetLoginThrottle(ctx, ip); tr.Failures != 1 {
		t.Fatalf("old failures must be forgotten, got %d", tr.Failures)
	}
}
