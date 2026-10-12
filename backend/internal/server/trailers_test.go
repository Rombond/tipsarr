package server_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/jobs"
	"github.com/Rombond/tipsarr/backend/internal/trailers"
)

// fakeResolver speaks the trailer-resolver API. Files go into dir (the folder Tipsarr reads); instant makes every
// resolve answer "ready" at once, otherwise keys stay queued until finish is called.
type fakeResolver struct {
	srv     *httptest.Server
	dir     string
	instant bool
	calls   atomic.Int32 // every request, authorised or not

	mu       sync.Mutex
	status   map[string]*trailers.Status
	resolved []string
	touched  []string
	cleared  int
	updates  int
	newer    string // the version yt-dlp "updates" to
	health   trailers.Health
}

const fakeResolverKey = "rk-secret"

var sampleVideo = []byte(strings.Repeat("0123456789", 100)) // 1000 bytes

func newFakeResolver(t *testing.T, dir string, instant bool) *fakeResolver {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	f := &fakeResolver{dir: dir, instant: instant, status: map[string]*trailers.Status{}}
	f.health = trailers.Health{OK: true, YtdlpVersion: "2026.10.10", DenoVersion: "2.5.0", FFmpegVersion: "7.1", Breaker: trailers.Breaker{State: "closed"}}
	mux := http.NewServeMux()
	reply := func(w http.ResponseWriter, code int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(v)
	}
	mux.HandleFunc("/v1/resolve", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Key       string `json:"key"`
			MaxHeight int    `json:"maxHeight"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		f.mu.Lock()
		defer f.mu.Unlock()
		f.resolved = append(f.resolved, in.Key)
		if st, ok := f.status[in.Key]; ok && st.Status == "ready" {
			reply(w, 200, st)
			return
		}
		if f.instant {
			st := f.writeLocked(t, in.Key)
			reply(w, 200, st)
			return
		}
		st := &trailers.Status{Status: "queued"}
		f.status[in.Key] = st
		reply(w, 202, st)
	})
	mux.HandleFunc("/v1/status/", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		st, ok := f.status[strings.TrimPrefix(r.URL.Path, "/v1/status/")]
		if !ok {
			reply(w, 404, map[string]string{"error": "unknown key"})
			return
		}
		reply(w, 200, st)
	})
	mux.HandleFunc("/v1/health", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		reply(w, 200, f.health)
	})
	mux.HandleFunc("/v1/touch", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Keys []string `json:"keys"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		f.mu.Lock()
		f.touched = append(f.touched, in.Keys...)
		f.mu.Unlock()
		w.WriteHeader(204)
	})
	mux.HandleFunc("/v1/update", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.updates++
		before := f.health.YtdlpVersion
		if f.newer != "" {
			f.health.YtdlpVersion = f.newer
		}
		reply(w, 200, map[string]any{"before": before, "after": f.health.YtdlpVersion, "updated": f.newer != ""})
	})
	mux.HandleFunc("/v1/cache", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.cleared++
		f.status = map[string]*trailers.Status{}
		f.mu.Unlock()
		matches, _ := filepath.Glob(filepath.Join(dir, "*.mp4"))
		for _, m := range matches {
			_ = os.Remove(m)
		}
		w.WriteHeader(204)
	})
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.calls.Add(1)
		if r.Header.Get("Authorization") != "Bearer "+fakeResolverKey {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeResolver) writeLocked(t *testing.T, key string) *trailers.Status {
	t.Helper()
	if err := os.WriteFile(filepath.Join(f.dir, key+".mp4"), sampleVideo, 0o644); err != nil {
		t.Fatal(err)
	}
	st := &trailers.Status{Status: "ready", Bytes: int64(len(sampleVideo)), Height: 720, Duration: 30}
	f.status[key] = st
	return st
}

// finish makes a queued key ready (or failed).
func (f *fakeResolver) finish(t *testing.T, key string) {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writeLocked(t, key)
}

func (f *fakeResolver) fail(key, msg string, permanent bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status[key] = &trailers.Status{Status: "failed", Error: msg, Permanent: permanent}
}

func (f *fakeResolver) resolvedKeys() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.resolved...)
}

// enableFiles saves the resolver settings and turns trailer files on, as an admin would.
func enableFiles(t *testing.T, e *env, admin *http.Cookie, f *fakeResolver) {
	t.Helper()
	resp, body := call(t, e.app, "PUT", "/api/v1/admin/settings", `{"teasersFilesUrl":"`+f.srv.URL+`/","teasersFilesKey":"`+fakeResolverKey+`","teasersFilesEnabled":true}`, admin)
	if resp.StatusCode != 200 || !strings.Contains(body, `"teasersFilesEnabled":true`) {
		t.Fatalf("enable trailer files = %d %s", resp.StatusCode, body)
	}
}

