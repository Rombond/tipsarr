package server_test

import (
	"bufio"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/Rombond/tipsarr/backend/internal/store"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/requests"
)

// fakeArr is a stand-in Radarr or Sonarr that records every call.
type fakeArr struct {
	kind        string
	srv         *httptest.Server
	mu          sync.Mutex
	calls       []string
	posted      map[string]any
	failPost    atomic.Bool
	hasFile     atomic.Bool
	rt          atomic.Int32 // the Rotten Tomatoes score Radarr reports (0 = none)
	downloading atomic.Bool
	moviesJSON  atomic.Value // string: body of GET /movie without a tmdbId filter
}

func newFakeArr(t *testing.T, kind string) *fakeArr {
	f := &fakeArr{kind: kind}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/api/v3")
		f.mu.Lock()
		f.calls = append(f.calls, r.Method+" "+p)
		f.mu.Unlock()
		if r.Header.Get("X-Api-Key") != "rk" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		out := func(s string) { _, _ = w.Write([]byte(s)) }
		switch {
		case p == "/system/status":
			out(`{"appName":"` + kind + `","version":"5.0"}`)
		case p == "/qualityprofile":
			out(`[{"id":4,"name":"HD-1080p"}]`)
		case p == "/rootfolder":
			out(`[{"id":1,"path":"/data/media","freeSpace":123}]`)
		case p == "/queue":
			if !f.downloading.Load() {
				out(`{"records":[],"totalRecords":0}`)
			} else if kind == "radarr" {
				out(`{"records":[{"movieId":42,"size":1000,"sizeleft":400,"timeleft":"00:05:00","status":"downloading"}],"totalRecords":1}`)
			} else {
				out(`{"records":[{"seriesId":77,"size":1000,"sizeleft":500,"timeleft":"00:10:00","status":"downloading"}],"totalRecords":1}`)
			}
		case kind == "radarr" && r.Method == http.MethodPost && p == "/movie":
			f.recordPost(r)
			if f.failPost.Load() {
				w.WriteHeader(http.StatusInternalServerError)
				out(`boom`)
				return
			}
			w.WriteHeader(http.StatusCreated)
			out(`{"id":42,"tmdbId":5}`)
		case kind == "radarr" && p == "/movie":
			if r.URL.Query().Get("tmdbId") == "" {
				if v, ok := f.moviesJSON.Load().(string); ok {
					out(v)
					return
				}
			}
			out(`[]`)
		case kind == "radarr" && p == "/movie/lookup/tmdb":
			out(`{"title":"Request Me","tmdbId":5,"year":2025,"imdbId":"tt1234567","ratings":{"imdb":{"votes":1200,"value":7.8,"type":"user"},"tmdb":{"votes":50,"value":7.1,"type":"user"},"metacritic":{"votes":0,"value":71,"type":"user"},"rottenTomatoes":{"votes":0,"value":` + strconv.Itoa(int(f.rt.Load())) + `,"type":"user"}}}`)
		case kind == "radarr" && p == "/movie/42":
			if f.hasFile.Load() {
				out(`{"id":42,"tmdbId":5,"hasFile":true}`)
			} else {
				out(`{"id":42,"tmdbId":5,"hasFile":false}`)
			}
		case kind == "sonarr" && r.Method == http.MethodPost && p == "/series":
			f.recordPost(r)
			w.WriteHeader(http.StatusCreated)
			out(`{"id":77,"tvdbId":81189}`)
		case kind == "sonarr" && p == "/series":
			out(`[]`)
		case kind == "sonarr" && p == "/series/lookup":
			out(`[{"title":"Show Two","tvdbId":81189,"seasons":[{"seasonNumber":0,"monitored":false},{"seasonNumber":1,"monitored":false},{"seasonNumber":2,"monitored":false}]}]`)
		case kind == "sonarr" && p == "/languageprofile":
			w.WriteHeader(http.StatusNotFound)
		case kind == "sonarr" && p == "/series/77":
			if f.hasFile.Load() {
				out(`{"id":77,"tvdbId":81189,"statistics":{"episodeFileCount":20,"episodeCount":20}}`)
			} else {
				out(`{"id":77,"tvdbId":81189,"statistics":{"episodeFileCount":3,"episodeCount":20}}`)
			}
		default:
			out(`{}`)
		}
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeArr) recordPost(r *http.Request) {
	var b map[string]any
	_ = json.NewDecoder(r.Body).Decode(&b)
	f.mu.Lock()
	f.posted = b
	f.mu.Unlock()
}

func (f *fakeArr) writes() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var w []string
	for _, c := range f.calls {
		if !strings.HasPrefix(c, "GET ") {
			w = append(w, c)
		}
	}
	return w
}

