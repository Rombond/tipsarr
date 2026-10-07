// Package jobs runs named background jobs on an interval and records their status.
package jobs

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/store"
)

var (
	ErrUnknown        = errors.New("unknown job")
	ErrAlreadyRunning = errors.New("job already running")
	// ErrSkipped lets a job report "nothing to do" (e.g. not configured) without failing.
	ErrSkipped = errors.New("skipped")
)

type Job struct {
	Name         string
	Every        time.Duration
	InitialDelay time.Duration
	// Quiet jobs run often; their runs are only recorded when they fail (or recover from a failure).
	Quiet bool
	// Run returns a short human-readable summary.
	Run func(ctx context.Context) (string, error)
}

type Manager struct {
	// OnChange (optional) is called when a non-quiet job starts or finishes.
	OnChange func(name, status, message string)

	store *store.Store
	jobs  map[string]Job
	order []string

	mu      sync.Mutex
	running map[string]bool
	failing map[string]bool // quiet jobs whose last recorded run was an error
}

func New(s *store.Store) *Manager {
	return &Manager{store: s, jobs: map[string]Job{}, running: map[string]bool{}, failing: map[string]bool{}}
}

func (m *Manager) Register(j Job) {
	m.jobs[j.Name] = j
	m.order = append(m.order, j.Name)
}

// Start launches one scheduler goroutine per job; they stop when ctx is cancelled.
func (m *Manager) Start(ctx context.Context) {
	// A "running" row left by a previous process means that run was interrupted.
	if runs, err := m.store.JobRuns(ctx); err == nil {
		for _, r := range runs {
			if r.Status == "running" {
				r.Status, r.Message, r.LastFinishedAt = "error", "interrupted by restart", time.Now().Unix()
				_ = m.store.SaveJobRun(ctx, &r)
			}
		}
	}
	for _, name := range m.order {
		j := m.jobs[name]
		go func() {
			select {
			case <-time.After(j.InitialDelay):
			case <-ctx.Done():
				return
			}
			failures := 0
			for {
				err := m.run(ctx, j)
				if err != nil && !errors.Is(err, ErrAlreadyRunning) {
					failures++
					slog.Warn("job failed", "job", j.Name, "err", err, "attempt", failures)
				} else {
					failures = 0
				}
				select {
				case <-time.After(nextWait(j.Every, failures)):
				case <-ctx.Done():
					return
				}
			}
		}()
	}
}

const (
	retryFirst = 30 * time.Second
	retryMax   = 10 * time.Minute
)

// nextWait is the pause before a job runs again: its normal interval after a success, and a
// growing delay (30s, 1m, 2m, ... capped at 10 min and at the interval itself) after failures,
// so a job that failed because a service was still starting does not stay stale for hours.
func nextWait(every time.Duration, failures int) time.Duration {
	if failures <= 0 {
		return every
	}
	d := retryFirst << min(failures-1, 5)
	return min(d, retryMax, every)
}

// RunNow starts a job in the background. It returns immediately.
func (m *Manager) RunNow(ctx context.Context, name string) error {
	j, ok := m.jobs[name]
	if !ok {
		return ErrUnknown
	}
	m.mu.Lock()
	busy := m.running[name]
	m.mu.Unlock()
	if busy {
		return ErrAlreadyRunning
	}
	go func() {
		if err := m.run(context.WithoutCancel(ctx), j); err != nil && !errors.Is(err, ErrAlreadyRunning) {
			slog.Warn("job failed", "job", j.Name, "err", err)
		}
	}()
	return nil
}

func (m *Manager) run(ctx context.Context, j Job) error {
	m.mu.Lock()
	if m.running[j.Name] {
		m.mu.Unlock()
		return ErrAlreadyRunning
	}
	m.running[j.Name] = true
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		delete(m.running, j.Name)
		m.mu.Unlock()
	}()

	rec := &store.JobRun{Name: j.Name, LastStartedAt: time.Now().Unix(), Status: "running"}
	if !j.Quiet {
		_ = m.store.SaveJobRun(ctx, rec)
		m.notifyChange(j.Name, "running", "")
	}

	jobCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	msg, err := j.Run(jobCtx)

	rec.LastFinishedAt = time.Now().Unix()
	switch {
	case errors.Is(err, ErrSkipped):
		rec.Status, rec.Message, err = "skipped", err.Error(), nil
		if msg != "" {
			rec.Message = msg
		}
	case err != nil:
		rec.Status, rec.Message = "error", err.Error()
	default:
		rec.Status, rec.Message = "ok", msg
	}
	m.mu.Lock()
	record := !j.Quiet || rec.Status == "error" || m.failing[j.Name]
	m.failing[j.Name] = j.Quiet && rec.Status == "error"
	m.mu.Unlock()
	if record {
		m.notifyChange(j.Name, rec.Status, rec.Message)
		if serr := m.store.SaveJobRun(context.WithoutCancel(ctx), rec); serr != nil {
			slog.Warn("save job run", "err", serr)
		}
	}
	return err
}

type Status struct {
	store.JobRun
	EverySeconds int  `json:"everySeconds"`
	Running      bool `json:"running"`
}

// Statuses returns every registered job with its last run.
func (m *Manager) Statuses(ctx context.Context) ([]Status, error) {
	runs, err := m.store.JobRuns(ctx)
	if err != nil {
		return nil, err
	}
	byName := map[string]store.JobRun{}
	for _, r := range runs {
		byName[r.Name] = r
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Status, 0, len(m.order))
	for _, name := range m.order {
		if m.jobs[name].Quiet {
			continue // internal housekeeping, not shown
		}
		r, ok := byName[name]
		if !ok {
			r = store.JobRun{Name: name, Status: "never"}
		}
		out = append(out, Status{JobRun: r, EverySeconds: int(m.jobs[name].Every.Seconds()), Running: m.running[name]})
	}
	return out, nil
}

func (m *Manager) notifyChange(name, status, message string) {
	if m.OnChange != nil {
		m.OnChange(name, status, message)
	}
}