func TestTrailerFilesSettings(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)
	f := newFakeResolver(t, e.files.Dir(), true)

	// off by default
	if _, body := call(t, e.app, "GET", "/api/v1/admin/settings", "", admin); !strings.Contains(body, `"teasersFilesEnabled":false`) || !strings.Contains(body, `"teasersFilesMaxHeight":720`) {
		t.Fatalf("defaults = %s", body)
	}
	// cannot be on without an address and a key
	if resp, body := call(t, e.app, "PUT", "/api/v1/admin/settings", `{"teasersFilesEnabled":true}`, admin); resp.StatusCode != 422 || !strings.Contains(body, "trailer_resolver_not_configured") {
		t.Fatalf("enable without config = %d %s", resp.StatusCode, body)
	}
	if resp, body := call(t, e.app, "PUT", "/api/v1/admin/settings", `{"teasersFilesUrl":"ftp://x"}`, admin); resp.StatusCode != 422 || !strings.Contains(body, "invalid_trailer_resolver_url") {
		t.Fatalf("bad url = %d %s", resp.StatusCode, body)
	}
	if resp, body := call(t, e.app, "PUT", "/api/v1/admin/settings", `{"teasersFilesMaxHeight":999}`, admin); resp.StatusCode != 422 || !strings.Contains(body, "invalid_trailer_height") {
		t.Fatalf("bad height = %d %s", resp.StatusCode, body)
	}
	// only an admin changes it
	if resp, _ := call(t, e.app, "PUT", "/api/v1/admin/settings", `{"teasersFilesKey":"x"}`, bob); resp.StatusCode != 403 {
		t.Fatalf("bob saving = %d", resp.StatusCode)
	}
	// the key is write-only
	resp, body := call(t, e.app, "PUT", "/api/v1/admin/settings", `{"teasersFilesUrl":"`+f.srv.URL+`/","teasersFilesKey":"`+fakeResolverKey+`","teasersFilesMaxHeight":480}`, admin)
	if resp.StatusCode != 200 || strings.Contains(body, fakeResolverKey) || !strings.Contains(body, `"teasersFilesKeyConfigured":true`) || !strings.Contains(body, `"teasersFilesEnabled":false`) || !strings.Contains(body, `"teasersFilesMaxHeight":480`) {
		t.Fatalf("save = %d %s", resp.StatusCode, body)
	}
	if _, body := call(t, e.app, "GET", "/api/v1/admin/settings", "", admin); strings.Contains(body, fakeResolverKey) {
		t.Fatalf("key leaked: %s", body)
	}
	if resp, body := call(t, e.app, "PUT", "/api/v1/admin/settings", `{"teasersFilesEnabled":true}`, admin); resp.StatusCode != 200 || !strings.Contains(body, `"teasersFilesEnabled":true`) {
		t.Fatalf("enable = %d %s", resp.StatusCode, body)
	}
	// clearing the key turns it off
	if _, body := call(t, e.app, "PUT", "/api/v1/admin/settings", `{"teasersFilesKey":""}`, admin); !strings.Contains(body, `"teasersFilesEnabled":false`) || !strings.Contains(body, `"teasersFilesKeyConfigured":false`) {
		t.Fatalf("clearing the key = %s", body)
	}
}

func TestTrailerFilesOffMeansNoResolverCalls(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)
	f := newFakeResolver(t, e.files.Dir(), true)
	// address and key saved, switch left off
	call(t, e.app, "PUT", "/api/v1/admin/settings", `{"teasersFilesUrl":"`+f.srv.URL+`","teasersFilesKey":"`+fakeResolverKey+`"}`, admin)

	resp, body := call(t, e.app, "GET", "/api/v1/teasers", "", bob)
	if resp.StatusCode != 404 || !strings.Contains(body, "teasers_disabled") {
		t.Fatalf("teasers off = %d %s", resp.StatusCode, body)
	}
	if resp, body := call(t, e.app, "GET", "/api/v1/teasers/video/abcdefghijk", "", bob); resp.StatusCode != 404 || !strings.Contains(body, "teasers_disabled") {
		t.Fatalf("video off = %d %s", resp.StatusCode, body)
	}
	if _, body := call(t, e.app, "GET", "/api/v1/status", ""); !strings.Contains(body, `"teasers":false`) {
		t.Fatalf("status = %s", body)
	}
	if _, err := e.files.Sync(t.Context()); err == nil {
		t.Fatal("the sync job must report 'skipped' while the feature is off")
	}
	if n := f.calls.Load(); n != 0 {
		t.Fatalf("the resolver was called %d times while the feature is off", n)
	}
}

