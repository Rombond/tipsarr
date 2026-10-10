package push

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/notify"
	"github.com/Rombond/tipsarr/backend/internal/store"
)

type relay struct {
	srv   *httptest.Server
	mu    sync.Mutex
	auth  []string
	items []item
	fail  int               // answer 503 this many times first
	reply map[string]string // token -> status, default sent
}

func newRelay(t *testing.T) *relay {
	t.Helper()
	r := &relay{reply: map[string]string{}}
	r.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.mu.Lock()
		defer r.mu.Unlock()
		if req.URL.Path != "/v1/push/batch" {
			t.Errorf("path %s", req.URL.Path)
		}
		if r.fail > 0 {
			r.fail--
			w.WriteHeader(503)
			return
		}
		r.auth = append(r.auth, req.Header.Get("Authorization"))
		var body struct{ Items []item }
		json.NewDecoder(req.Body).Decode(&body)
		out := make([]result, len(body.Items))
		for i, it := range body.Items {
			r.items = append(r.items, it)
			out[i] = result{Status: "sent"}
			if s := r.reply[it.Token]; s != "" {
				out[i].Status = s
			}
		}
		json.NewEncoder(w).Encode(map[string]any{"items": out})
	}))
	t.Cleanup(r.srv.Close)
	return r
}

func (r *relay) got() []item {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]item(nil), r.items...)
}

type fixture struct {
	st  *store.Store
	svc *Service
	rel *relay
}

