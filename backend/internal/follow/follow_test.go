package follow

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/push"
	"github.com/Rombond/tipsarr/backend/internal/store"
	"github.com/Rombond/tipsarr/backend/internal/store/storetest"
)

type fakeSender struct {
	mu     sync.Mutex
	got    []push.LiveItem
	status string
}

func (f *fakeSender) SendLive(_ context.Context, items []push.LiveItem) ([]push.LiveResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.got = append(f.got, items...)
	out := make([]push.LiveResult, len(items))
	for i := range out {
		out[i].Status = "sent"
		if f.status != "" {
			out[i].Status = f.status
		}
	}
	return out, nil
}

func (f *fakeSender) take() []push.LiveItem {
	f.mu.Lock()
	defer f.mu.Unlock()
	g := f.got
	f.got = nil
	return g
}

const tok = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func setup(t *testing.T) (*Service, *fakeSender, *time.Time, *store.Store) {
	t.Helper()
	st := storetest.Open(t)
	fs := &fakeSender{}
	s := New(st, fs)
	now := time.Unix(1_800_000_000, 0)
	s.SetClock(func() time.Time { return now })
	s.SetRetry(time.Hour)
	return s, fs, &now, st
}

func put(t *testing.T, s *Service, session, platform, stage string) {
	t.Helper()
	f := &store.Follow{RequestID: "r1", UserID: "u1", SessionID: session, Platform: platform, PushToken: tok, ActivityID: "a-" + session}
	if err := s.Put(context.Background(), f, stage, 0); err != nil {
		t.Fatal(err)
	}
}

func run(s *Service, ev event) { s.handle(context.Background(), ev) }

func TestThrottling(t *testing.T) {
	s, fs, now, _ := setup(t)
	put(t, s, "s1", "ios", "approved")

	// first download: stage change, immediate, high
	run(s, event{kind: evProgress, id: "r1", stage: "downloading", percent: 1, eta: 900})
	if g := fs.take(); len(g) != 1 || !g[0].High || g[0].Stage != "downloading" || g[0].Percent != 1 {
		t.Fatalf("%+v", g)
	}
	// same stage, too soon
	*now = now.Add(30 * time.Second)
	run(s, event{kind: evProgress, id: "r1", stage: "downloading", percent: 20})
	if g := fs.take(); len(g) != 0 {
		t.Fatalf("too soon: %+v", g)
	}
	// 60 s later, small change: no
	*now = now.Add(40 * time.Second)
	run(s, event{kind: evProgress, id: "r1", stage: "downloading", percent: 3})
	if g := fs.take(); len(g) != 0 {
		t.Fatalf("small change: %+v", g)
	}
	// >= 5 %: low priority send
	run(s, event{kind: evProgress, id: "r1", stage: "downloading", percent: 7})
	if g := fs.take(); len(g) != 1 || g[0].High || g[0].Percent != 7 {
		t.Fatalf("%+v", g)
	}
	// 5 min later, same percent: refresh (the ETA moves)
	*now = now.Add(301 * time.Second)
	run(s, event{kind: evProgress, id: "r1", stage: "downloading", percent: 8, eta: 100})
	if g := fs.take(); len(g) != 1 {
		t.Fatalf("refresh: %+v", g)
	}
	// back to searching within 60 s: held
	*now = now.Add(10 * time.Second)
	run(s, event{kind: evStage, id: "r1", stage: "searching"})
	if g := fs.take(); len(g) != 0 {
		t.Fatalf("flap: %+v", g)
	}
	*now = now.Add(60 * time.Second)
	run(s, event{kind: evStage, id: "r1", stage: "searching"})
	if g := fs.take(); len(g) != 1 || g[0].High {
		t.Fatalf("searching: %+v", g)
	}
	// downloading again right after: sent at once, high
	*now = now.Add(time.Second)
	run(s, event{kind: evProgress, id: "r1", stage: "downloading", percent: 9})
	if g := fs.take(); len(g) != 1 || !g[0].High {
		t.Fatalf("download restart: %+v", g)
	}
}

func TestFinalEndsAndDeletes(t *testing.T) {
	s, fs, _, st := setup(t)
	put(t, s, "s1", "ios", "approved")
	put(t, s, "s2", "android", "approved")
	run(s, event{kind: evStage, id: "r1", stage: "available", percent: 100})
	g := fs.take()
	if len(g) != 2 || !g[0].End || !g[0].High || g[0].DismissAfter != 3600 || g[0].Stage != "available" || g[0].Percent != 100 {
		t.Fatalf("%+v", g)
	}
	if rows, _ := st.FollowsForRequest(context.Background(), "r1"); len(rows) != 0 {
		t.Fatalf("follows left: %d", len(rows))
	}
	if s.followed("r1") {
		t.Fatal("still marked")
	}
}