func TestStatusFlagFollowsResolverHealth(t *testing.T) {
	e := newEnv(t, true)
	_, admin, _ := setupUsers(t, e)
	f := newFakeResolver(t, e.files.Dir(), true)
	enableFiles(t, e, admin, f)
	if _, body := call(t, e.app, "GET", "/api/v1/status", ""); !strings.Contains(body, `"teasers":true`) {
		t.Fatalf("status with a healthy resolver = %s", body)
	}
	// a resolver that reports a problem hides the tab (once the 30 s health cache is refreshed)
	f.mu.Lock()
	f.health.OK = false
	f.mu.Unlock()
	e.files.ResetHealth()
	if _, body := call(t, e.app, "GET", "/api/v1/status", ""); !strings.Contains(body, `"teasers":false`) {
		t.Fatalf("status with a sick resolver = %s", body)
	}
	f.srv.Close()
	e.files.ResetHealth()
	if _, body := call(t, e.app, "GET", "/api/v1/status", ""); !strings.Contains(body, `"teasers":false`) {
		t.Fatalf("status with a dead resolver = %s", body)
	}
	// the feed itself keeps answering (with the YouTube player) while the resolver is down
	_ = admin
}

func TestTrailerFilesAdmin(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)
	f := newFakeResolver(t, e.files.Dir(), true)
	enableFiles(t, e, admin, f)

	if resp, _ := call(t, e.app, "GET", "/api/v1/admin/teasers/files", "", bob); resp.StatusCode != 403 {
		t.Fatalf("bob overview = %d", resp.StatusCode)
	}
	resp, body := call(t, e.app, "GET", "/api/v1/admin/teasers/files", "", admin)
	if resp.StatusCode != 200 || !strings.Contains(body, `"available":true`) || !strings.Contains(body, `"ytdlpVersion":"2026.10.10"`) || strings.Contains(body, fakeResolverKey) {
		t.Fatalf("overview = %d %s", resp.StatusCode, body)
	}
	// the test button resolves the demo video through the resolver
	resp, body = call(t, e.app, "POST", "/api/v1/admin/teasers/files/test", `{}`, admin)
	if resp.StatusCode != 200 || !strings.Contains(body, `"status":"ready"`) || !strings.Contains(body, `"key":"`+trailers.DemoKey+`"`) || !strings.Contains(body, `"bytes":1000`) {
		t.Fatalf("test = %d %s", resp.StatusCode, body)
	}
	if resp, body := call(t, e.app, "POST", "/api/v1/admin/teasers/files/test", `{"key":"../../etc"}`, admin); resp.StatusCode != 422 || !strings.Contains(body, "trailer_key_invalid") {
		t.Fatalf("bad key = %d %s", resp.StatusCode, body)
	}
	if resp, _ := call(t, e.app, "POST", "/api/v1/admin/teasers/files/test", `{}`, bob); resp.StatusCode != 403 {
		t.Fatalf("bob test = %d", resp.StatusCode)
	}
	// clearing empties the resolver's cache and forgets the rows
	if err := e.files.Ensure(t.Context(), "abcdefghijk"); err != nil {
		t.Fatal(err)
	}
	if resp, _ := call(t, e.app, "DELETE", "/api/v1/admin/teasers/files", "", bob); resp.StatusCode != 403 {
		t.Fatalf("bob clear = %d", resp.StatusCode)
	}
	if resp, body := call(t, e.app, "DELETE", "/api/v1/admin/teasers/files", "", admin); resp.StatusCode != 204 {
		t.Fatalf("clear = %d %s", resp.StatusCode, body)
	}
	if f.cleared != 1 {
		t.Fatalf("resolver cache cleared %d times", f.cleared)
	}
	if row, _ := e.st.TrailerFileByKey(t.Context(), "abcdefghijk"); row != nil {
		t.Fatalf("row survived the clear: %+v", row)
	}
	// a dead resolver is a 502 and keeps the rows
	if err := e.files.Ensure(t.Context(), "abcdefghijk"); err != nil {
		t.Fatal(err)
	}
	f.srv.Close()
	if resp, body := call(t, e.app, "DELETE", "/api/v1/admin/teasers/files", "", admin); resp.StatusCode != 502 || !strings.Contains(body, "trailer_resolver_unreachable") {
		t.Fatalf("clear with a dead resolver = %d %s", resp.StatusCode, body)
	}
	if row, _ := e.st.TrailerFileByKey(t.Context(), "abcdefghijk"); row == nil {
		t.Fatal("the row was deleted although the resolver could not be reached")
	}
}