// setup logs alice (admin) and bob (user) in and saves the TMDB key.
func setupUsers(t *testing.T, e *env) (jfURL string, admin, bob *http.Cookie) {
	t.Helper()
	jf := fakeJellyfin(t)
	admin = loginAs(t, e.app, jf.URL, "alice", "secret")
	bob = loginAs(t, e.app, jf.URL, "bob", "hunter2")
	call(t, e.app, "PUT", "/api/v1/admin/settings", `{"tmdbApiKey":"k123"}`, admin)
	return jf.URL, admin, bob
}

func addInstance(t *testing.T, e *env, admin *http.Cookie, kind string, arr *fakeArr) {
	t.Helper()
	resp, body := call(t, e.app, "POST", "/api/v1/admin/servarr",
		`{"kind":"`+kind+`","name":"`+kind+`","url":"`+arr.srv.URL+`","apiKey":"rk","qualityProfileId":4,"rootFolder":"/data/media","isDefault":true}`, admin)
	if resp.StatusCode != http.StatusCreated || strings.Contains(body, `"rk"`) || !strings.Contains(body, `"apiKeyConfigured":true`) {
		t.Fatalf("create instance = %d %s", resp.StatusCode, body)
	}
}

type reqView struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Stage       string `json:"stage"`
	DryRun      bool   `json:"dryRun"`
	Error       string `json:"error"`
	Title       string `json:"title"`
	Seasons     []int  `json:"seasons"`
	RequestedBy struct {
		Name string `json:"name"`
	} `json:"requestedBy"`
	Progress *struct {
		Percent    int `json:"percent"`
		ETASeconds int `json:"etaSeconds"`
	} `json:"progress"`
}

func decodeReq(t *testing.T, body string) reqView {
	t.Helper()
	var v reqView
	if err := json.Unmarshal([]byte(body), &v); err != nil {
		t.Fatalf("decode %s: %v", body, err)
	}
	return v
}

func TestServarrAdmin(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)
	arr := newFakeArr(t, "radarr")

	if resp, _ := call(t, e.app, "GET", "/api/v1/admin/servarr", "", bob); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("user list = %d", resp.StatusCode)
	}
	// probe is read-only and returns choices for the form
	resp, body := call(t, e.app, "POST", "/api/v1/admin/servarr/probe", `{"kind":"radarr","url":"`+arr.srv.URL+`","apiKey":"rk"}`, admin)
	if resp.StatusCode != 200 || !strings.Contains(body, `"HD-1080p"`) || !strings.Contains(body, `/data/media`) {
		t.Fatalf("probe = %d %s", resp.StatusCode, body)
	}
	if resp, _ := call(t, e.app, "POST", "/api/v1/admin/servarr/probe", `{"kind":"radarr","url":"`+arr.srv.URL+`","apiKey":"bad"}`, admin); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("bad key probe = %d", resp.StatusCode)
	}
	addInstance(t, e, admin, "radarr", arr)
	_, body = call(t, e.app, "GET", "/api/v1/admin/servarr", "", admin)
	if !strings.Contains(body, `"isDefault":true`) || strings.Contains(body, `"rk"`) {
		t.Fatalf("list = %s", body)
	}
	if w := arr.writes(); len(w) != 0 {
		t.Fatalf("admin setup wrote to radarr: %v", w)
	}
}

