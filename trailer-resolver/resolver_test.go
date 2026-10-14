package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeRunner struct {
	mu    sync.Mutex
	calls []string
	fail  map[string]error
	size  int
}

func (f *fakeRunner) Download(_ context.Context, key string, _ int, dst string) (Result, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, key)
	if err := f.fail[key]; err != nil {
		return Result{}, err
	}
	n := f.size
	if n == 0 {
		n = 1000
	}
	return Result{Height: 720, Duration: 90}, os.WriteFile(dst, bytes.Repeat([]byte{1}, n), 0o644)
}

type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) Now() time.Time      { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *clock) Add(d time.Duration) { c.mu.Lock(); c.t = c.t.Add(d); c.mu.Unlock() }
func testConfig(dir string) Config {
	return Config{APIKey: "k", CacheDir: dir, CacheLimit: 1 << 30, MaxHeight: 720, HourlyCap: 100, Timeout: time.Minute, Workers: 1, BreakerAfter: 3, BreakerFor: time.Hour, UnusedAfter: 24 * time.Hour,
		YtDlp: "/nonexistent", Deno: "/nonexistent", FFmpeg: "/nonexistent"}
}

func newTest(t *testing.T, r Runner) (*Server, *Worker, *Store, *clock) {
	t.Helper()
	clk := &clock{t: time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)}
	cfg := testConfig(t.TempDir())
	st, err := NewStore(cfg.CacheDir, clk.Now)
	if err != nil {
		t.Fatal(err)
	}
	w := NewWorker(cfg, st, r, clk.Now)
	w.sleep = func(context.Context, time.Duration) {}
	return NewServer(cfg, st, w), w, st, clk
}

func do(t *testing.T, s *Server, method, path, body string, auth bool) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if auth {
		req.Header.Set("Authorization", "Bearer k")
	}
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

const keyA, keyB, keyC = "AAAAAAAAAAA", "BBBBBBBBBBB", "CCCCCCCCCCC"

func TestResolveQueuesThenReady(t *testing.T) {
	f := &fakeRunner{}
	s, w, st, _ := newTest(t, f)

	if code, _ := do(t, s, "POST", "/v1/resolve", `{"key":"`+keyA+`"}`, false); code != http.StatusUnauthorized {
		t.Fatalf("no key = %d", code)
	}
	for _, bad := range []string{`{"key":"short"}`, `{"key":"../../etc/pw"}`, `{"key":"` + keyA + `","maxHeight":5}`, `nope`} {
		if code, _ := do(t, s, "POST", "/v1/resolve", bad, true); code != http.StatusBadRequest {
			t.Fatalf("%s = %d", bad, code)
		}
	}
	code, b := do(t, s, "POST", "/v1/resolve", `{"key":"`+keyA+`"}`, true)
	if code != http.StatusAccepted || b["status"] != "queued" {
		t.Fatalf("resolve = %d %v", code, b)
	}
	e, _ := st.NextQueued()
	w.process(context.Background(), e)

	code, b = do(t, s, "GET", "/v1/status/"+keyA, "", true)
	if code != 200 || b["status"] != "ready" || b["height"] != float64(720) || b["bytes"] != float64(1000) {
		t.Fatalf("status = %d %v", code, b)
	}
	// asking again does not download again
	if code, b = do(t, s, "POST", "/v1/resolve", `{"key":"`+keyA+`"}`, true); code != 200 || b["status"] != "ready" {
		t.Fatalf("again = %d %v", code, b)
	}
	if _, ok := st.NextQueued(); ok || len(f.calls) != 1 {
		t.Fatalf("calls = %v", f.calls)
	}
	if code, _ = do(t, s, "GET", "/v1/status/"+keyB, "", true); code != http.StatusNotFound {
		t.Fatalf("unknown = %d", code)
	}
}