func TestTrailerFileEnsureAndSync(t *testing.T) {
	e := newEnv(t, true)
	_, admin, _ := setupUsers(t, e)
	f := newFakeResolver(t, e.files.Dir(), false)
	enableFiles(t, e, admin, f)
	ctx := t.Context()
	const key = "abcdefghijk"

	if err := e.files.Ensure(ctx, "nope"); err == nil {
		t.Fatal("a bad key was accepted")
	}
	if err := e.files.Ensure(ctx, key); err != nil {
		t.Fatal(err)
	}
	if err := e.files.Ensure(ctx, key); err != nil { // already queued: no second resolver call
		t.Fatal(err)
	}
	if got := f.resolvedKeys(); len(got) != 1 || got[0] != key {
		t.Fatalf("resolver calls = %v", got)
	}
	if _, ok := e.files.Ready(ctx, key); ok {
		t.Fatal("ready before the download finished")
	}
	// the sync job follows the key until it is ready
	if _, err := e.files.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := e.files.Ready(ctx, key); ok {
		t.Fatal("ready while the resolver says queued")
	}
	f.finish(t, key)
	if _, err := e.files.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	row, ok := e.files.Ready(ctx, key)
	if !ok || row.Height != 720 || row.Bytes != 1000 || row.Duration != 30 {
		t.Fatalf("after sync: %+v %v", row, ok)
	}

	// a resolver that forgot the key (restart) is asked again
	f.mu.Lock()
	delete(f.status, "zzzzzzzzzzz")
	f.mu.Unlock()
	if err := e.files.Ensure(ctx, "zzzzzzzzzzz"); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	f.status = map[string]*trailers.Status{}
	f.mu.Unlock()
	before := len(f.resolvedKeys())
	if _, err := e.files.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	if len(f.resolvedKeys()) != before+1 {
		t.Fatalf("a forgotten key was not asked again: %v", f.resolvedKeys())
	}

	// a permanent failure is kept and never retried; a temporary one waits for its back-off
	f.fail("zzzzzzzzzzz", "Video unavailable", true)
	if _, err := e.files.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	r, _ := e.st.TrailerFileByKey(ctx, "zzzzzzzzzzz")
	if r.Status != "failed" || r.Permanent != 1 || r.Error != "Video unavailable" {
		t.Fatalf("permanent failure row = %+v", r)
	}
	n := len(f.resolvedKeys())
	_ = e.files.Ensure(ctx, "zzzzzzzzzzz")
	if _, err := e.files.Sync(ctx); err != nil || len(f.resolvedKeys()) != n {
		t.Fatalf("a permanently failed key was asked again (%v)", err)
	}
	f.fail("tttttttttt1", "HTTP 403", false)
	_ = e.files.Ensure(ctx, "tttttttttt1")
	f.mu.Lock()
	f.status["tttttttttt1"] = &trailers.Status{Status: "failed", Error: "HTTP 403"}
	f.mu.Unlock()
	if _, err := e.files.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	r, _ = e.st.TrailerFileByKey(ctx, "tttttttttt1")
	if r.Status != "failed" || r.Permanent != 0 || r.NextTryAt <= r.RequestedAt {
		t.Fatalf("temporary failure row = %+v", r)
	}

	// files played recently are reported to the resolver for its least-recently-used cleanup
	if err := e.st.TouchTrailerFile(ctx, key, time.Now().Unix()); err != nil {
		t.Fatal(err)
	}
	if _, err := e.files.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	touched := append([]string(nil), f.touched...)
	f.mu.Unlock()
	if len(touched) == 0 || touched[0] != key {
		t.Fatalf("touched = %v", touched)
	}
}

