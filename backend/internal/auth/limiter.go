package auth

import (
	"strings"
	"sync"
	"time"
)

// Limiter is a small in-memory failure counter used to slow down password guessing.
// A key (username or client address) is blocked for the rest of the window once it has
// accumulated max failures; a successful login clears the username key.
type Limiter struct {
	max    int
	window time.Duration
	now    func() time.Time

	mu       sync.Mutex
	failures map[string][]time.Time
}

func NewLimiter(max int, window time.Duration) *Limiter {
	return &Limiter{max: max, window: window, now: time.Now, failures: map[string][]time.Time{}}
}

func (l *Limiter) recent(key string) []time.Time {
	cutoff := l.now().Add(-l.window)
	list := l.failures[key]
	i := 0
	for i < len(list) && list[i].Before(cutoff) {
		i++
	}
	list = list[i:]
	if len(list) == 0 {
		delete(l.failures, key)
	} else {
		l.failures[key] = list
	}
	return list
}

// Blocked reports whether any key has used up its failures within the window.
func (l *Limiter) Blocked(keys ...string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, k := range keys {
		if len(l.recent(k)) >= l.max {
			return true
		}
	}
	return false
}

func (l *Limiter) Fail(keys ...string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	for _, k := range keys {
		l.failures[k] = append(l.recent(k), now)
	}
}

func (l *Limiter) Reset(keys ...string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, k := range keys {
		delete(l.failures, k)
	}
}

// UserKey and IPKey build limiter keys.
func UserKey(username string) string { return "u:" + strings.ToLower(strings.TrimSpace(username)) }
func IPKey(ip string) string         { return "ip:" + ip }