func TestPermanentFailureIsFinalAndDoesNotTripTheBreaker(t *testing.T) {
	f := &fakeRunner{fail: map[string]error{keyA: &DownloadError{Msg: "Video unavailable", Permanent: true}}}
	s, w, st, _ := newTest(t, f)
	for range 5 {
		do(t, s, "POST", "/v1/resolve", `{"key":"`+keyA+`"}`, true)
		if e, ok := st.NextQueued(); ok {
			w.process(context.Background(), e)
		}
	}
	code, b := do(t, s, "GET", "/v1/status/"+keyA, "", true)
	if code != 200 || b["status"] != "failed" || b["permanent"] != true {
		t.Fatalf("status = %v", b)
	}
	if len(f.calls) != 1 {
		t.Fatalf("a permanent failure was retried: %v", f.calls)
	}
	if state, _ := w.BreakerState(); state != "closed" {
		t.Fatal("breaker opened on a video problem")
	}
}

func TestTemporaryFailuresRetryThenOpenTheBreaker(t *testing.T) {
	boom := &DownloadError{Msg: "HTTP Error 403: Forbidden"}
	f := &fakeRunner{fail: map[string]error{keyA: boom, keyB: boom, keyC: boom}}
	s, w, st, clk := newTest(t, f)
	for _, k := range []string{keyA, keyB, keyC} {
		do(t, s, "POST", "/v1/resolve", `{"key":"`+k+`"}`, true)
		e, ok := st.NextQueued()
		if !ok {
			t.Fatal("nothing queued")
		}
		w.process(context.Background(), e)
	}
	// each one waits for a retry
	e, _ := st.Get(keyA)
	if e.Status != stQueued || e.RetryAt <= clk.Now().Unix() || e.Attempts != 1 {
		t.Fatalf("entry = %+v", e)
	}
	if state, until := w.BreakerState(); state != "open" || until != clk.Now().Add(time.Hour).Unix() {
		t.Fatalf("breaker = %s %d", state, until)
	}
	if w.blockedFor() <= 0 {
		t.Fatal("queue should be blocked")
	}
	clk.Add(2 * time.Hour)
	if state, _ := w.BreakerState(); state != "closed" {
		t.Fatal("breaker should close after its time")
	}
	// gives up after the retries
	for i := 0; i < 5; i++ {
		clk.Add(time.Hour)
		if e, ok := st.NextQueued(); ok && e.Key == keyA {
			w.process(context.Background(), e)
		}
	}
	if e, _ = st.Get(keyA); e.Status != stFailed {
		t.Fatalf("entry = %+v", e)
	}
}

func TestHourlyCap(t *testing.T) {
	s, w, _, clk := newTest(t, &fakeRunner{})
	_ = s
	w.cfg.HourlyCap = 2
	w.noteStart()
	w.noteStart()
	if d := w.blockedFor(); d <= 0 || d > time.Hour+time.Second {
		t.Fatalf("blockedFor = %v", d)
	}
	clk.Add(61 * time.Minute)
	if d := w.blockedFor(); d != 0 {
		t.Fatalf("after an hour = %v", d)
	}
}

func TestEvictionLeastRecentlyUsedAndUnused(t *testing.T) {
	f := &fakeRunner{size: 400}
	s, w, st, clk := newTest(t, f)
	for _, k := range []string{keyA, keyB, keyC} {
		do(t, s, "POST", "/v1/resolve", `{"key":"`+k+`"}`, true)
		e, _ := st.NextQueued()
		w.process(context.Background(), e)
		clk.Add(time.Minute)
	}
	st.Touch([]string{keyA}) // A was played last, so B is the oldest
	if n := st.Evict(1000, 24*time.Hour); n != 1 {
		t.Fatalf("evicted %d", n)
	}
	if _, ok := st.Get(keyB); ok {
		t.Fatal("B should be gone")
	}
	if _, err := os.Stat(st.VideoPath(keyB)); !os.IsNotExist(err) {
		t.Fatal("B file still there")
	}
	clk.Add(48 * time.Hour)
	if n := st.Evict(1<<30, 24*time.Hour); n != 2 {
		t.Fatalf("unused evicted %d", n)
	}
}

