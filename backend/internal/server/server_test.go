package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Rombond/tipsarr/backend/internal/api"
	"github.com/Rombond/tipsarr/backend/internal/auth"
	"github.com/Rombond/tipsarr/backend/internal/media"
	"github.com/Rombond/tipsarr/backend/internal/server"
	"github.com/Rombond/tipsarr/backend/internal/store"
)

// fakeJellyfin accepts alice/secret (admin) and bob/hunter2 (regular).
func fakeJellyfin(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/System/Info/Public", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ServerName":"Test","Version":"10.10.0","Id":"abc"}`))
	})
	mux.HandleFunc("/Users/AuthenticateByName", func(w http.ResponseWriter, r *http.Request) {
		var in struct{ Username, Pw string }
		_ = json.NewDecoder(r.Body).Decode(&in)
		switch {
		case in.Username == "alice" && in.Pw == "secret":
			_, _ = w.Write([]byte(`{"AccessToken":"t","User":{"Id":"u-alice","Name":"alice","Policy":{"IsAdministrator":true}}}`))
		case in.Username == "bob" && in.Pw == "hunter2":
			_, _ = w.Write([]byte(`{"AccessToken":"t","User":{"Id":"u-bob","Name":"bob","Policy":{"IsAdministrator":false}}}`))
		default:
			w.WriteHeader(http.StatusUnauthorized)
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

var tmdbHits atomic.Int32

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
			{"id":2,"media_type":"tv","name":"Show Two","first_air_date":"2023-05-06"},
			{"id":3,"media_type":"person","name":"Someone"}]}`))
	}))
	mux.HandleFunc("/movie/1", hit(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":1,"title":"Movie One","runtime":120,"genres":[{"id":28,"name":"Action"}],
			"imdb_id":"tt1","belongs_to_collection":{"id":9,"name":"Saga"},
			"credits":{"cast":[{"id":5,"name":"Ann","character":"Hero"}],"crew":[{"id":6,"name":"Dir","job":"Director"},{"id":7,"name":"X","job":"Editor"}]},
			"recommendations":{"results":[{"id":4,"title":"Rec"}]},"similar":{"results":[]}}`))
	}))
	mux.HandleFunc("/tv/2", hit(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":2,"name":"Show Two","episode_run_time":[45],"number_of_seasons":2,"number_of_episodes":20,
			"seasons":[{"season_number":1,"name":"Season 1","episode_count":10},{"season_number":2,"name":"Season 2","episode_count":10}],
			"external_ids":{"imdb_id":"tt2"},"recommendations":{"results":[]},"similar":{"results":[]}}`))
	}))
	mux.HandleFunc("/search/multi", hit(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"page":1,"total_pages":1,"results":[
			{"id":1,"media_type":"movie","title":"Movie One"},{"id":8,"media_type":"person","name":"Zed","profile_path":"/z.jpg"}]}`))
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
			_, _ = w.Write([]byte("JPEGDATA"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newApp(t *testing.T) *httptest.Server {
	t.Helper()
	tm, im := fakeTMDB(t), fakeImages(t)
	st, err := store.Open(context.Background(), "sqlite:"+filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	h, _ := server.New(api.Deps{
		Store: st, Auth: auth.New(st), Media: media.New(st, tm.URL),
		DryRun: true, ConfigDir: t.TempDir(), ImageBaseURL: im.URL,
	})
	app := httptest.NewServer(h)
	t.Cleanup(app.Close)
	return app
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