func TestRequestFlowDryRun(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)
	radarr := newFakeArr(t, "radarr")
	addInstance(t, e, admin, "radarr", radarr)

	// a user requests a movie: pending, and the title now shows an active request
	resp, body := call(t, e.app, "POST", "/api/v1/requests", `{"type":"movie","tmdbId":5}`, bob)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create = %d %s", resp.StatusCode, body)
	}
	v := decodeReq(t, body)
	if v.Status != "pending" || v.Stage != "requested" || v.Title != "Request Me" || v.RequestedBy.Name != "bob" {
		t.Fatalf("created = %+v", v)
	}
	if resp, _ := call(t, e.app, "POST", "/api/v1/requests", `{"type":"movie","tmdbId":5}`, bob); resp.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate = %d", resp.StatusCode)
	}
	if _, b := call(t, e.app, "GET", "/api/v1/media/movie/5", "", bob); !strings.Contains(b, `"requestStatus":"pending"`) {
		t.Fatalf("detail should show the request: %s", b)
	}
	if resp, _ := call(t, e.app, "POST", "/api/v1/requests", `{"type":"movie","tmdbId":999999}`, bob); resp.StatusCode == http.StatusCreated {
		t.Fatal("unknown title must not be requestable")
	}

	// permissions
	if resp, _ := call(t, e.app, "POST", "/api/v1/requests/"+v.ID+"/approve", `{}`, bob); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("user approve = %d", resp.StatusCode)
	}
	// admin approves in dry-run: recorded, NOT sent
	resp, body = call(t, e.app, "POST", "/api/v1/requests/"+v.ID+"/approve", `{}`, admin)
	a := decodeReq(t, body)
	if resp.StatusCode != 200 || a.Status != "approved" || a.Stage != "approved" || !a.DryRun {
		t.Fatalf("approve = %d %s", resp.StatusCode, body)
	}
	if w := radarr.writes(); len(w) != 0 {
		t.Fatalf("DRY-RUN LEAK: radarr got writes %v", w)
	}
	// nothing is in flight, so the poller has nothing to do
	if n, err := e.reqs.Poll(context.Background()); n != 0 || err != nil {
		t.Fatalf("poll = %d %v", n, err)
	}

	// admin-created requests auto-approve, but without a Sonarr instance they stay pending
	resp, body = call(t, e.app, "POST", "/api/v1/requests", `{"type":"tv","tmdbId":2,"seasons":[1]}`, admin)
	tv := decodeReq(t, body)
	if resp.StatusCode != http.StatusCreated || tv.Status != "pending" || len(tv.Seasons) != 1 {
		t.Fatalf("admin tv without sonarr = %d %s", resp.StatusCode, body)
	}
	if resp, _ := call(t, e.app, "POST", "/api/v1/requests", `{"type":"tv","tmdbId":2,"seasons":[9]}`, bob); resp.StatusCode != http.StatusConflict && resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("bad season / duplicate = %d", resp.StatusCode)
	}
	// decline with reason
	resp, body = call(t, e.app, "POST", "/api/v1/requests/"+tv.ID+"/decline", `{"reason":"too big"}`, admin)
	if d := decodeReq(t, body); resp.StatusCode != 200 || d.Status != "declined" {
		t.Fatalf("decline = %d %s", resp.StatusCode, body)
	}

	// visibility: bob sees only his own; alice sees all; counts follow
	_, body = call(t, e.app, "GET", "/api/v1/requests", "", bob)
	if !strings.Contains(body, `"total":1`) {
		t.Fatalf("bob list = %s", body)
	}
	_, body = call(t, e.app, "GET", "/api/v1/requests", "", admin)
	if !strings.Contains(body, `"total":2`) {
		t.Fatalf("admin list = %s", body)
	}
	if resp, _ := call(t, e.app, "GET", "/api/v1/requests/"+tv.ID, "", bob); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("bob reading alice's request = %d", resp.StatusCode)
	}
	_, body = call(t, e.app, "GET", "/api/v1/requests/counts", "", admin)
	if !strings.Contains(body, `"approved":1`) || !strings.Contains(body, `"declined":1`) {
		t.Fatalf("counts = %s", body)
	}
	// owners cannot delete approved requests; admins can
	if resp, _ := call(t, e.app, "DELETE", "/api/v1/requests/"+v.ID, "", bob); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("owner delete approved = %d", resp.StatusCode)
	}
	if resp, _ := call(t, e.app, "DELETE", "/api/v1/requests/"+v.ID, "", admin); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("admin delete = %d", resp.StatusCode)
	}
	if w := radarr.writes(); len(w) != 0 {
		t.Fatalf("DRY-RUN LEAK at end: %v", w)
	}
}

