// Package follow pushes the stage and progress of followed requests to the phones that follow them
// (iOS Live Activity, Android ongoing notification), through the push relay's live updates.
package follow

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/push"
	"github.com/Rombond/tipsarr/backend/internal/store"
)

const (
	// IOSLifetime ends a follow before Apple's 8 h Live Activity limit.
	IOSLifetime = 8*time.Hour - 10*time.Minute
	// AndroidLifetime: an ongoing notification has no system limit; each re-follow extends it.
	AndroidLifetime = 24 * time.Hour
	// MaxPerSession caps the follows of one app install.
	MaxPerSession = 10

	minGap        = 60 * time.Second  // between two sends to one follow, except stage changes
	stepPercent   = 1                 // any move is worth a send, at most once a minute (minGap)
	refreshGap    = 300 * time.Second // progress sent anyway after this long
	finalDismiss  = 3600              // seconds an ended Live Activity stays on the lock screen
	queueSize     = 256
	expiryEvery   = time.Minute
	defaultRetry  = 10 * time.Second
	stageDownload = "downloading"
)

// Final reports whether a stage ends a follow.
func Final(stage string) bool {
	return stage == "available" || stage == "declined" || stage == "failed"
}

// Sender delivers live updates (push.Service).
type Sender interface {
	SendLive(ctx context.Context, items []push.LiveItem) ([]push.LiveResult, error)
}

type kind int

const (
	evStage kind = iota
	evProgress
	evRemoved
)

type event struct {
	kind    kind
	id      string
	stage   string
	percent int
	eta     int
}

type Service struct {
	store  *store.Store
	sender Sender
	now    func() time.Time
	retry  time.Duration

	ch chan event

	mu  sync.Mutex
	ids map[string]bool // followed request ids, so hooks cost nothing when no one follows

	// putMu orders a new follow (save, then mark) with the worker's unmark (count, then unmark): without it
	// the worker could count no row, a follow be saved and marked, and the worker unmark it for good.
	putMu sync.Mutex
}

func New(st *store.Store, sender Sender) *Service {
	return &Service{store: st, sender: sender, now: time.Now, retry: defaultRetry, ch: make(chan event, queueSize), ids: map[string]bool{}}
}

// SetClock and SetRetry are for tests.
func (s *Service) SetClock(now func() time.Time) { s.now = now }
func (s *Service) SetRetry(d time.Duration)      { s.retry = d }

// Start loads the followed ids and runs the worker and the expiry ticker until ctx ends.
func (s *Service) Start(ctx context.Context) {
	if ids, err := s.store.FollowedRequestIDs(ctx); err == nil {
		s.mu.Lock()
		for _, id := range ids {
			s.ids[id] = true
		}
		s.mu.Unlock()
	} else {
		slog.Warn("follow: load", "err", err)
	}
	go func() {
		t := time.NewTicker(expiryEvery)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case ev := <-s.ch:
				s.handle(ctx, ev)
			case <-t.C:
				s.Expire(ctx)
			}
		}
	}()
}

func (s *Service) followed(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ids[id]
}

func (s *Service) mark(id string, on bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if on {
		s.ids[id] = true
	} else {
		delete(s.ids, id)
	}
}

// unmarkIfUnfollowed forgets a request id once no follow is left for it.
func (s *Service) unmarkIfUnfollowed(ctx context.Context, id string) {
	s.putMu.Lock()
	defer s.putMu.Unlock()
	if rest, err := s.store.FollowsForRequest(ctx, id); err == nil && len(rest) == 0 {
		s.mark(id, false)
	}
}

// enqueue never blocks the caller: a full queue drops progress, and hands stage changes to a goroutine.
func (s *Service) enqueue(ev event) {
	if !s.followed(ev.id) {
		return
	}
	select {
	case s.ch <- ev:
	default:
		if ev.kind != evProgress {
			go func() { s.ch <- ev }()
		}
	}
}

