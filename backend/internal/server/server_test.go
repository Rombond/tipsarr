package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/api"
	"github.com/Rombond/tipsarr/backend/internal/auth"
	"github.com/Rombond/tipsarr/backend/internal/avatars"
	"github.com/Rombond/tipsarr/backend/internal/boxoffice"
	"github.com/Rombond/tipsarr/backend/internal/events"
	"github.com/Rombond/tipsarr/backend/internal/issues"
	"github.com/Rombond/tipsarr/backend/internal/jobs"
	"github.com/Rombond/tipsarr/backend/internal/library"
	"github.com/Rombond/tipsarr/backend/internal/marks"
	"github.com/Rombond/tipsarr/backend/internal/media"
	"github.com/Rombond/tipsarr/backend/internal/notify"
	"github.com/Rombond/tipsarr/backend/internal/requests"
	"github.com/Rombond/tipsarr/backend/internal/server"
	"github.com/Rombond/tipsarr/backend/internal/store"
	"github.com/Rombond/tipsarr/backend/internal/suggestions"
)

// fakeJellyfin accepts alice/secret (admin) and bob/hunter2 (regular).
func fakeJellyfin(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/System/Info/Public", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ServerName":"Test","Version":"10.10.0","Id":"abc"}`))
	})
	mux.HandleFunc("POST /Users/AuthenticateByName", func(w http.ResponseWriter, r *http.Request) {
		var in struct{ Username, Pw string }
		_ = json.NewDecoder(r.Body).Decode(&in)
		switch {
		case in.Username == "alice" && in.Pw == "secret":
			_, _ = w.Write([]byte(`{"AccessToken":"t","User":{"Id":"aaaaaaaabbbbccccddddeeeeeeeeeeee","Name":"alice","Policy":{"IsAdministrator":true}}}`))
		case in.Username == "bob" && in.Pw == "hunter2":
			_, _ = w.Write([]byte(`{"AccessToken":"t","User":{"Id":"bbbbbbbbbbbbccccddddeeeeeeeeeeee","Name":"bob","Policy":{"IsAdministrator":false}}}`))
		default:
			w.WriteHeader(http.StatusUnauthorized)
		}
	})
	// ---- library / history (need the API key) ----
	const aliceID, bobID = "aaaaaaaabbbbccccddddeeeeeeeeeeee", "bbbbbbbbbbbbccccddddeeeeeeeeeeee"
	keyed := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.Header.Get("Authorization"), `Token="jfkey"`) || r.Header.Get("X-Emby-Token") != "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			h(w, r)
		}
	}
	writeItems := func(w http.ResponseWriter, items string) {
		_, _ = w.Write([]byte(`{"Items":[` + items + `],"TotalRecordCount":0}`))
	}
	mux.HandleFunc("/Users", keyed(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"Id":"` + aliceID + `","Name":"alice","Policy":{"IsAdministrator":true}},{"Id":"` + bobID + `","Name":"bob","Policy":{"IsAdministrator":false}},{"Id":"cccccccccccccccccccccccccccccccc","Name":"carol","Policy":{"IsAdministrator":false}}]`))
	}))
	mux.HandleFunc("GET /Plugins", keyed(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"Id":"p1","Name":"Webhook","Status":"Active"},{"Id":"958aad66378d4a2db89ba76f0ee1b06a","Name":"LDAP Authentication","Status":"Active"}]`))
	}))
	mux.HandleFunc("GET /Plugins/{id}/Configuration", keyed(func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("id") != "958aad66378d4a2db89ba76f0ee1b06a" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(`{"LdapServer":"127.0.0.1","LdapPort":1,"UseSsl":false,"LdapBindUser":"uid=svc,ou=people,dc=x,dc=com","LdapBindPassword":"pw","LdapBaseDn":"ou=people,dc=x,dc=com"}`))
	}))
	// one user: used to re-check a session's account (jfDisabled lists ids answering "disabled")
	mux.HandleFunc("GET /Users/{id}", keyed(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		names := map[string]string{aliceID: "alice", bobID: "bob", "cccccccccccccccccccccccccccccccc": "carol"}
		name, ok := names[id]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(`{"Id":"` + id + `","Name":"` + name + `","Policy":{"IsAdministrator":` + strconv.FormatBool(id == aliceID) + `,"IsDisabled":` + strconv.FormatBool(jfDisabled.Load() == id) + `}}`))
	}))
	mux.HandleFunc("/Items", keyed(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("IncludeItemTypes") {
		case "Movie,Series":
			writeItems(w, `{"Id":"jm1","Name":"Movie One","Type":"Movie","ProviderIds":{"Tmdb":"1"}},
				{"Id":"jm9","Name":"No Tmdb","Type":"Movie","ProviderIds":{}},
				{"Id":"js1","Name":"Show Two","Type":"Series","ProviderIds":{"Tmdb":"2"}}`)
		case "Episode":
			var eps []string
			for i := 1; i <= 10; i++ { // season 1 complete, season 2 absent
				eps = append(eps, `{"Id":"e`+strconv.Itoa(i)+`","Type":"Episode","SeriesId":"js1","ParentIndexNumber":1,"IndexNumber":`+strconv.Itoa(i)+`}`)
			}
			writeItems(w, strings.Join(eps, ","))
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	mux.HandleFunc("/Users/"+aliceID+"/Items", keyed(func(w http.ResponseWriter, r *http.Request) {
		writeItems(w, `{"Id":"jm1","Type":"Movie","ProviderIds":{"Tmdb":"1"},"UserData":{"Played":true,"PlayCount":2,"LastPlayedDate":"2026-09-01T10:00:00Z"}},
			{"Id":"e1","Type":"Episode","SeriesId":"js1","UserData":{"Played":true,"PlayCount":1,"LastPlayedDate":"2026-09-02T10:00:00Z"}},
			{"Id":"e2","Type":"Episode","SeriesId":"js1","UserData":{"Played":true,"PlayCount":1,"LastPlayedDate":"2026-09-03T10:00:00Z"}},
			{"Id":"eX","Type":"Episode","SeriesId":"unknown-series","UserData":{"Played":true,"PlayCount":1,"LastPlayedDate":"2026-09-03T10:00:00Z"}}`)
	}))
	mux.HandleFunc("/Users/cccccccccccccccccccccccccccccccc/Items", keyed(func(w http.ResponseWriter, r *http.Request) { writeItems(w, "") }))
	mux.HandleFunc("/Users/"+bobID+"/Items", keyed(func(w http.ResponseWriter, r *http.Request) { writeItems(w, "") }))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

var tmdbHits atomic.Int32

// jfDisabled is the id of the one fake Jellyfin user currently reported as disabled.
var jfDisabled atomic.Value

func init() { jfDisabled.Store("") }

// fakeTMDB serves just enough of the TMDB API for the handlers under test.
func fakeTMDB(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	hit := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			tmdbHits.Add(1)
			if r.URL.Query().Get("api_key") != "k123" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			h(w, r)
		}
	}
	mux.HandleFunc("/trending/all/week", hit(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"page":1,"total_pages":3,"results":[
			{"id":1,"media_type":"movie","title":"Movie One","release_date":"2024-01-02","poster_path":"/p1.jpg","vote_average":7.5},
			{"id":2,"media_type":"tv","name":"Show Two","first_air_date":"2023-05-06","poster_path":"/p2.jpg"},
			{"id":3,"media_type":"person","name":"Someone"}]}`))
	}))
	mux.HandleFunc("/movie/1", hit(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":1,"title":"Movie One","runtime":120,"genres":[{"id":28,"name":"Action"}],
			"imdb_id":"tt1","budget":63000000,"revenue":463517383,"production_companies":[{"name":"Warner Bros."},{"name":"Village Roadshow"}],"belongs_to_collection":{"id":9,"name":"Saga","backdrop_path":"/sagab.jpg"},
			"videos":{"results":[{"key":"abc123","site":"YouTube","type":"Trailer","official":false},{"key":"OFFICIAL1","site":"YouTube","type":"Trailer","official":true},{"key":"vim","site":"Vimeo","type":"Trailer","official":true}]},
			"credits":{"cast":[{"id":5,"name":"Ann","character":"Hero"}],"crew":[{"id":6,"name":"Dir","job":"Director"},{"id":7,"name":"X","job":"Editor"}]},
			"recommendations":{"results":[{"id":4,"title":"Rec"}]},"similar":{"results":[]}}`))
	}))
	mux.HandleFunc("/tv/2", hit(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":2,"name":"Show Two","episode_run_time":[45],"number_of_seasons":2,"number_of_episodes":20,
			"seasons":[{"season_number":1,"name":"Season 1","episode_count":10},{"season_number":2,"name":"Season 2","episode_count":10}],
			"external_ids":{"imdb_id":"tt2","tvdb_id":81189},"recommendations":{"results":[]},"similar":{"results":[]}}`))
	}))
	mux.HandleFunc("/search/multi", hit(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"page":1,"total_pages":1,"results":[
			{"id":1,"media_type":"movie","title":"Movie One"},{"id":8,"media_type":"person","name":"Zed","profile_path":"/z.jpg"}]}`))
	}))
	mux.HandleFunc("/movie/5", hit(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":5,"title":"Request Me","release_date":"2025-05-05","poster_path":"/rm.jpg","genres":[],
			"credits":{"cast":[],"crew":[]},"recommendations":{"results":[]},"similar":{"results":[]}}`))
	}))
	rec := func(kind string, ids ...int) string {
		var parts []string
		for _, id := range ids {
			title := "title"
			if kind == "tv" {
				title = "name"
			}
			parts = append(parts, `{"id":`+strconv.Itoa(id)+`,"`+title+`":"R`+strconv.Itoa(id)+`","poster_path":"/r`+strconv.Itoa(id)+`.jpg","vote_average":7}`)
		}
		return `{"page":1,"total_pages":1,"results":[` + strings.Join(parts, ",") + `]}`
	}
	mux.HandleFunc("/movie/1/recommendations", hit(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(rec("movie", 10, 11, 12, 13, 1))) }))
	mux.HandleFunc("/movie/1/similar", hit(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(rec("movie", 11, 15, 16))) }))
	mux.HandleFunc("/tv/2/recommendations", hit(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(rec("tv", 20, 21, 22, 23, 24))) }))
	mux.HandleFunc("/tv/2/similar", hit(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(rec("tv", 25))) }))
	mux.HandleFunc("/search/movie", hit(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("query") {
		case "Verity":
			_, _ = w.Write([]byte(`{"results":[{"id":900,"title":"Verity","release_date":"2026-10-02","poster_path":"/v.jpg","vote_average":7.1,"overview":"A thriller."}]}`))
		case "Resident Evil":
			_, _ = w.Write([]byte(`{"results":[{"id":901,"title":"Resident Evil","release_date":"2002-03-15","poster_path":"/re1.jpg"},{"id":902,"title":"Resident Evil","release_date":"2026-08-28","poster_path":"/re2.jpg"}]}`))
		case "Heart of the Beast":
			_, _ = w.Write([]byte(`{"results":[{"id":903,"title":"Heart of the Beast: Part One","release_date":"2026-09-18","poster_path":"/hb.jpg"}]}`))
		default:
			_, _ = w.Write([]byte(`{"results":[]}`))
		}
	}))
	mux.HandleFunc("/tv/30", hit(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":30,"name":"Anime Show","original_language":"ja","genres":[{"id":16,"name":"Animation"}],
			"number_of_seasons":1,"number_of_episodes":12,"seasons":[{"season_number":1,"name":"Season 1","episode_count":12}],
			"external_ids":{"tvdb_id":555},"recommendations":{"results":[]},"similar":{"results":[]}}`))
	}))
	for _, id := range []string{"10", "11", "12", "13", "15", "16"} {
		id := id
		mux.HandleFunc("/movie/"+id, hit(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"id":` + id + `,"title":"R` + id + `","poster_path":"/r` + id + `.jpg","genres":[],"credits":{"cast":[],"crew":[]},"recommendations":{"results":[]},"similar":{"results":[]}}`))
		}))
	}
	mux.HandleFunc("/movie/904", hit(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":904,"title":"Digger (2026)","release_date":"2026-10-02","poster_path":"/dg.jpg","vote_average":6.4,"overview":"Pinned by hand.",
			"genres":[],"credits":{"cast":[],"crew":[]},"recommendations":{"results":[]},"similar":{"results":[]}}`))
	}))
	mux.HandleFunc("/movie/404", hit(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) }))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func fakeImages(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/w185/AbCdEf123456.jpg" {
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write([]byte("JPEGDATA"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	return srv
}

var bomHits struct {
	sync.Mutex
	urls []string
}

// fakeBOM serves the real chart fixture for two weekends and an empty page for the rest.
func fakeBOM(t *testing.T) *httptest.Server {
	t.Helper()
	fixture, err := os.ReadFile("../boxoffice/testdata/weekend.html")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bomHits.Lock()
		bomHits.urls = append(bomHits.urls, r.URL.RequestURI())
		bomHits.Unlock()
		served := false
		for _, wk := range []string{"W40", "W39", "W38", "W37", "W36", "W35", "W34", "W33", "W32"} {
			served = served || strings.HasPrefix(r.URL.Path, "/weekend/2026"+wk+"/")
		}
		if served {
			_, _ = w.Write(fixture)
			return
		}
		_, _ = w.Write([]byte("<html><body>no data yet</body></html>"))
	}))
	t.Cleanup(srv.Close)
	return srv
}

type env struct {
	app      *httptest.Server
	st       *store.Store
	lib      *library.Service
	reqs     *requests.Service
	sugg     *suggestions.Service
	box      *boxoffice.Service
	hub      *events.Hub
	notifier *notify.Service
}

func newApp(t *testing.T) *httptest.Server {
	app, _, _ := newAppFull(t)
	return app
}

func newAppFull(t *testing.T) (*httptest.Server, *store.Store, *library.Service) {
	e := newEnv(t, true)
	return e.app, e.st, e.lib
}

func newEnv(t *testing.T, dryRun bool) *env {
	return newEnvWith(t, dryRun, "")
}

func newEnvWith(t *testing.T, dryRun bool, setupToken string) *env {
	t.Helper()
	tm, im := fakeTMDB(t), fakeImages(t)
	st, err := store.Open(context.Background(), "sqlite:"+filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	lib := library.New(st)
	lib.SetDebounce(20 * time.Millisecond)
	jm := jobs.New(st)
	jm.Register(jobs.Job{Name: "library-sync", Every: time.Hour, InitialDelay: time.Hour, Run: func(ctx context.Context) (string, error) {
		r, err := lib.SyncLibrary(ctx)
		return r.String(), err
	}})
	jm.Register(jobs.Job{Name: "history-sync", Every: time.Hour, InitialDelay: time.Hour, Run: func(ctx context.Context) (string, error) {
		r, err := lib.SyncHistory(ctx, "")
		return r.String(), err
	}})
	hub := events.New()
	mediaSvc := media.New(st, tm.URL)
	notifier := notify.New(st, dryRun)
	notifier.SetRetryDelays(10*time.Millisecond, 10*time.Millisecond)
	reqs := requests.New(st, mediaSvc, hub, notifier, dryRun)
	box := boxoffice.New(st, mediaSvc, fakeBOM(t).URL)
	box.SetPause(0)
	box.SetNow(func() time.Time { return time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC) }) // a Tuesday: latest weekend is 2026W40
	jm.Register(jobs.Job{Name: "boxoffice-refresh", Every: time.Hour, InitialDelay: time.Hour, Run: box.Refresh})
	sugg := suggestions.New(st, mediaSvc, hub)
	sugg.SetQueueDelay(20 * time.Millisecond)
	lib.OnHistoryChanged = sugg.QueueRefresh
	authSvc := auth.New(st)
	authSvc.SetRecheckEvery(0) // tests: ask the fake Jellyfin about the account on every request
	h, _ := server.New(api.Deps{
		Store: st, Auth: authSvc, Media: mediaSvc, Library: lib, Jobs: jm, Requests: reqs, Issues: issues.New(st, mediaSvc, hub, notifier), Avatars: avatars.New(t.TempDir(), st), Suggestions: sugg, BoxOffice: box, Marks: marks.New(st, mediaSvc), LoginLimiter: auth.NewLimiter(8, 10*time.Minute), SetupToken: setupToken, Hub: hub, Notify: notifier,
		DryRun: dryRun, ConfigDir: t.TempDir(), ImageBaseURL: im.URL,
	})
	app := httptest.NewServer(h)
	t.Cleanup(app.Close)
	return &env{app: app, st: st, lib: lib, reqs: reqs, sugg: sugg, box: box, hub: hub, notifier: notifier}
}

func call(t *testing.T, app *httptest.Server, method, path, body string, cookies ...*http.Cookie) (*http.Response, string) {
	t.Helper()
	req, _ := http.NewRequest(method, app.URL+path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var sb strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		sb.Write(buf[:n])
		if err != nil {
			break
		}
	}
	return resp, sb.String()
}

func sessionCookie(resp *http.Response) *http.Cookie {
	for _, c := range resp.Cookies() {
		if c.Name == auth.CookieName {
			return c
		}
	}
	return nil
}

func TestSetupLoginFlow(t *testing.T) {
	jf := fakeJellyfin(t)
	app := newApp(t)

	// not configured: login is unavailable, status says so
	if _, body := call(t, app, "GET", "/api/v1/setup/status", ""); !strings.Contains(body, `"configured":false`) {
		t.Fatalf("status = %s", body)
	}
	if resp, _ := call(t, app, "POST", "/api/v1/auth/login", `{"username":"alice","password":"secret"}`); resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("login before setup = %d", resp.StatusCode)
	}

	// bad Jellyfin URL is rejected, good one accepted, second setup refused
	if resp, _ := call(t, app, "POST", "/api/v1/setup", `{"jellyfinUrl":"http://127.0.0.1:1"}`); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("bad url setup = %d", resp.StatusCode)
	}
	if resp, body := call(t, app, "POST", "/api/v1/setup", `{"jellyfinUrl":"`+jf.URL+`"}`); resp.StatusCode != 200 {
		t.Fatalf("setup = %d %s", resp.StatusCode, body)
	}
	if resp, _ := call(t, app, "POST", "/api/v1/setup", `{"jellyfinUrl":"`+jf.URL+`"}`); resp.StatusCode != http.StatusConflict {
		t.Fatalf("second setup = %d", resp.StatusCode)
	}

	// wrong password
	if resp, _ := call(t, app, "POST", "/api/v1/auth/login", `{"username":"alice","password":"nope"}`); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bad login = %d", resp.StatusCode)
	}

	// /me needs a session
	if resp, _ := call(t, app, "GET", "/api/v1/me", ""); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous /me = %d", resp.StatusCode)
	}

	// admin login
	resp, body := call(t, app, "POST", "/api/v1/auth/login", `{"username":"alice","password":"secret"}`)
	c := sessionCookie(resp)
	if resp.StatusCode != 200 || c == nil || !c.HttpOnly {
		t.Fatalf("login = %d cookie=%v body=%s", resp.StatusCode, c, body)
	}
	if !strings.Contains(body, `"role":"admin"`) {
		t.Fatalf("expected admin role: %s", body)
	}
	if _, body := call(t, app, "GET", "/api/v1/me", "", c); !strings.Contains(body, `"name":"alice"`) {
		t.Fatalf("/me = %s", body)
	}

	// regular user stays a user
	resp, body = call(t, app, "POST", "/api/v1/auth/login", `{"username":"bob","password":"hunter2"}`)
	if !strings.Contains(body, `"role":"user"`) {
		t.Fatalf("bob = %s", body)
	}
	_ = resp

	// logout kills the session
	call(t, app, "POST", "/api/v1/auth/logout", "", c)
	if resp, _ := call(t, app, "GET", "/api/v1/me", "", c); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("/me after logout = %d", resp.StatusCode)
	}
}

func TestHealthAndOpenAPI(t *testing.T) {
	app := newApp(t)
	if resp, body := call(t, app, "GET", "/api/v1/health", ""); resp.StatusCode != 200 || !strings.Contains(body, `"ok":true`) {
		t.Fatalf("health = %d %s", resp.StatusCode, body)
	}
	if resp, body := call(t, app, "GET", "/api/v1/openapi.json", ""); resp.StatusCode != 200 || !strings.Contains(body, `"/auth/login"`) {
		t.Fatalf("openapi = %d %.200s", resp.StatusCode, body)
	}
	// unknown non-API path falls through to the SPA handler (404 hint when not built)
	if resp, _ := call(t, app, "GET", "/some/page", ""); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("spa fallback = %d", resp.StatusCode)
	}
}

// loginAs sets Jellyfin up (if needed) and returns a session cookie for the user.
func loginAs(t *testing.T, app *httptest.Server, jfURL, user, pass string) *http.Cookie {
	t.Helper()
	call(t, app, "POST", "/api/v1/setup", `{"jellyfinUrl":"`+jfURL+`"}`) // 409 on repeat is fine
	resp, body := call(t, app, "POST", "/api/v1/auth/login", `{"username":"`+user+`","password":"`+pass+`"}`)
	c := sessionCookie(resp)
	if c == nil {
		t.Fatalf("login %s failed: %d %s", user, resp.StatusCode, body)
	}
	return c
}

func TestMediaEndpoints(t *testing.T) {
	jf := fakeJellyfin(t)
	app := newApp(t)
	admin := loginAs(t, app, jf.URL, "alice", "secret")
	bob := loginAs(t, app, jf.URL, "bob", "hunter2")

	// anonymous is refused
	if resp, _ := call(t, app, "GET", "/api/v1/discover/trending", ""); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anon trending = %d", resp.StatusCode)
	}
	// TMDB key not configured yet
	if resp, _ := call(t, app, "GET", "/api/v1/discover/trending", "", bob); resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("no key trending = %d", resp.StatusCode)
	}

	// settings: admin only, key is write-only
	if resp, _ := call(t, app, "PUT", "/api/v1/admin/settings", `{"tmdbApiKey":"k123"}`, bob); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("user settings = %d", resp.StatusCode)
	}
	resp, body := call(t, app, "PUT", "/api/v1/admin/settings", `{"tmdbApiKey":"k123"}`, admin)
	if resp.StatusCode != 200 || !strings.Contains(body, `"tmdbConfigured":true`) || strings.Contains(body, "k123") || !strings.Contains(body, `"dryRun":true`) {
		t.Fatalf("settings = %d %s", resp.StatusCode, body)
	}

	// trending: people filtered out, items normalised
	_, body = call(t, app, "GET", "/api/v1/discover/trending", "", bob)
	var trending struct {
		TotalPages int          `json:"totalPages"`
		Items      []media.Item `json:"items"`
	}
	if err := json.Unmarshal([]byte(body), &trending); err != nil || len(trending.Items) != 2 || trending.TotalPages != 3 {
		t.Fatalf("trending = %s (%v)", body, err)
	}
	if it := trending.Items[1]; it.Type != "tv" || it.Title != "Show Two" || it.ReleaseDate != "2023-05-06" || it.Availability != "none" {
		t.Fatalf("tv item = %+v", it)
	}

	// second identical call is served from the DB cache
	before := tmdbHits.Load()
	call(t, app, "GET", "/api/v1/discover/trending", "", bob)
	if tmdbHits.Load() != before {
		t.Fatal("expected cache hit, TMDB was called again")
	}

	// movie detail
	_, body = call(t, app, "GET", "/api/v1/media/movie/1", "", bob)
	var d media.Detail
	if err := json.Unmarshal([]byte(body), &d); err != nil {
		t.Fatal(err)
	}
	if d.Title != "Movie One" || d.RuntimeMinutes != 120 || d.IMDbID != "tt1" || d.CollectionID != 9 ||
		len(d.Cast) != 1 || len(d.Directors) != 1 || d.Directors[0].Name != "Dir" || len(d.Recommendations) != 1 || d.Recommendations[0].Type != "movie" {
		t.Fatalf("detail = %+v", d)
	}
	// tv detail: seasons, runtime from episode_run_time
	_, body = call(t, app, "GET", "/api/v1/media/tv/2", "", bob)
	var tv media.Detail
	_ = json.Unmarshal([]byte(body), &tv)
	if tv.Type != "tv" || tv.Title != "Show Two" || tv.RuntimeMinutes != 45 || len(tv.Seasons) != 2 || tv.NumberOfEpisodes != 20 {
		t.Fatalf("tv detail = %s", body)
	}
	// unknown id -> 404, bad type -> 422
	if resp, _ := call(t, app, "GET", "/api/v1/media/movie/404", "", bob); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("404 = %d", resp.StatusCode)
	}
	if resp, _ := call(t, app, "GET", "/api/v1/media/book/1", "", bob); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("bad type = %d", resp.StatusCode)
	}

	// search splits items and people
	_, body = call(t, app, "GET", "/api/v1/search?q=movie", "", bob)
	var sr media.SearchResult
	_ = json.Unmarshal([]byte(body), &sr)
	if len(sr.Items) != 1 || len(sr.People) != 1 || sr.People[0].Name != "Zed" {
		t.Fatalf("search = %s", body)
	}
	if resp, _ := call(t, app, "GET", "/api/v1/search", "", bob); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("search without q = %d", resp.StatusCode)
	}
}

func TestImageProxy(t *testing.T) {
	jf := fakeJellyfin(t)
	app := newApp(t)
	bob := loginAs(t, app, jf.URL, "bob", "hunter2")

	if resp, _ := call(t, app, "GET", "/api/v1/images/tmdb/w185/AbCdEf123456.jpg", ""); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anon image = %d", resp.StatusCode)
	}
	resp, body := call(t, app, "GET", "/api/v1/images/tmdb/w185/AbCdEf123456.jpg", "", bob)
	if resp.StatusCode != 200 || body != "JPEGDATA" || !strings.Contains(resp.Header.Get("Cache-Control"), "immutable") {
		t.Fatalf("image = %d %q %v", resp.StatusCode, body, resp.Header)
	}
	// again: served from the disk cache (upstream could now be down; here just must still work)
	if resp, body = call(t, app, "GET", "/api/v1/images/tmdb/w185/AbCdEf123456.jpg", "", bob); resp.StatusCode != 200 || body != "JPEGDATA" {
		t.Fatalf("cached image = %d %q", resp.StatusCode, body)
	}
	for _, bad := range []string{"/api/v1/images/tmdb/w999/AbCdEf123456.jpg", "/api/v1/images/tmdb/w185/..%2f..%2fx.jpg", "/api/v1/images/tmdb/w185/nope.jpg"} {
		if resp, _ := call(t, app, "GET", bad, "", bob); resp.StatusCode != http.StatusNotFound {
			t.Fatalf("%s = %d", bad, resp.StatusCode)
		}
	}
}

// waitFor polls cond for up to 3s.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	for i := 0; i < 150; i++ {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestLibrarySyncAndAvailability(t *testing.T) {
	jf := fakeJellyfin(t)
	app, st, _ := newAppFull(t)
	admin := loginAs(t, app, jf.URL, "alice", "secret")
	bob := loginAs(t, app, jf.URL, "bob", "hunter2")
	call(t, app, "PUT", "/api/v1/admin/settings", `{"tmdbApiKey":"k123"}`, admin)

	// before any sync nothing is available
	_, body := call(t, app, "GET", "/api/v1/discover/trending", "", bob)
	if strings.Contains(body, `"availability":"available"`) {
		t.Fatalf("unexpected availability before sync: %s", body)
	}

	// sync needs the API key and an admin
	if resp, _ := call(t, app, "POST", "/api/v1/admin/sync/library-sync", "", admin); resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("sync without key = %d", resp.StatusCode)
	}
	if resp, _ := call(t, app, "POST", "/api/v1/admin/sync/library-sync", "", bob); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("user sync = %d", resp.StatusCode)
	}
	resp, body := call(t, app, "PUT", "/api/v1/admin/settings", `{"jellyfinApiKey":"jfkey"}`, admin)
	if resp.StatusCode != 200 || !strings.Contains(body, `"jellyfinApiKeyConfigured":true`) || strings.Contains(body, "jfkey") ||
		!strings.Contains(body, "/api/v1/hooks/jellyfin?token=") {
		t.Fatalf("settings = %d %s", resp.StatusCode, body)
	}

	if resp, _ := call(t, app, "POST", "/api/v1/admin/sync/library-sync", "", admin); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("sync = %d", resp.StatusCode)
	}
	waitFor(t, "library sync", func() bool {
		_, b := call(t, app, "GET", "/api/v1/admin/sync", "", admin)
		return strings.Contains(b, `"name":"library-sync"`) && strings.Contains(b, `"status":"ok"`)
	})
	_, body = call(t, app, "GET", "/api/v1/admin/sync", "", admin)
	if !strings.Contains(body, `"movies":1`) || !strings.Contains(body, `"shows":1`) || !strings.Contains(body, "1 without a TMDB id") || !strings.Contains(body, "10 episodes") {
		t.Fatalf("sync status = %s", body)
	}
	_ = st

	// lists: movie One (tmdb 1) and show Two (tmdb 2) are available
	_, body = call(t, app, "GET", "/api/v1/discover/trending", "", bob)
	var l media.List
	_ = json.Unmarshal([]byte(body), &l)
	if len(l.Items) != 2 || l.Items[0].Availability != "available" || l.Items[1].Availability != "available" {
		t.Fatalf("trending availability = %s", body)
	}
	// detail: movie available, show partial (season 1 of 2 present)
	_, body = call(t, app, "GET", "/api/v1/media/movie/1", "", bob)
	var md media.Detail
	_ = json.Unmarshal([]byte(body), &md)
	if md.Availability != "available" || md.Recommendations[0].Availability != "none" {
		t.Fatalf("movie detail availability = %s", md.Availability)
	}
	if md.Budget != 63000000 || md.Revenue != 463517383 || len(md.Studios) != 2 || md.Studios[0] != "Warner Bros." {
		t.Fatalf("budget/revenue/studios = %d %d %v", md.Budget, md.Revenue, md.Studios)
	}
	if md.TrailerKey != "OFFICIAL1" || md.CollectionBackdropPath != "/sagab.jpg" {
		t.Fatalf("trailer/collection backdrop = %q %q", md.TrailerKey, md.CollectionBackdropPath)
	}
	if want := jf.URL + "/web/#/details?id=jm1"; md.WatchURL != want {
		t.Fatalf("watch url = %q, want %q", md.WatchURL, want)
	}
	// a public URL (what browsers can reach) replaces the internal one in links
	call(t, app, "PUT", "/api/v1/admin/settings", `{"jellyfinPublicUrl":"https://jf.example.org/"}`, admin)
	_, body = call(t, app, "GET", "/api/v1/media/movie/1", "", bob)
	if !strings.Contains(body, `"watchUrl":"https://jf.example.org/web/#/details?id=jm1"`) {
		t.Fatalf("public watch url: %s", body)
	}
	if resp, _ := call(t, app, "PUT", "/api/v1/admin/settings", `{"jellyfinPublicUrl":"ftp://x"}`, admin); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("bad public url = %d", resp.StatusCode)
	}
	_, body = call(t, app, "GET", "/api/v1/media/tv/2", "", bob)
	var td media.Detail
	_ = json.Unmarshal([]byte(body), &td)
	if td.Availability != "partial" {
		t.Fatalf("tv detail availability = %q", td.Availability)
	}
}

func TestHistorySyncAndWebhook(t *testing.T) {
	jf := fakeJellyfin(t)
	app, st, _ := newAppFull(t)
	admin := loginAs(t, app, jf.URL, "alice", "secret")
	_, body := call(t, app, "PUT", "/api/v1/admin/settings", `{"jellyfinApiKey":"jfkey"}`, admin)
	var settings struct {
		WebhookPath string `json:"webhookPath"`
	}
	_ = json.Unmarshal([]byte(body), &settings)
	const alice = "aaaaaaaabbbbccccddddeeeeeeeeeeee"
	ctx := context.Background()

	// history sync (also pulls the library first since it is empty)
	call(t, app, "POST", "/api/v1/admin/sync/history-sync", "", admin)
	waitFor(t, "history sync", func() bool {
		v, _ := st.HistoryVersion(ctx, alice)
		return v == 1
	})
	rows, _ := st.UserHistory(ctx, alice, 0)
	if len(rows) != 2 {
		t.Fatalf("history rows = %+v", rows)
	}
	byKey := map[string]store.WatchHistory{}
	for _, r := range rows {
		byKey[r.MediaType] = r
	}
	if m := byKey["movie"]; m.TMDBID != 1 || m.PlayCount != 2 {
		t.Fatalf("movie history = %+v", m)
	}
	if tv := byKey["tv"]; tv.TMDBID != 2 || tv.PlayCount != 2 || tv.LastPlayedAt != time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC).Unix() {
		t.Fatalf("tv history = %+v", tv) // 2 episodes aggregated to the show, unknown series skipped
	}

	// webhook: bad token rejected
	if resp, _ := call(t, app, "POST", "/api/v1/hooks/jellyfin?token=nope", `{"NotificationType":"PlaybackStop"}`); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bad token = %d", resp.StatusCode)
	}
	// unchanged history: re-sync must NOT bump the version
	call(t, app, "POST", settings.WebhookPath, `{"NotificationType":"PlaybackStop","UserId":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`)
	time.Sleep(300 * time.Millisecond)
	if v, _ := st.HistoryVersion(ctx, alice); v != 1 {
		t.Fatalf("version bumped without change: %d", v)
	}
	// a library event refreshes the library (debounced)
	if resp, _ := call(t, app, "POST", settings.WebhookPath, `{"NotificationType":"ItemAdded"}`); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("webhook = %d", resp.StatusCode)
	}
}