func TestRequestFlowLive(t *testing.T) {
	e := newEnv(t, false)
	_, admin, bob := setupUsers(t, e)
	radarr, sonarr := newFakeArr(t, "radarr"), newFakeArr(t, "sonarr")
	addInstance(t, e, admin, "radarr", radarr)
	addInstance(t, e, admin, "sonarr", sonarr)
	ctx := context.Background()
	live, _, cancel := e.hub.Subscribe("", true, 0) // admin view of the event stream
	defer cancel()

	// failure first: Radarr rejects the add -> failed with the error, retry works
	radarr.failPost.Store(true)
	_, body := call(t, e.app, "POST", "/api/v1/requests", `{"type":"movie","tmdbId":5}`, bob)
	v := decodeReq(t, body)
	resp, body := call(t, e.app, "POST", "/api/v1/requests/"+v.ID+"/approve", `{}`, admin)
	if f := decodeReq(t, body); resp.StatusCode != 200 || f.Status != "failed" || !strings.Contains(f.Error, "boom") {
		t.Fatalf("failed approve = %d %s", resp.StatusCode, body)
	}
	radarr.failPost.Store(false)
	resp, body = call(t, e.app, "POST", "/api/v1/requests/"+v.ID+"/retry", ``, admin)
	a := decodeReq(t, body)
	if resp.StatusCode != 200 || a.Status != "approved" || a.DryRun || a.Stage != "searching" {
		t.Fatalf("retry = %d %s", resp.StatusCode, body)
	}
	if radarr.posted["qualityProfileId"] != float64(4) || radarr.posted["rootFolderPath"] != "/data/media" {
		t.Fatalf("radarr body = %v", radarr.posted)
	}

	// downloading: progress is read from the queue
	radarr.downloading.Store(true)
	if n, err := e.reqs.Poll(ctx); n != 1 || err != nil {
		t.Fatalf("poll = %d %v", n, err)
	}
	_, body = call(t, e.app, "GET", "/api/v1/requests/"+v.ID+"/progress", "", bob)
	d := decodeReq(t, body)
	if d.Stage != "downloading" || d.Progress == nil || d.Progress.Percent != 60 || d.Progress.ETASeconds != 300 {
		t.Fatalf("progress = %s", body)
	}
	// ... and the SSE hub saw it
	sawProgress := false
	for len(live) > 0 {
		if ev := <-live; ev.Type == "request.progress" {
			sawProgress = true
		}
	}
	if !sawProgress {
		t.Fatal("no request.progress event published")
	}

	// the file arrives: available
	radarr.hasFile.Store(true)
	radarr.downloading.Store(false)
	e.reqs.Poll(ctx)
	_, body = call(t, e.app, "GET", "/api/v1/requests/"+v.ID, "", bob)
	if f := decodeReq(t, body); f.Status != "available" || f.Stage != "available" || f.Progress != nil {
		t.Fatalf("finished = %s", body)
	}
	if n, _ := e.reqs.Poll(ctx); n != 0 {
		t.Fatalf("finished request still polled: %d", n)
	}

	// TV: only the requested seasons are monitored
	_, body = call(t, e.app, "POST", "/api/v1/requests", `{"type":"tv","tmdbId":2,"seasons":[2]}`, admin)
	tv := decodeReq(t, body)
	if tv.Status != "approved" || tv.Stage != "searching" {
		t.Fatalf("admin tv auto-approve = %s", body)
	}
	monitored := map[int]bool{}
	for _, s := range sonarr.posted["seasons"].([]any) {
		m := s.(map[string]any)
		monitored[int(m["seasonNumber"].(float64))] = m["monitored"].(bool)
	}
	if monitored[1] || !monitored[2] {
		t.Fatalf("sonarr seasons = %v", monitored)
	}
	sonarr.hasFile.Store(true)
	e.reqs.Poll(ctx)
	if _, b := call(t, e.app, "GET", "/api/v1/requests/"+tv.ID, "", admin); decodeReq(t, b).Status != "available" {
		t.Fatalf("tv not available: %s", b)
	}
	_ = requests.ErrDuplicate
}

func TestWebhookNotifications(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)

	var mu sync.Mutex
	var got []struct {
		event, sig string
		body       []byte
	}
	var fail atomic.Bool
	recv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if fail.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		mu.Lock()
		got = append(got, struct {
			event, sig string
			body       []byte
		}{r.Header.Get("X-Tipsarr-Event"), r.Header.Get("X-Tipsarr-Signature"), b})
		mu.Unlock()
	}))
	defer recv.Close()

	if resp, _ := call(t, e.app, "POST", "/api/v1/admin/webhooks", `{"name":"x","url":"`+recv.URL+`","events":["nope"],"enabled":true}`, admin); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("unknown event = %d", resp.StatusCode)
	}
	resp, body := call(t, e.app, "POST", "/api/v1/admin/webhooks", `{"name":"ntfy","url":"`+recv.URL+`","events":["request.created"],"enabled":true,"secret":"s3cret"}`, admin)
	var wh struct {
		ID               string `json:"id"`
		SecretConfigured bool   `json:"secretConfigured"`
	}
	_ = json.Unmarshal([]byte(body), &wh)
	if resp.StatusCode != http.StatusCreated || !wh.SecretConfigured || strings.Contains(body, "s3cret") {
		t.Fatalf("create webhook = %d %s", resp.StatusCode, body)
	}

	// subscribed event is delivered, signed, and flagged as dry-run
	call(t, e.app, "POST", "/api/v1/requests", `{"type":"movie","tmdbId":5}`, bob)
	waitFor(t, "webhook delivery", func() bool { mu.Lock(); defer mu.Unlock(); return len(got) == 1 })
	mu.Lock()
	g := got[0]
	mu.Unlock()
	mac := hmac.New(sha256.New, []byte("s3cret"))
	mac.Write(g.body)
	if g.event != "request.created" || g.sig != "sha256="+hex.EncodeToString(mac.Sum(nil)) {
		t.Fatalf("delivery = %s sig=%s", g.event, g.sig)
	}
	var env struct {
		Event  string `json:"event"`
		DryRun bool   `json:"dryRun"`
		Data   struct {
			Request struct{ Title string } `json:"request"`
		} `json:"data"`
	}
	_ = json.Unmarshal(g.body, &env)
	if env.Event != "request.created" || !env.DryRun || env.Data.Request.Title != "Request Me" {
		t.Fatalf("payload = %s", g.body)
	}

	// test endpoint: ok, then a failing receiver surfaces as 502
	if resp, _ := call(t, e.app, "POST", "/api/v1/admin/webhooks/"+wh.ID+"/test", "", admin); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("test = %d", resp.StatusCode)
	}
	fail.Store(true)
	if resp, _ := call(t, e.app, "POST", "/api/v1/admin/webhooks/"+wh.ID+"/test", "", admin); resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("failing test = %d", resp.StatusCode)
	}
}