// Stage is called on every stage change of a request (requests.LiveNotifier).
func (s *Service) Stage(id, stage string, percent, eta int) {
	s.enqueue(event{kind: evStage, id: id, stage: stage, percent: percent, eta: eta})
}

// Progress is called when a download's percent or ETA changes (requests.LiveNotifier).
func (s *Service) Progress(id string, percent, eta int) {
	s.enqueue(event{kind: evProgress, id: id, stage: stageDownload, percent: percent, eta: eta})
}

// Removed is called when a request is deleted (requests.LiveNotifier).
func (s *Service) Removed(id string) {
	s.enqueue(event{kind: evRemoved, id: id})
}

// handle processes one event.
func (s *Service) handle(ctx context.Context, ev event) {
	follows, err := s.store.FollowsForRequest(ctx, ev.id)
	if err != nil {
		slog.Warn("follow: load follows", "err", err)
		return
	}
	if len(follows) == 0 {
		s.unmarkIfUnfollowed(ctx, ev.id)
		return
	}
	now := s.now()
	switch {
	case ev.kind == evRemoved:
		s.end(ctx, follows, nil, 0)
	case Final(ev.stage):
		s.end(ctx, follows, &ev, finalDismiss)
	default:
		var items []push.LiveItem
		var sent []store.Follow
		for _, f := range follows {
			high, ok := decide(f, ev, now)
			if !ok {
				continue
			}
			items = append(items, item(f, ev.stage, ev.percent, ev.eta, now, false, 0, high))
			sent = append(sent, f)
		}
		s.send(ctx, items, sent, true)
	}
}

// decide applies the throttling rules to one follow: whether to send, and at which priority.
func decide(f store.Follow, ev event, now time.Time) (high, ok bool) {
	since := now.Sub(time.Unix(f.LastSentAt, 0))
	if ev.stage != f.LastStage {
		if ev.stage == "searching" && f.LastStage == stageDownload {
			return false, since >= minGap // a paused download: held a minute in case it resumes
		}
		return true, true // requested, approved, or a download that starts
	}
	if ev.stage != stageDownload || since < minGap {
		return false, false
	}
	diff := ev.percent - f.LastPercent
	if diff < 0 {
		diff = -diff
	}
	if diff >= stepPercent || since >= refreshGap { // the ETA moves even when the percent does not
		return false, true
	}
	return false, false
}

func item(f store.Follow, stage string, percent, eta int, now time.Time, end bool, dismiss int, high bool) push.LiveItem {
	return push.LiveItem{Platform: f.Platform, Token: f.PushToken, Sandbox: f.Sandbox, ID: f.RequestID, Server: f.Server,
		Stage: stage, Percent: percent, ETASeconds: eta, Timestamp: now.Unix(), End: end, DismissAfter: dismiss, High: high}
}

// end sends the final update to each follow and deletes them. ev nil = the request is gone: last stage sent, dismissed at once.
func (s *Service) end(ctx context.Context, follows []store.Follow, ev *event, dismiss int) {
	now := s.now()
	items := make([]push.LiveItem, len(follows))
	for i, f := range follows {
		if ev != nil {
			items[i] = item(f, ev.stage, ev.percent, ev.eta, now, true, dismiss, true)
		} else {
			items[i] = item(f, f.LastStage, f.LastPercent, 0, now, true, dismiss, true)
		}
	}
	for _, f := range follows {
		if err := s.store.DeleteFollowByID(ctx, f.ID); err != nil {
			slog.Warn("follow: delete", "err", err)
		}
	}
	if len(follows) > 0 {
		s.unmarkIfUnfollowed(ctx, follows[0].RequestID)
	}
	s.send(ctx, items, nil, true)
}

