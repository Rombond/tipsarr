package auth

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/store"
)

// Limiter slows down password guessing with failure counters kept in the database (they survive a restart).
// Once a key (username or client address) reaches its threshold, every further failure locks it for an
// escalating time: Base, then twice as long each failure, never more than Cap, so nobody (the admin
// included) can be locked out for longer than Cap by someone guessing their password. A counter is
// forgotten after Forget without failures; a successful sign-in clears the username counter only.
type Limiter struct {
	store    *store.Store
	UserMax  int           // failures per username before lockouts start
	IPMax    int           // failures per client address (higher: several people can share one address)
	Base     time.Duration // first lockout
	Cap      time.Duration // longest lockout
	Forget   time.Duration // a counter idle this long starts again from zero
	FailWait time.Duration // constant delay added to every failed sign-in
	now      func() time.Time
}

// NewLimiter: 8 failures per username (30 per address), then 1 min doubling up to 1 h, ~1 s per failure.
func NewLimiter(st *store.Store) *Limiter {
	return &Limiter{store: st, UserMax: 8, IPMax: 30, Base: time.Minute, Cap: time.Hour, Forget: 24 * time.Hour, FailWait: time.Second, now: time.Now}
}

func (l *Limiter) max(key string) int {
	if strings.HasPrefix(key, "ip:") {
		return l.IPMax
	}
	return l.UserMax
}

// Blocked returns how long the most locked key stays locked (0 = may try).
func (l *Limiter) Blocked(ctx context.Context, keys ...string) time.Duration {
	var wait time.Duration
	now := l.now()
	for _, k := range keys {
		t, err := l.store.GetLoginThrottle(ctx, k)
		if err != nil {
			slog.Warn("login limiter", "err", err)
			continue
		}
		if until := time.Unix(t.LockedUntil, 0); until.After(now) && until.Sub(now) > wait {
			wait = until.Sub(now)
		}
	}
	return wait
}

// Fail counts a failed sign-in for every key, locks the keys over their threshold, then waits FailWait.
func (l *Limiter) Fail(ctx context.Context, keys ...string) {
	now := l.now()
	for _, k := range keys {
		t, err := l.store.GetLoginThrottle(ctx, k)
		if err != nil {
			slog.Warn("login limiter", "err", err)
			continue
		}
		if now.Unix()-t.UpdatedAt > int64(l.Forget.Seconds()) {
			t.Failures, t.LockedUntil = 0, 0
		}
		t.Failures++
		t.UpdatedAt = now.Unix()
		if over := t.Failures - l.max(k); over >= 0 {
			lock := l.Base
			for i := 0; i < over && lock < l.Cap; i++ {
				lock *= 2
			}
			lock = min(lock, l.Cap)
			t.LockedUntil = now.Add(lock).Unix()
			slog.Warn("sign-in locked after repeated failures", "key", k, "failures", t.Failures, "for", lock.String())
		}
		if err := l.store.SaveLoginThrottle(ctx, t); err != nil {
			slog.Warn("login limiter", "err", err)
		}
	}
	if l.FailWait > 0 {
		select {
		case <-time.After(l.FailWait):
		case <-ctx.Done():
		}
	}
}

// Reset clears the counters of the keys (a successful sign-in clears its username key).
func (l *Limiter) Reset(ctx context.Context, keys ...string) {
	for _, k := range keys {
		if err := l.store.DeleteLoginThrottle(ctx, k); err != nil {
			slog.Warn("login limiter", "err", err)
		}
	}
}

// UserKey and IPKey build limiter keys.
func UserKey(username string) string { return "u:" + strings.ToLower(strings.TrimSpace(username)) }
func IPKey(ip string) string         { return "ip:" + ip }