func TestEventStream(t *testing.T) {
	e := newEnv(t, true)
	_, _, bob := setupUsers(t, e)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", e.app.URL+"/api/v1/events", nil)
	req.AddCookie(bob)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("stream = %d %s", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	if anon, _ := http.Get(e.app.URL + "/api/v1/events"); anon.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous stream = %d", anon.StatusCode)
	}

	lines := make(chan string, 20)
	go func() {
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			lines <- sc.Text()
		}
		close(lines)
	}()
	call(t, e.app, "POST", "/api/v1/requests", `{"type":"movie","tmdbId":5}`, bob)
	var sawEvent, sawData bool
	for !(sawEvent && sawData) {
		select {
		case l, ok := <-lines:
			if !ok {
				t.Fatal("stream closed early")
			}
			sawEvent = sawEvent || l == "event: request.updated"
			sawData = sawData || (strings.HasPrefix(l, "data: ") && strings.Contains(l, `"status":"pending"`))
		case <-ctx.Done():
			t.Fatal("timed out waiting for the request.updated event")
		}
	}
}

func TestRequestOptionsAndOverrides(t *testing.T) {
	e := newEnv(t, false)
	_, admin, bob := setupUsers(t, e)
	radarr := newFakeArr(t, "radarr")

	if resp, _ := call(t, e.app, "GET", "/api/v1/requests/options?type=movie", "", admin); resp.StatusCode != http.StatusConflict {
		t.Fatalf("options without an instance = %d", resp.StatusCode)
	}
	addInstance(t, e, admin, "radarr", radarr)
	// everyone chooses the quality profile; the folder choice is the admin's call (off by default)
	resp, body := call(t, e.app, "GET", "/api/v1/requests/options?type=movie", "", bob)
	if resp.StatusCode != 200 || !strings.Contains(body, `"HD-1080p"`) || strings.Contains(body, `/data/media`) || !strings.Contains(body, `"qualityProfileId":4`) {
		t.Fatalf("user options while folders are off = %d %s", resp.StatusCode, body)
	}
	if resp, _ := call(t, e.app, "POST", "/api/v1/requests", `{"type":"movie","tmdbId":5,"rootFolder":"/data/other"}`, bob); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("user folder while off = %d", resp.StatusCode)
	}
	if _, b := call(t, e.app, "GET", "/api/v1/status", ""); !strings.Contains(b, `"userFolderChoice":false`) {
		t.Fatalf("status = %s", b)
	}
	if resp, body := call(t, e.app, "GET", "/api/v1/requests/options?type=movie", "", admin); resp.StatusCode != 200 || !strings.Contains(body, `/data/media`) {
		t.Fatalf("admin options = %d %s", resp.StatusCode, body)
	}
	call(t, e.app, "PUT", "/api/v1/admin/settings", `{"userFolderChoice":true}`, admin)
	if _, b := call(t, e.app, "GET", "/api/v1/status", ""); !strings.Contains(b, `"userFolderChoice":true`) {
		t.Fatalf("status = %s", b)
	}
	if _, body = call(t, e.app, "GET", "/api/v1/requests/options?type=movie", "", bob); !strings.Contains(body, `/data/media`) {
		t.Fatalf("user options with folders on = %s", body)
	}
	// a user's choice is remembered and used when an admin approves
	resp, body = call(t, e.app, "POST", "/api/v1/requests", `{"type":"movie","tmdbId":5,"qualityProfileId":9,"rootFolder":"/data/other"}`, bob)
	v := decodeReq(t, body)
	if resp.StatusCode != http.StatusCreated || v.Status != "pending" || !strings.Contains(body, `"qualityProfileId":9`) {
		t.Fatalf("user create = %d %s", resp.StatusCode, body)
	}
	// ...and can be changed while it is pending, by the requester
	resp, body = call(t, e.app, "PATCH", "/api/v1/requests/"+v.ID, `{"qualityProfileId":4,"rootFolder":"/data/media"}`, bob)
	if resp.StatusCode != 200 || !strings.Contains(body, `"qualityProfileId":4`) {
		t.Fatalf("patch = %d %s", resp.StatusCode, body)
	}
	call(t, e.app, "PATCH", "/api/v1/requests/"+v.ID, `{"qualityProfileId":9,"rootFolder":"/data/other"}`, bob)
	if resp, _ := call(t, e.app, "POST", "/api/v1/requests/"+v.ID+"/approve", `{}`, admin); resp.StatusCode != 200 {
		t.Fatalf("approve = %d", resp.StatusCode)
	}
	if radarr.posted["qualityProfileId"] != float64(9) || radarr.posted["rootFolderPath"] != "/data/other" {
		t.Fatalf("radarr body = %v", radarr.posted)
	}
	// approved requests can no longer be edited
	if resp, _ := call(t, e.app, "PATCH", "/api/v1/requests/"+v.ID, `{"qualityProfileId":4}`, bob); resp.StatusCode != http.StatusConflict {
		t.Fatalf("patch after approval = %d", resp.StatusCode)
	}
	// deleting a request that is still waiting in Radarr removes it there too
	if resp, _ := call(t, e.app, "DELETE", "/api/v1/requests/"+v.ID, "", admin); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete = %d", resp.StatusCode)
	}
	found := false
	for _, w := range radarr.writes() {
		found = found || w == "DELETE /movie/42"
	}
	if !found {
		t.Fatalf("radarr writes = %v, want DELETE /movie/42", radarr.writes())
	}
}