func TestRemovedAndExpiry(t *testing.T) {
	s, fs, now, st := setup(t)
	put(t, s, "s1", "ios", "searching")
	run(s, event{kind: evRemoved, id: "r1"})
	if g := fs.take(); len(g) != 1 || !g[0].End || g[0].DismissAfter != 0 || g[0].Stage != "searching" {
		t.Fatalf("%+v", g)
	}
	put(t, s, "s1", "ios", "searching")
	*now = now.Add(IOSLifetime - time.Minute)
	s.Expire(context.Background())
	if g := fs.take(); len(g) != 0 {
		t.Fatalf("expired early: %+v", g)
	}
	*now = now.Add(2 * time.Minute)
	s.Expire(context.Background())
	if g := fs.take(); len(g) != 1 || !g[0].End || g[0].DismissAfter != 0 {
		t.Fatalf("%+v", g)
	}
	if rows, _ := st.FollowsForRequest(context.Background(), "r1"); len(rows) != 0 {
		t.Fatal("expired follow kept")
	}
}

func TestInvalidTokenDropsFollow(t *testing.T) {
	s, fs, _, st := setup(t)
	put(t, s, "s1", "android", "approved")
	fs.status = "invalid_token"
	run(s, event{kind: evProgress, id: "r1", stage: "downloading", percent: 10})
	if rows, _ := st.FollowsForRequest(context.Background(), "r1"); len(rows) != 0 {
		t.Fatal("follow kept after invalid_token")
	}
}

func TestReputRules(t *testing.T) {
	s, _, now, st := setup(t)
	ctx := context.Background()
	f := &store.Follow{RequestID: "r1", UserID: "u1", SessionID: "s1", Platform: "ios", PushToken: tok, ActivityID: "a"}
	s.Put(ctx, f, "approved", 0)
	first := f.ExpiresAt
	*now = now.Add(time.Hour)
	f2 := &store.Follow{RequestID: "r1", UserID: "u1", SessionID: "s1", Platform: "ios", PushToken: tok + "00", ActivityID: "a"}
	s.Put(ctx, f2, "searching", 0)
	rows, _ := st.FollowsBySession(ctx, "s1")
	if len(rows) != 1 || rows[0].ExpiresAt != first || rows[0].PushToken != tok+"00" || rows[0].LastStage != "approved" {
		t.Fatalf("same activity: %+v", rows)
	}
	f3 := &store.Follow{RequestID: "r1", UserID: "u1", SessionID: "s1", Platform: "ios", PushToken: tok, ActivityID: "b"}
	s.Put(ctx, f3, "searching", 0)
	rows, _ = st.FollowsBySession(ctx, "s1")
	if len(rows) != 1 || rows[0].ExpiresAt == first || rows[0].LastStage != "searching" {
		t.Fatalf("new activity: %+v", rows)
	}
}

func TestUnfollowedRequestCostsNothing(t *testing.T) {
	s, _, _, _ := setup(t)
	s.Progress("nobody", 50, 0)
	select {
	case <-s.ch:
		t.Fatal("queued an event for a request nobody follows")
	default:
	}
}

func TestStaleRetryDropped(t *testing.T) {
	s, _, now, st := setup(t)
	ctx := context.Background()
	put(t, s, "s1", "ios", "requested")
	rows, _ := st.FollowsForRequest(ctx, "r1")
	old := item(rows[0], "approved", 0, 0, *now, false, 0, true)

	// still current: kept
	if items, _ := s.stillCurrent(ctx, []push.LiveItem{old}, rows); len(items) != 1 {
		t.Fatalf("current retry dropped")
	}
	// the worker sent a newer state meanwhile: dropped, and a late mark does not roll it back
	if err := st.MarkFollowSent(ctx, rows[0].ID, "downloading", 40, now.Unix()+30); err != nil {
		t.Fatal(err)
	}
	if items, _ := s.stillCurrent(ctx, []push.LiveItem{old}, rows); len(items) != 0 {
		t.Fatalf("stale retry kept")
	}
	_ = st.MarkFollowSent(ctx, rows[0].ID, "approved", 0, now.Unix())
	if cur, _ := st.FollowsForRequest(ctx, "r1"); cur[0].LastStage != "downloading" {
		t.Fatalf("state rolled back to %s", cur[0].LastStage)
	}
	// the follow ended: dropped
	_ = st.DeleteFollowByID(ctx, rows[0].ID)
	if items, _ := s.stillCurrent(ctx, []push.LiveItem{old}, rows); len(items) != 0 {
		t.Fatalf("retry after the end kept")
	}
}