func TestRestartKeepsReadyAndRequeuesRunning(t *testing.T) {
	f := &fakeRunner{}
	s, w, st, clk := newTest(t, f)
	do(t, s, "POST", "/v1/resolve", `{"key":"`+keyA+`"}`, true)
	e, _ := st.NextQueued()
	w.process(context.Background(), e)
	do(t, s, "POST", "/v1/resolve", `{"key":"`+keyB+`"}`, true)
	st.NextQueued() // B is running when the process dies
	again, err := NewStore(st.dir, clk.Now)
	if err != nil {
		t.Fatal(err)
	}
	if a, _ := again.Get(keyA); a.Status != stReady {
		t.Fatalf("A = %+v", a)
	}
	if b, _ := again.Get(keyB); b.Status != stQueued {
		t.Fatalf("B = %+v", b)
	}
	// a ready entry whose file vanished is fetched again
	os.Remove(again.VideoPath(keyA))
	third, _ := NewStore(st.dir, clk.Now)
	if a, _ := third.Get(keyA); a.Status != stQueued {
		t.Fatalf("A without file = %+v", a)
	}
}

func TestHealthAndClear(t *testing.T) {
	f := &fakeRunner{}
	s, w, st, _ := newTest(t, f)
	do(t, s, "POST", "/v1/resolve", `{"key":"`+keyA+`"}`, true)
	e, _ := st.NextQueued()
	w.process(context.Background(), e)
	code, b := do(t, s, "GET", "/v1/health", "", true)
	counts, _ := b["counts"].(map[string]any)
	if code != 200 || counts["ready"] != float64(1) || b["cacheBytes"] != float64(1000) {
		t.Fatalf("health = %d %v", code, b)
	}
	if code, _ = do(t, s, "DELETE", "/v1/cache", "", true); code != http.StatusNoContent {
		t.Fatalf("clear = %d", code)
	}
	if _, ok := st.Get(keyA); ok {
		t.Fatal("entry survived clear")
	}
}

func TestClassify(t *testing.T) {
	cases := map[string]bool{
		"ERROR: [youtube] abc: Video unavailable":                                    true,
		"ERROR: [youtube] abc: This video is unavailable":                            true,
		"ERROR: [youtube] abc: Private video. Sign in if you've been granted access": true,
		"ERROR: unable to download video data: HTTP Error 403: Forbidden":            false,
		"ERROR: [youtube] abc: Sign in to confirm you're not a bot":                  false,
	}
	for msg, want := range cases {
		err := classify(msg, errors.New("exit status 1"))
		var de *DownloadError
		if !errors.As(err, &de) || de.Permanent != want {
			t.Errorf("%q permanent = %v, want %v", msg, de != nil && de.Permanent, want)
		}
	}
}

func TestLoginWall(t *testing.T) {
	cases := map[string]bool{
		"ERROR: [youtube] abc: Please sign in. Use --cookies-from-browser or --cookies for the authentication.": true,
		"ERROR: [youtube] abc: Sign in to confirm you're not a bot":                                             false,
		"ERROR: [youtube] abc: This video is unavailable":                                                       false,
	}
	for msg, want := range cases {
		if got := loginWall(msg); got != want {
			t.Errorf("loginWall(%q) = %v, want %v", msg, got, want)
		}
	}
}

func TestSweepWorkRemovesOnlyOldWorkFolders(t *testing.T) {
	_, _, st, clk := newTest(t, &fakeRunner{})
	old := filepath.Join(st.dir, ".work-AAAAAAAAAAA-1")
	fresh := filepath.Join(st.dir, ".work-BBBBBBBBBBB-2")
	for _, d := range []string{old, fresh} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		os.WriteFile(filepath.Join(d, "in.mp4"), []byte("x"), 0o644)
	}
	past := clk.Now().Add(-time.Hour)
	os.Chtimes(old, past, past)
	if n := st.SweepWork(10 * time.Minute); n != 1 {
		t.Fatalf("swept %d", n)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatal("old folder still there")
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Fatal("fresh folder was removed")
	}
}