func TestUserProfile(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)
	_, body := call(t, e.app, "GET", "/api/v1/me", "", bob)
	var me struct{ ID string }
	_ = json.Unmarshal([]byte(body), &me)
	if me.ID == "" {
		t.Fatalf("me = %s", body)
	}
	call(t, e.app, "POST", "/api/v1/requests", `{"type":"movie","tmdbId":5}`, bob)

	resp, body := call(t, e.app, "GET", "/api/v1/users/"+me.ID, "", bob)
	if resp.StatusCode != 200 || !strings.Contains(body, `"requests":1`) || !strings.Contains(body, `"pending":1`) || !strings.Contains(body, `"movies":1`) {
		t.Fatalf("own profile = %d %s", resp.StatusCode, body)
	}
	if resp, _ := call(t, e.app, "GET", "/api/v1/users/"+me.ID, "", admin); resp.StatusCode != 200 {
		t.Fatalf("admin reading a profile = %d", resp.StatusCode)
	}
	_, ab := call(t, e.app, "GET", "/api/v1/me", "", admin)
	var am struct{ ID string }
	_ = json.Unmarshal([]byte(ab), &am)
	if resp, _ := call(t, e.app, "GET", "/api/v1/users/"+am.ID, "", bob); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("user reading another profile = %d", resp.StatusCode)
	}
	if resp, _ := call(t, e.app, "GET", "/api/v1/users/nobody", "", admin); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown user = %d", resp.StatusCode)
	}
	// admins can filter the request list by user
	_, body = call(t, e.app, "GET", "/api/v1/requests?user="+am.ID, "", admin)
	if !strings.Contains(body, `"total":0`) {
		t.Fatalf("filtered list = %s", body)
	}
	_, body = call(t, e.app, "GET", "/api/v1/requests?user="+me.ID, "", admin)
	if !strings.Contains(body, `"total":1`) {
		t.Fatalf("filtered list = %s", body)
	}
}

