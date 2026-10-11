package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"time"
)

var keyRe = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

const (
	stQueued  = "queued"
	stRunning = "running"
	stReady   = "ready"
	stFailed  = "failed"
)

// Entry is what is known about one YouTube key. It is saved next to the video as <key>.json, so a restart keeps it.
type Entry struct {
	Key       string `json:"key"`
	Status    string `json:"status"`
	Bytes     int64  `json:"bytes,omitempty"`
	Height    int    `json:"height,omitempty"`
	Duration  int    `json:"duration,omitempty"`
	Error     string `json:"error,omitempty"`
	Permanent bool   `json:"permanent,omitempty"`
	Attempts  int    `json:"attempts,omitempty"`
	MaxHeight int    `json:"maxHeight,omitempty"`
	ReadyAt   int64  `json:"readyAt,omitempty"`
	LastUsed  int64  `json:"lastUsed,omitempty"`
	RetryAt   int64  `json:"retryAt,omitempty"` // temporary failures wait until then
}

type Store struct {
	dir string
	now func() time.Time

	mu      sync.Mutex
	entries map[string]*Entry
}

func NewStore(dir string, now func() time.Time) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &Store{dir: dir, now: now, entries: map[string]*Entry{}}
	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var e Entry
		if json.Unmarshal(b, &e) != nil || !keyRe.MatchString(e.Key) {
			continue
		}
		if e.Status == stRunning {
			e.Status = stQueued // the process died while it was running
		}
		if e.Status == stReady {
			if st, err := os.Stat(s.VideoPath(e.Key)); err != nil || st.Size() == 0 {
				e.Status = stQueued
			}
		}
		s.entries[e.Key] = &e
	}
	return s, nil
}

func (s *Store) VideoPath(key string) string { return filepath.Join(s.dir, key+".mp4") }
func (s *Store) metaPath(key string) string  { return filepath.Join(s.dir, key+".json") }

// Get returns a copy.
func (s *Store) Get(key string) (Entry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[key]
	if !ok {
		return Entry{}, false
	}
	return *e, true
}

// Update changes (or creates) an entry and saves it.
func (s *Store) Update(key string, f func(e *Entry)) Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[key]
	if !ok {
		e = &Entry{Key: key, Status: stQueued}
		s.entries[key] = e
	}
	f(e)
	s.save(e)
	return *e
}

func (s *Store) save(e *Entry) {
	b, _ := json.Marshal(e)
	tmp := s.metaPath(e.Key) + ".tmp"
	if os.WriteFile(tmp, b, 0o644) == nil {
		_ = os.Rename(tmp, s.metaPath(e.Key))
	}
}

// Delete removes the entry and its files.
func (s *Store) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entries, key)
	_ = os.Remove(s.VideoPath(key))
	_ = os.Remove(s.metaPath(key))
}

// NextQueued picks the oldest queued entry that is allowed to run now, and marks it running.
func (s *Store) NextQueued() (Entry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now().Unix()
	var best *Entry
	for _, e := range s.entries {
		if e.Status != stQueued || e.RetryAt > now {
			continue
		}
		if best == nil || e.RetryAt < best.RetryAt || (e.RetryAt == best.RetryAt && e.Key < best.Key) {
			best = e
		}
	}
	if best == nil {
		return Entry{}, false
	}
	best.Status = stRunning
	s.save(best)
	return *best, true
}

type Counts struct {
	Queued  int `json:"queued"`
	Running int `json:"running"`
	Ready   int `json:"ready"`
	Failed  int `json:"failed"`
}

func (s *Store) Stats() (Counts, int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var c Counts
	var bytes int64
	for _, e := range s.entries {
		switch e.Status {
		case stQueued:
			c.Queued++
		case stRunning:
			c.Running++
		case stReady:
			c.Ready++
			bytes += e.Bytes
		case stFailed:
			c.Failed++
		}
	}
	return c, bytes
}

// Touch records that the keys were played just now.
func (s *Store) Touch(keys []string) {
	now := s.now().Unix()
	for _, k := range keys {
		if !keyRe.MatchString(k) {
			continue
		}
		s.mu.Lock()
		if e, ok := s.entries[k]; ok && e.Status == stReady {
			e.LastUsed = now
			s.save(e)
		}
		s.mu.Unlock()
	}
}

// Evict deletes ready videos that were not used for a long time, then the least recently used ones until the
// total is under limit. It returns the number of videos deleted.
func (s *Store) Evict(limit int64, unusedAfter time.Duration) int {
	s.mu.Lock()
	var ready []*Entry
	var total int64
	for _, e := range s.entries {
		if e.Status == stReady {
			ready = append(ready, e)
			total += e.Bytes
		}
	}
	sort.Slice(ready, func(i, j int) bool { return lastUse(ready[i]) < lastUse(ready[j]) })
	cut := s.now().Add(-unusedAfter).Unix()
	var drop []string
	for _, e := range ready {
		if lastUse(e) < cut || total > limit {
			drop = append(drop, e.Key)
			total -= e.Bytes
		}
	}
	s.mu.Unlock()
	for _, k := range drop {
		s.Delete(k)
	}
	return len(drop)
}

func lastUse(e *Entry) int64 {
	if e.LastUsed > e.ReadyAt {
		return e.LastUsed
	}
	return e.ReadyAt
}

// Clear deletes every video and entry.
func (s *Store) Clear() {
	s.mu.Lock()
	keys := make([]string, 0, len(s.entries))
	for k := range s.entries {
		keys = append(keys, k)
	}
	s.mu.Unlock()
	for _, k := range keys {
		s.Delete(k)
	}
}

var errNotFound = errors.New("not found")

// SweepWork deletes work folders (".work-*") left by a crash or a failed clean-up when they are older than maxAge.
func (s *Store) SweepWork(maxAge time.Duration) int {
	dirs, _ := filepath.Glob(filepath.Join(s.dir, ".work-*"))
	cut := s.now().Add(-maxAge)
	n := 0
	for _, d := range dirs {
		if st, err := os.Stat(d); err == nil && st.IsDir() && st.ModTime().Before(cut) {
			if os.RemoveAll(d) == nil {
				n++
			}
		}
	}
	return n
}