func setup(t *testing.T, dryRun bool) *fixture {
	t.Helper()
	st, err := store.Open(context.Background(), "sqlite:"+filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	ctx := context.Background()
	st.EnsureUser(ctx, "alice", "alice", true)
	st.EnsureUser(ctx, "bob", "bob", false)
	st.EnsureUser(ctx, "carol", "carol", false)
	rel := newRelay(t)
	st.SetSetting(ctx, SettingEnabled, "true")
	st.SetSetting(ctx, SettingURL, rel.srv.URL+"/")
	st.SetSetting(ctx, SettingKey, "pr_secret")
	svc := New(st, dryRun)
	svc.SetRetryDelays(5*time.Millisecond, 5*time.Millisecond)
	return &fixture{st: st, svc: svc, rel: rel}
}

func (f *fixture) device(t *testing.T, user, session, platform, token, lang string, cats int) {
	t.Helper()
	if err := f.st.CreateSession(context.Background(), &store.Session{ID: session, UserID: user, ExpiresAt: time.Now().Add(time.Hour).Unix()}); err != nil {
		t.Fatal(err)
	}
	if err := f.st.SaveDevice(context.Background(), &store.Device{UserID: user, SessionID: session, Platform: platform, PushToken: token, Language: lang, Categories: cats}); err != nil {
		t.Fatal(err)
	}
}

func tokens(items []item) map[string]item {
	m := map[string]item{}
	for _, it := range items {
		m[it.Token] = it
	}
	return m
}

func TestTargeting(t *testing.T) {
	f := setup(t, false)
	f.device(t, "alice", "s1", "ios", "tok-alice", "en", store.PushAll)
	f.device(t, "bob", "s2", "android", "tok-bob", "fr", store.PushAll)
	f.device(t, "carol", "s3", "ios", "tok-carol", "en", store.PushAll)
	ctx := context.Background()

	pick := func(ev notify.PushEvent) map[string]bool {
		devs, err := f.svc.Targets(ctx, ev)
		if err != nil {
			t.Fatal(err)
		}
		m := map[string]bool{}
		for _, d := range devs {
			m[d.PushToken] = true
		}
		return m
	}
	// requester only
	if got := pick(notify.PushEvent{Name: "request.approved", Users: []string{"bob"}}); len(got) != 1 || !got["tok-bob"] {
		t.Fatalf("requester: %v", got)
	}
	// admins only (bob created it, so he is skipped, carol is not an admin)
	if got := pick(notify.PushEvent{Name: "request.created", Admins: true, Skip: "bob"}); len(got) != 1 || !got["tok-alice"] {
		t.Fatalf("admins: %v", got)
	}
	// reporter AND admins, no duplicates, the actor is skipped
	got := pick(notify.PushEvent{Name: "issue.comment", Users: []string{"bob"}, Admins: true, Skip: "alice"})
	if len(got) != 1 || !got["tok-bob"] {
		t.Fatalf("issue.comment: %v", got)
	}
	got = pick(notify.PushEvent{Name: "issue.comment", Users: []string{"bob"}, Admins: true})
	if len(got) != 2 || !got["tok-bob"] || !got["tok-alice"] {
		t.Fatalf("issue.comment all: %v", got)
	}
	// nobody to notify
	if got := pick(notify.PushEvent{Name: "request.approved"}); len(got) != 0 {
		t.Fatalf("empty: %v", got)
	}
	if _, err := f.svc.Targets(ctx, notify.PushEvent{Name: "nope"}); err == nil {
		t.Fatal("unknown event accepted")
	}
}

func TestCategoryTogglesAreHonoured(t *testing.T) {
	f := setup(t, false)
	f.device(t, "bob", "s1", "ios", "tok-1", "en", store.PushIssues) // issues only
	f.device(t, "carol", "s2", "ios", "tok-2", "en", 0)              // nothing
	ctx := context.Background()
	for _, tc := range []struct {
		ev   string
		want int
	}{{"request.approved", 0}, {"issue.comment", 1}} {
		devs, _ := f.svc.Targets(ctx, notify.PushEvent{Name: tc.ev, Users: []string{"bob", "carol"}})
		if len(devs) != tc.want {
			t.Errorf("%s: %d devices, want %d", tc.ev, len(devs), tc.want)
		}
	}
}

func TestSendBuildsBatchWithFixedShape(t *testing.T) {
	f := setup(t, false)
	f.device(t, "bob", "s1", "ios", "tok-ios", "fr", store.PushAll)
	f.device(t, "alice", "s2", "android", "tok-and", "en", store.PushAll)
	if err := f.svc.send(context.Background(), notify.PushEvent{Name: "request.available", ID: "req42", Users: []string{"bob", "alice"}}); err != nil {
		t.Fatal(err)
	}
	got := tokens(f.rel.got())
	if len(got) != 2 {
		t.Fatalf("%v", got)
	}
	if it := got["tok-ios"]; it.Platform != "ios" || it.Event != "request.available" || it.ID != "req42" || it.Lang != "fr" {
		t.Fatalf("%+v", it)
	}
	if it := got["tok-and"]; it.Platform != "android" || it.Lang != "en" {
		t.Fatalf("%+v", it)
	}
	if f.rel.auth[0] != "Bearer pr_secret" {
		t.Fatalf("auth %q", f.rel.auth[0])
	}
}

func TestInvalidTokenRemovesDevice(t *testing.T) {
	f := setup(t, false)
	f.device(t, "bob", "s1", "ios", "tok-dead", "en", store.PushAll)
	f.device(t, "bob", "s2", "android", "tok-live", "en", store.PushAll)
	f.rel.reply["tok-dead"] = "invalid_token"
	f.svc.send(context.Background(), notify.PushEvent{Name: "request.approved", ID: "1", Users: []string{"bob"}})
	devs, _ := f.st.DevicesFor(context.Background(), []string{"bob"}, false)
	if len(devs) != 1 || devs[0].PushToken != "tok-live" {
		t.Fatalf("devices left: %+v", devs)
	}
}

func TestRetriesWhenRelayIsDownThenGivesUp(t *testing.T) {
	f := setup(t, false)
	f.device(t, "bob", "s1", "ios", "tok", "en", store.PushAll)
	f.rel.fail = 2
	f.svc.send(context.Background(), notify.PushEvent{Name: "request.approved", ID: "1", Users: []string{"bob"}})
	if n := len(f.rel.got()); n != 1 {
		t.Fatalf("delivered after retries: %d", n)
	}
	f.rel.fail = 100
	f.svc.send(context.Background(), notify.PushEvent{Name: "request.approved", ID: "2", Users: []string{"bob"}})
	if n := len(f.rel.got()); n != 1 {
		t.Fatalf("delivered while relay is down: %d", n)
	}
	if devs, _ := f.st.DevicesFor(context.Background(), []string{"bob"}, false); len(devs) != 1 {
		t.Fatal("a relay outage must not drop devices")
	}
}

func TestOnlyFailedItemsAreRetried(t *testing.T) {
	f := setup(t, false)
	f.device(t, "bob", "s1", "ios", "tok-a", "en", store.PushAll)
	f.device(t, "carol", "s2", "ios", "tok-b", "en", store.PushAll)
	f.rel.reply["tok-b"] = "error"
	f.svc.send(context.Background(), notify.PushEvent{Name: "request.approved", ID: "1", Users: []string{"bob", "carol"}})
	count := map[string]int{}
	for _, it := range f.rel.got() {
		count[it.Token]++
	}
	if count["tok-a"] != 1 || count["tok-b"] != 3 {
		t.Fatalf("attempts %v", count)
	}
}

func TestOffByDefaultAndDryRun(t *testing.T) {
	f := setup(t, false)
	f.device(t, "bob", "s1", "ios", "tok", "en", store.PushAll)
	ctx := context.Background()
	f.st.SetSetting(ctx, SettingEnabled, "false")
	if f.svc.Configured(ctx) {
		t.Fatal("configured while off")
	}
	f.svc.send(ctx, notify.PushEvent{Name: "request.approved", ID: "1", Users: []string{"bob"}})
	if len(f.rel.got()) != 0 {
		t.Fatal("sent while off")
	}
	f.st.SetSetting(ctx, SettingEnabled, "true")
	f.svc.dryRun = true
	f.svc.send(ctx, notify.PushEvent{Name: "request.approved", ID: "1", Users: []string{"bob"}})
	if len(f.rel.got()) != 0 {
		t.Fatal("dry-run must never reach the relay")
	}
	f.svc.dryRun = false
	f.st.SetSetting(ctx, SettingKey, "")
	if f.svc.Configured(ctx) {
		t.Fatal("configured without a key")
	}
}

func TestSessionEndRemovesDevice(t *testing.T) {
	f := setup(t, false)
	ctx := context.Background()
	f.device(t, "bob", "s1", "ios", "tok-1", "en", store.PushAll)
	f.device(t, "bob", "s2", "android", "tok-2", "en", store.PushAll)
	f.device(t, "carol", "s3", "ios", "tok-3", "en", store.PushAll)
	f.st.DeleteSession(ctx, "s1")
	devs, _ := f.st.DevicesFor(ctx, []string{"bob"}, false)
	if len(devs) != 1 || devs[0].PushToken != "tok-2" {
		t.Fatalf("after DeleteSession: %+v", devs)
	}
	f.st.DeleteOtherUserSessions(ctx, "bob", "none")
	if devs, _ := f.st.DevicesFor(ctx, []string{"bob"}, false); len(devs) != 0 {
		t.Fatalf("after DeleteOtherUserSessions: %+v", devs)
	}
	f.st.DeleteUserSessions(ctx, "carol")
	if devs, _ := f.st.DevicesFor(ctx, []string{"carol"}, false); len(devs) != 0 {
		t.Fatalf("after DeleteUserSessions: %+v", devs)
	}
}

func TestSameTokenUnderNewSessionReplacesOldRow(t *testing.T) {
	f := setup(t, false)
	ctx := context.Background()
	f.device(t, "bob", "s1", "ios", "same", "en", store.PushAll)
	f.device(t, "bob", "s2", "ios", "same", "en", store.PushAll)
	if devs, _ := f.st.DevicesFor(ctx, []string{"bob"}, false); len(devs) != 1 || devs[0].SessionID != "s2" {
		t.Fatalf("%+v", devs)
	}
	// refreshing the same session updates in place
	f.st.SaveDevice(ctx, &store.Device{UserID: "bob", SessionID: "s2", Platform: "ios", PushToken: "rotated", Language: "fr", Categories: 1})
	devs, _ := f.st.DevicesFor(ctx, []string{"bob"}, false)
	if len(devs) != 1 || devs[0].PushToken != "rotated" || devs[0].Language != "fr" || devs[0].Categories != 1 {
		t.Fatalf("%+v", devs)
	}
}

func TestValidURL(t *testing.T) {
	for _, ok := range []string{"https://push.example.org", "http://10.0.0.2:8080"} {
		if ValidURL(ok) != nil {
			t.Errorf("%s refused", ok)
		}
	}
	for _, bad := range []string{"", "push.example.org", "ftp://x", "https://", "javascript:alert(1)"} {
		if ValidURL(bad) == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}