func TestIssues(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)
	hook := make(chan string, 4)
	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		hook <- r.Header.Get("X-Tipsarr-Event") + " " + string(b)
	}))
	defer sink.Close()
	call(t, e.app, "POST", "/api/v1/admin/webhooks", `{"name":"h","url":"`+sink.URL+`","events":["issue.created","issue.resolved"],"enabled":true}`, admin)

	if resp, _ := call(t, e.app, "POST", "/api/v1/issues", `{"type":"movie","tmdbId":5,"kind":"video","message":""}`, bob); resp.StatusCode != 422 {
		t.Fatalf("empty message = %d", resp.StatusCode)
	}
	resp, body := call(t, e.app, "POST", "/api/v1/issues", `{"type":"movie","tmdbId":5,"kind":"subtitles","message":"French subtitles are out of sync"}`, bob)
	if resp.StatusCode != http.StatusCreated || !strings.Contains(body, `"status":"open"`) || !strings.Contains(body, "out of sync") {
		t.Fatalf("create = %d %s", resp.StatusCode, body)
	}
	var iss struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal([]byte(body), &iss)
	select {
	case got := <-hook:
		if !strings.HasPrefix(got, "issue.created ") || !strings.Contains(got, "Request Me") {
			t.Fatalf("webhook = %s", got)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no issue.created webhook")
	}

	// visibility: the reporter and admins only (bob cannot see an issue he did not file)
	_, body = call(t, e.app, "GET", "/api/v1/issues", "", bob)
	if !strings.Contains(body, `"total":1`) {
		t.Fatalf("reporter list = %s", body)
	}
	if _, b := call(t, e.app, "GET", "/api/v1/issues/counts", "", admin); !strings.Contains(b, `"open":1`) {
		t.Fatalf("counts = %s", b)
	}
	resp, body = call(t, e.app, "POST", "/api/v1/issues/"+iss.ID+"/comments", `{"message":"Looking into it"}`, admin)
	if resp.StatusCode != 200 || !strings.Contains(body, "Looking into it") || !strings.Contains(body, `"commentCount":2`) {
		t.Fatalf("comment = %d %s", resp.StatusCode, body)
	}
	resp, body = call(t, e.app, "POST", "/api/v1/issues/"+iss.ID+"/resolve", ``, admin)
	if resp.StatusCode != 200 || !strings.Contains(body, `"status":"resolved"`) {
		t.Fatalf("resolve = %d %s", resp.StatusCode, body)
	}
	if got := <-hook; !strings.HasPrefix(got, "issue.resolved ") {
		t.Fatalf("webhook = %s", got)
	}
	if resp, _ := call(t, e.app, "POST", "/api/v1/issues/"+iss.ID+"/reopen", ``, bob); resp.StatusCode != 200 {
		t.Fatalf("reporter reopen = %d", resp.StatusCode)
	}
	if resp, _ := call(t, e.app, "DELETE", "/api/v1/issues/"+iss.ID, "", bob); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("user delete = %d", resp.StatusCode)
	}
	if resp, _ := call(t, e.app, "DELETE", "/api/v1/issues/"+iss.ID, "", admin); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("admin delete = %d", resp.StatusCode)
	}
}

func TestImportFromServarr(t *testing.T) {
	e := newEnv(t, true)
	_, admin, _ := setupUsers(t, e)
	radarr := newFakeArr(t, "radarr")
	// TMDB 5 is monitored and missing; 6 is monitored with a file; 7 is not monitored
	radarr.moviesJSON.Store(`[{"id":42,"title":"Request Me","tmdbId":5,"hasFile":false,"monitored":true},` +
		`{"id":43,"title":"Done","tmdbId":6,"hasFile":true,"monitored":true},{"id":44,"title":"Off","tmdbId":7,"hasFile":false,"monitored":false}]`)
	addInstance(t, e, admin, "radarr", radarr)
	ctx := context.Background()

	msg, err := e.reqs.ImportFromServarr(ctx)
	if err != nil || msg != "1 imported" {
		t.Fatalf("import = %q %v", msg, err)
	}
	_, body := call(t, e.app, "GET", "/api/v1/requests", "", admin)
	if !strings.Contains(body, `"total":1`) || !strings.Contains(body, `"source":"radarr"`) || !strings.Contains(body, `"status":"approved"`) {
		t.Fatalf("list = %s", body)
	}
	// running again does not duplicate
	if msg, _ := e.reqs.ImportFromServarr(ctx); msg != "0 imported" {
		t.Fatalf("second import = %q", msg)
	}
	// the toggle turns it off
	call(t, e.app, "PUT", "/api/v1/admin/settings", `{"servarrAutoImport":false}`, admin)
	if msg, _ := e.reqs.ImportFromServarr(ctx); msg != "auto-import is off" {
		t.Fatalf("off = %q", msg)
	}
	if w := radarr.writes(); len(w) != 0 {
		t.Fatalf("import wrote to radarr: %v", w)
	}
}