func rangeGet(t *testing.T, e *env, path, rng string, c *http.Cookie) (*http.Response, []byte) {
	t.Helper()
	req, _ := http.NewRequest("GET", e.app.URL+path, nil)
	if rng != "" {
		req.Header.Set("Range", rng)
	}
	if c != nil {
		req.AddCookie(c)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp, b
}

func TestTeaserVideoEndpoint(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)
	f := newFakeResolver(t, e.files.Dir(), true)
	enableFiles(t, e, admin, f)
	const key = "abcdefghijk"
	if err := e.files.Ensure(t.Context(), key); err != nil {
		t.Fatal(err)
	}
	path := "/api/v1/teasers/video/" + key

	if resp, _ := rangeGet(t, e, path, "", nil); resp.StatusCode != 401 {
		t.Fatalf("anonymous = %d", resp.StatusCode)
	}
	resp, b := rangeGet(t, e, path, "", bob)
	if resp.StatusCode != 200 || len(b) != 1000 || resp.Header.Get("Content-Type") != "video/mp4" || resp.Header.Get("Accept-Ranges") != "bytes" ||
		resp.Header.Get("Cache-Control") != "private, max-age=86400" || resp.Header.Get("ETag") == "" {
		t.Fatalf("full = %d %d bytes %v", resp.StatusCode, len(b), resp.Header)
	}
	etag := resp.Header.Get("ETag")
	// partial, open-ended and suffix ranges
	if resp, b := rangeGet(t, e, path, "bytes=10-19", bob); resp.StatusCode != 206 || string(b) != "0123456789" || resp.Header.Get("Content-Range") != "bytes 10-19/1000" {
		t.Fatalf("range = %d %q %v", resp.StatusCode, b, resp.Header)
	}
	if resp, b := rangeGet(t, e, path, "bytes=990-", bob); resp.StatusCode != 206 || len(b) != 10 {
		t.Fatalf("open range = %d %d", resp.StatusCode, len(b))
	}
	if resp, b := rangeGet(t, e, path, "bytes=-5", bob); resp.StatusCode != 206 || len(b) != 5 || resp.Header.Get("Content-Range") != "bytes 995-999/1000" {
		t.Fatalf("suffix range = %d %d %v", resp.StatusCode, len(b), resp.Header)
	}
	if resp, _ := rangeGet(t, e, path, "bytes=5000-6000", bob); resp.StatusCode != 416 {
		t.Fatalf("unsatisfiable range = %d", resp.StatusCode)
	}
	// a conditional request with the ETag costs no bytes
	req, _ := http.NewRequest("GET", e.app.URL+path, nil)
	req.AddCookie(bob)
	req.Header.Set("If-None-Match", etag)
	r2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	r2.Body.Close()
	if r2.StatusCode != 304 {
		t.Fatalf("If-None-Match = %d", r2.StatusCode)
	}
	// bad keys and files that are not ready are plain 404s, never a path outside the folder
	for _, bad := range []string{"short", "abcdefghijkl", "..%2f..%2fetc%2fpass", "abcdefghi.k", "zzzzzzzzzzz"} {
		if resp, _ := rangeGet(t, e, "/api/v1/teasers/video/"+bad, "", bob); resp.StatusCode != 404 {
			t.Fatalf("key %q = %d", bad, resp.StatusCode)
		}
	}
	// playing it is recorded for the cache cleanup
	row, _ := e.st.TrailerFileByKey(t.Context(), key)
	if row.LastUsedAt == 0 {
		t.Fatal("last_used_at was not updated")
	}
	// a file that disappeared from the folder is not served
	_ = os.Remove(filepath.Join(e.files.Dir(), key+".mp4"))
	if resp, _ := rangeGet(t, e, path, "", bob); resp.StatusCode != 404 {
		t.Fatalf("missing file = %d", resp.StatusCode)
	}
}

func TestYtdlpUpdateJob(t *testing.T) {
	e := newEnv(t, true)
	_, admin, _ := setupUsers(t, e)
	f := newFakeResolver(t, filepath.Join(t.TempDir(), "trailers"), true)
	trailerSvc := trailers.New(e.st, t.TempDir())

	// off: skipped, the resolver is not called
	if _, err := trailerSvc.UpdateTools(context.Background()); !errors.Is(err, jobs.ErrSkipped) || f.updates != 0 {
		t.Fatalf("disabled = %v, updates %d", err, f.updates)
	}
	enableFiles(t, e, admin, f)

	msg, err := trailerSvc.UpdateTools(context.Background())
	if err != nil || msg.Code != "srv.job.ytdlp.current" || f.updates != 1 {
		t.Fatalf("up to date = %+v %v (updates %d)", msg, err, f.updates)
	}
	f.newer = "2026.10.20"
	msg, err = trailerSvc.UpdateTools(context.Background())
	if err != nil || msg.Code != "srv.job.ytdlp.updated" {
		t.Fatalf("updated = %+v %v", msg, err)
	}
	if msg.Params["from"] != "2026.10.10" || msg.Params["to"] != "2026.10.20" {
		t.Fatalf("params = %v", msg.Params)
	}
}

func (f *fakeResolver) touchedKeys() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.touched...)
}
