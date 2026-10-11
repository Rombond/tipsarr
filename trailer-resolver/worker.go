package main

import (
	"context"
	"errors"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"
)

// retry waits for temporary failures: 1 minute, 5 minutes, 30 minutes, then the entry is given up.
var retryWaits = []time.Duration{time.Minute, 5 * time.Minute, 30 * time.Minute}

type Breaker struct {
	mu          sync.Mutex
	failures    int
	openedUntil time.Time
}

type Worker struct {
	cfg    Config
	store  *Store
	runner Runner
	now    func() time.Time
	sleep  func(ctx context.Context, d time.Duration) // tests replace it

	wake chan struct{}

	mu        sync.Mutex
	starts    []time.Time // download starts in the last hour
	breaker   Breaker
	lastError string
}

func NewWorker(cfg Config, st *Store, r Runner, now func() time.Time) *Worker {
	return &Worker{cfg: cfg, store: st, runner: r, now: now, wake: make(chan struct{}, 1), sleep: sleepCtx}
}

func sleepCtx(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
	case <-ctx.Done():
	}
}

// Poke tells the workers there may be something to do.
func (w *Worker) Poke() {
	select {
	case w.wake <- struct{}{}:
	default:
	}
}

// Run starts cfg.Workers loops and returns when ctx ends.
func (w *Worker) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for range w.cfg.Workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.loop(ctx)
		}()
	}
	wg.Wait()
}

func (w *Worker) loop(ctx context.Context) {
	for ctx.Err() == nil {
		if wait := w.blockedFor(); wait > 0 {
			w.waitFor(ctx, min(wait, time.Minute))
			continue
		}
		e, ok := w.store.NextQueued()
		if !ok {
			w.waitFor(ctx, 15*time.Second)
			continue
		}
		w.noteStart()
		w.process(ctx, e)
		// pause between downloads so the address never looks like a scraper
		span := w.cfg.PauseMax - w.cfg.PauseMin
		pause := w.cfg.PauseMin
		if span > 0 {
			pause += time.Duration(rand.Int64N(int64(span)))
		}
		w.sleep(ctx, pause)
	}
}

func (w *Worker) waitFor(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
	case <-w.wake:
	case <-ctx.Done():
	}
}

// blockedFor is how long the queue must stay idle: the breaker is open, or the hourly cap is reached.
func (w *Worker) blockedFor() time.Duration {
	now := w.now()
	w.breaker.mu.Lock()
	until := w.breaker.openedUntil
	w.breaker.mu.Unlock()
	if until.After(now) {
		return until.Sub(now)
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	cut := now.Add(-time.Hour)
	kept := w.starts[:0]
	for _, t := range w.starts {
		if t.After(cut) {
			kept = append(kept, t)
		}
	}
	w.starts = kept
	if w.cfg.HourlyCap > 0 && len(w.starts) >= w.cfg.HourlyCap {
		return w.starts[0].Add(time.Hour).Sub(now) + time.Second
	}
	return 0
}

func (w *Worker) noteStart() {
	w.mu.Lock()
	w.starts = append(w.starts, w.now())
	w.mu.Unlock()
}

// BreakerState reports "open" with the time it ends, or "closed".
func (w *Worker) BreakerState() (string, int64) {
	w.breaker.mu.Lock()
	defer w.breaker.mu.Unlock()
	if w.breaker.openedUntil.After(w.now()) {
		return "open", w.breaker.openedUntil.Unix()
	}
	return "closed", 0
}

func (w *Worker) LastError() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.lastError
}

func (w *Worker) process(ctx context.Context, e Entry) {
	height := e.MaxHeight
	if height <= 0 {
		height = w.cfg.MaxHeight
	}
	dctx, cancel := context.WithTimeout(ctx, w.cfg.Timeout)
	defer cancel()
	res, err := w.runner.Download(dctx, e.Key, height, w.store.VideoPath(e.Key))
	if ctx.Err() != nil {
		w.store.Update(e.Key, func(e *Entry) { e.Status = stQueued })
		return
	}
	if err == nil {
		st := fileSize(w.store.VideoPath(e.Key))
		w.store.Update(e.Key, func(e *Entry) {
			e.Status, e.Error, e.Permanent = stReady, "", false
			e.Bytes, e.Height, e.Duration = st, res.Height, res.Duration
			e.ReadyAt, e.RetryAt = w.now().Unix(), 0
		})
		w.breaker.mu.Lock()
		w.breaker.failures = 0
		w.breaker.mu.Unlock()
		slog.Info("trailer ready", "key", e.Key, "height", res.Height, "bytes", st)
		w.store.Evict(w.cfg.CacheLimit, w.cfg.UnusedAfter)
		return
	}
	var de *DownloadError
	permanent := errors.As(err, &de) && de.Permanent
	msg := err.Error()
	w.mu.Lock()
	w.lastError = msg
	w.mu.Unlock()
	slog.Warn("trailer failed", "key", e.Key, "permanent", permanent, "err", msg)
	if permanent {
		w.store.Update(e.Key, func(e *Entry) { e.Status, e.Error, e.Permanent, e.Attempts = stFailed, msg, true, e.Attempts+1 })
		return // a video problem says nothing about the address being blocked
	}
	w.store.Update(e.Key, func(x *Entry) {
		x.Attempts++
		x.Error = msg
		if x.Attempts > len(retryWaits) {
			x.Status = stFailed
			return
		}
		x.Status = stQueued
		x.RetryAt = w.now().Add(retryWaits[x.Attempts-1]).Unix()
	})
	w.breaker.mu.Lock()
	w.breaker.failures++
	if w.cfg.BreakerAfter > 0 && w.breaker.failures >= w.cfg.BreakerAfter {
		w.breaker.openedUntil = w.now().Add(w.cfg.BreakerFor)
		w.breaker.failures = 0
		slog.Error("too many temporary failures in a row: pausing downloads", "until", w.breaker.openedUntil)
	}
	w.breaker.mu.Unlock()
}