// A profile without a saved language follows the language the app sends, in memory only.
func TestLanguageHint(t *testing.T) {
	e := newEnv(t, true)
	_, _, bob := setupUsers(t, e)
	get := func(path, lang string) string {
		req, _ := http.NewRequest("GET", e.app.URL+path, nil)
		req.AddCookie(bob)
		if lang != "" {
			req.Header.Set("X-Tipsarr-Language", lang)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return string(b)
	}
	// /me always shows the stored profile (language empty), hint or not
	if b := get("/api/v1/me", "fr-FR"); !strings.Contains(b, `"language":""`) {
		t.Fatalf("me with hint = %s", b)
	}
	// a PATCH that only changes the region must not save the hinted language
	req, _ := http.NewRequest("PATCH", e.app.URL+"/api/v1/me", strings.NewReader(`{"region":"FR"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tipsarr-Language", "fr-FR")
	req.AddCookie(bob)
	if resp, err := http.DefaultClient.Do(req); err != nil || resp.StatusCode != 200 {
		t.Fatalf("patch = %v %v", resp, err)
	}
	if b := get("/api/v1/me", ""); !strings.Contains(b, `"language":""`) || !strings.Contains(b, `"region":"FR"`) {
		t.Fatalf("me after patch = %s", b)
	}
}

func TestDefaultLanguage(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)
	if _, b := call(t, e.app, "GET", "/api/v1/status", ""); !strings.Contains(b, `"defaultLanguage":""`) {
		t.Fatalf("status = %s", b)
	}
	if resp, _ := call(t, e.app, "PUT", "/api/v1/admin/settings", `{"defaultLanguage":"french"}`, admin); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("bad language = %d", resp.StatusCode)
	}
	if resp, _ := call(t, e.app, "PUT", "/api/v1/admin/settings", `{"defaultLanguage":"fr-FR"}`, bob); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("user setting = %d", resp.StatusCode)
	}
	call(t, e.app, "PUT", "/api/v1/admin/settings", `{"defaultLanguage":"fr-FR"}`, admin)
	if _, b := call(t, e.app, "GET", "/api/v1/status", ""); !strings.Contains(b, `"defaultLanguage":"fr-FR"`) {
		t.Fatalf("status = %s", b)
	}
	// the default never overwrites what is saved on a profile
	if _, b := call(t, e.app, "GET", "/api/v1/me", "", bob); !strings.Contains(b, `"language":""`) {
		t.Fatalf("me = %s", b)
	}
}

// A person can list their available requests they never watched; an admin everyone's.
func TestUnwatchedRequests(t *testing.T) {
	jfBoxed.Store(false)
	e := newEnv(t, true)
	jfURL, admin, bob := setupUsers(t, e)
	_ = jfURL
	call(t, e.app, "PUT", "/api/v1/admin/settings", `{"jellyfinApiKey":"jfkey"}`, admin)
	adminCall := func(m, p string) (*http.Response, string) { return call(t, e.app, m, p, "", admin) }
	runJobAndWait(t, e.app, adminCall, "library-sync", "ok")
	runJobAndWait(t, e.app, adminCall, "history-sync", "ok") // alice watched Movie One (tmdb 1)

	ctx := context.Background()
	const aliceID, bobID = "aaaaaaaabbbbccccddddeeeeeeeeeeee", "bbbbbbbbbbbbccccddddeeeeeeeeeeee"
	for _, r := range []*store.Request{
		{ID: "r-bob", MediaType: "movie", TMDBID: 1, Title: "Movie One", RequestedBy: bobID, Status: store.StatusAvailable},     // bob never watched it
		{ID: "r-alice", MediaType: "movie", TMDBID: 1, Title: "Movie One", RequestedBy: aliceID, Status: store.StatusAvailable}, // alice did
		{ID: "r-pending", MediaType: "movie", TMDBID: 99, Title: "Pending", RequestedBy: bobID, Status: store.StatusPending},
		{ID: "r-import", MediaType: "movie", TMDBID: 98, Title: "Imported", RequestedBy: "", Status: store.StatusAvailable},
	} {
		if err := e.st.CreateRequest(ctx, r, nil); err != nil {
			t.Fatal(err)
		}
	}
	_, body := call(t, e.app, "GET", "/api/v1/requests?filter=unwatched", "", admin)
	if !strings.Contains(body, `"total":1`) || !strings.Contains(body, `"id":"r-bob"`) || strings.Contains(body, "r-alice") || strings.Contains(body, "r-import") {
		t.Fatalf("unwatched = %s", body)
	}
	// a person sees their own, and only their own
	_, body = call(t, e.app, "GET", "/api/v1/requests?filter=unwatched", "", bob)
	if !strings.Contains(body, `"total":1`) || !strings.Contains(body, `"id":"r-bob"`) {
		t.Fatalf("bob's own unwatched requests: %s", body)
	}
	_, body = call(t, e.app, "GET", "/api/v1/requests?filter=unwatched&user="+aliceID, "", admin)
	if !strings.Contains(body, `"total":0`) {
		t.Fatalf("admin looking at alice: %s", body)
	}
	_, body = call(t, e.app, "GET", "/api/v1/requests?filter=unwatched&user="+bobID, "", admin)
	if !strings.Contains(body, `"total":1`) {
		t.Fatalf("admin looking at bob: %s", body)
	}
	_, body = call(t, e.app, "GET", "/api/v1/requests/counts", "", admin)
	if strings.Contains(body, "unwatched") {
		t.Fatalf("the counts are about statuses only: %s", body)
	}
}