// send delivers items; follows (same order) get their throttle state updated, or are dropped on invalid_token.
// When retry is set, items that failed are sent once more after s.retry, high priority ones only.
func (s *Service) send(ctx context.Context, items []push.LiveItem, follows []store.Follow, retry bool) {
	if len(items) == 0 {
		return
	}
	res, err := s.sender.SendLive(ctx, items)
	var again []push.LiveItem
	var againF []store.Follow
	for i, it := range items {
		st := "error"
		if err == nil && i < len(res) {
			st = res[i].Status
		}
		var f *store.Follow
		if follows != nil {
			f = &follows[i]
		}
		switch st {
		case "sent":
			if f != nil {
				if err := s.store.MarkFollowSent(ctx, f.ID, it.Stage, it.Percent, it.Timestamp); err != nil {
					slog.Warn("follow: mark sent", "err", err)
				}
			}
		case "invalid_token":
			if f != nil {
				if err := s.store.DeleteFollowByID(ctx, f.ID); err != nil {
					slog.Warn("follow: drop", "err", err)
				}
			}
		case "error", "rate_limited":
			if it.High {
				again = append(again, it)
				if f != nil {
					againF = append(againF, *f)
				}
			}
		}
	}
	if err != nil {
		slog.Warn("follow: live update failed", "items", len(items), "err", err)
	}
	if retry && len(again) > 0 {
		if follows == nil {
			againF = nil // ended follows: nothing to update
		}
		time.AfterFunc(s.retry, func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			again, againF = s.stillCurrent(ctx, again, againF)
			s.send(ctx, again, againF, false)
		})
	}
}

// stillCurrent drops the retries the worker made stale meanwhile: a follow ended (deleted) or sent a newer
// state. Without it a late retry could push an update after the end, or roll the throttle state back.
// Ended follows (follows nil) are kept: their end must still reach the phone.
func (s *Service) stillCurrent(ctx context.Context, items []push.LiveItem, follows []store.Follow) ([]push.LiveItem, []store.Follow) {
	if follows == nil {
		return items, nil
	}
	var outI []push.LiveItem
	var outF []store.Follow
	for i, it := range items {
		rows, err := s.store.FollowsForRequest(ctx, it.ID)
		if err != nil {
			continue
		}
		for _, f := range rows {
			if f.ID == follows[i].ID && f.LastSentAt <= it.Timestamp {
				outI, outF = append(outI, it), append(outF, f)
				break
			}
		}
	}
	return outI, outF
}

// Expire ends the follows past their lifetime.
func (s *Service) Expire(ctx context.Context) {
	rows, err := s.store.ExpiredFollows(ctx, s.now().Unix())
	if err != nil {
		slog.Warn("follow: expired", "err", err)
		return
	}
	for _, f := range rows {
		s.end(ctx, []store.Follow{f}, nil, 0)
	}
}

// Put creates or refreshes a follow; stage and percent are the request's current state, which the app shows
// itself, so they start the throttle state. A new iOS activity id (or any Android call) restarts the lifetime.
func (s *Service) Put(ctx context.Context, f *store.Follow, stage string, percent int) error {
	now := s.now()
	life := AndroidLifetime
	if f.Platform == "ios" {
		life = IOSLifetime
	}
	f.CreatedAt, f.ExpiresAt = now.Unix(), now.Add(life).Unix()
	f.LastStage, f.LastPercent, f.LastSentAt = stage, percent, now.Unix()
	s.putMu.Lock()
	defer s.putMu.Unlock()
	_, err := s.store.SaveFollow(ctx, f, func(old *store.Follow) bool {
		return f.Platform == "android" || old.ActivityID != f.ActivityID
	})
	if err != nil {
		return err
	}
	s.mark(f.RequestID, true)
	return nil
}

func (s *Service) Delete(ctx context.Context, sessionID, requestID string) error {
	return s.store.DeleteFollow(ctx, sessionID, requestID)
}

func (s *Service) List(ctx context.Context, sessionID string) ([]store.Follow, error) {
	return s.store.FollowsBySession(ctx, sessionID)
}
