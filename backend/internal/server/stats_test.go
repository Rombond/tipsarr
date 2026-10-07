package server_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

type statsResp struct {
	Source string `json:"source"`
	Totals struct {
		Titles int     `json:"titles"`
		Movies int     `json:"movies"`
		Shows  int     `json:"shows"`
		Plays  int     `json:"plays"`
		Hours  float64 `json:"hours"`
	} `json:"totals"`
	Genres []struct {
		Name  string  `json:"name"`
		Hours float64 `json:"hours"`
	} `json:"genres"`
	Months []struct {
		Plays int `json:"plays"`
	} `json:"months"`
	Top []struct {
		Type      string `json:"type"`
		Title     string `json:"title"`
		Plays     int    `json:"plays"`
		PosterURL string `json:"posterUrl"`
	} `json:"top"`
	TopMovies []struct {
		Title string `json:"title"`
	} `json:"topMovies"`
	TopShows []struct {
		Title string `json:"title"`
	} `json:"topShows"`
	Plugin struct {
		Hint bool `json:"hint"`
	} `json:"plugin"`
}

func runJobAndWait(t *testing.T, _ any, call func(method, path string) (*http.Response, string), name, status string) {
	t.Helper()
	if resp, _ := call("POST", "/api/v1/admin/sync/"+name); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("%s = %d", name, resp.StatusCode)
	}
	waitFor(t, name, func() bool {
		_, b := call("GET", "/api/v1/admin/sync")
		i := strings.Index(b, `"name":"`+name+`"`)
		return i >= 0 && strings.Contains(b[i:min(len(b), i+300)], `"status":"`+status+`"`)
	})
}

func TestStatsFromPlaybackReporting(t *testing.T) {
	jfPlayback.Store(true)
	t.Cleanup(func() { jfPlayback.Store(false) })
	jf := fakeJellyfin(t)
	app := newApp(t)
	admin := loginAs(t, app, jf.URL, "alice", "secret")
	bob := loginAs(t, app, jf.URL, "bob", "hunter2")
	call(t, app, "PUT", "/api/v1/admin/settings", `{"jellyfinApiKey":"jfkey"}`, admin)
	adminCall := func(m, p string) (*http.Response, string) { return call(t, app, m, p, "", admin) }
	runJobAndWait(t, app, adminCall, "library-sync", "ok")
	runJobAndWait(t, app, adminCall, "playback-sync", "ok")

	get := func(q string, who *http.Cookie) statsResp {
		t.Helper()
		resp, body := call(t, app, "GET", "/api/v1/stats"+q, "", who)
		var out statsResp
		if resp.StatusCode != 200 || json.Unmarshal([]byte(body), &out) != nil {
			t.Fatalf("stats%s = %d %s", q, resp.StatusCode, body)
		}
		return out
	}
	mine := get("", admin)
	// alice: Movie One 2 h, Show Two 45 min (the 30 s session is a skip)
	if mine.Source != "plugin" || mine.Totals.Titles != 2 || mine.Totals.Plays != 2 || mine.Totals.Movies != 1 || mine.Totals.Shows != 1 || mine.Totals.Hours < 2.74 || mine.Totals.Hours > 2.76 {
		t.Fatalf("alice = %+v", mine)
	}
	if len(mine.Genres) < 2 || mine.Genres[0].Name != "Drama" || mine.Genres[1].Name != "Action" {
		t.Fatalf("genres = %+v", mine.Genres)
	}
	n := 0
	for _, m := range mine.Months {
		n += m.Plays
	}
	if len(mine.Months) != 12 || n != 2 {
		t.Fatalf("months = %+v", mine.Months)
	}
	if len(mine.Top) != 2 || mine.Top[0].Title != "Movie One" || !strings.Contains(mine.Top[0].PosterURL, "aaaa0000000000000000000000000001") {
		t.Fatalf("top = %+v", mine.Top)
	}
	if len(mine.TopMovies) != 1 || mine.TopMovies[0].Title != "Movie One" || len(mine.TopShows) != 1 || mine.TopShows[0].Title != "Show Two" {
		t.Fatalf("top per type = %+v / %+v", mine.TopMovies, mine.TopShows)
	}
	if mine.Plugin.Hint {
		t.Fatal("no hint when the plugin is installed")
	}
	// everyone: bob's hour counts too; the same title is not counted twice
	if all := get("?user=all", admin); all.Totals.Titles != 2 || all.Totals.Plays != 3 {
		t.Fatalf("all = %+v", all.Totals)
	}
	// bob sees only himself, and cannot look at others
	if b := get("", bob); b.Totals.Plays != 1 || b.Totals.Titles != 1 {
		t.Fatalf("bob = %+v", b.Totals)
	}
	for _, q := range []string{"?user=all", "?user=aaaaaaaabbbbccccddddeeeeeeeeeeee"} {
		if resp, _ := call(t, app, "GET", "/api/v1/stats"+q, "", bob); resp.StatusCode != http.StatusForbidden {
			t.Fatalf("bob %s = %d", q, resp.StatusCode)
		}
	}
	// a second sync copies nothing new
	runJobAndWait(t, app, adminCall, "playback-sync", "ok")
	if again := get("?user=all", admin); again.Totals.Plays != 3 {
		t.Fatalf("after resync = %+v", again.Totals)
	}
}

// Without the plugin the numbers are estimated, and only admins are told how to get exact ones.
func TestStatsEstimateAndPluginHint(t *testing.T) {
	jfPlayback.Store(false)
	jf := fakeJellyfin(t)
	app := newApp(t)
	admin := loginAs(t, app, jf.URL, "alice", "secret")
	bob := loginAs(t, app, jf.URL, "bob", "hunter2")
	call(t, app, "PUT", "/api/v1/admin/settings", `{"jellyfinApiKey":"jfkey"}`, admin)
	adminCall := func(m, p string) (*http.Response, string) { return call(t, app, m, p, "", admin) }
	runJobAndWait(t, app, adminCall, "library-sync", "ok")
	runJobAndWait(t, app, adminCall, "history-sync", "ok")
	runJobAndWait(t, app, adminCall, "playback-sync", "skipped")

	var a, b statsResp
	_, body := call(t, app, "GET", "/api/v1/stats", "", admin)
	_ = json.Unmarshal([]byte(body), &a)
	_, body = call(t, app, "GET", "/api/v1/stats", "", bob)
	_ = json.Unmarshal([]byte(body), &b)
	if a.Source != "estimate" || a.Totals.Titles != 2 || a.Totals.Hours <= 0 || len(a.Months) != 0 {
		t.Fatalf("alice estimate = %+v", a)
	}
	if !a.Plugin.Hint {
		t.Fatal("the admin should be told about the plugin")
	}
	if b.Plugin.Hint {
		t.Fatal("a regular user must never get the plugin hint")
	}
}

// A movie that was played and later deleted from Jellyfin is recognised on TMDB, so its plays
// still count and it shows up with its poster in the most watched list.
func TestStatsKeepsPlaysOfRemovedTitles(t *testing.T) {
	jfPlayback.Store(true)
	jfRemovedPlay.Store(true)
	t.Cleanup(func() { jfPlayback.Store(false); jfRemovedPlay.Store(false) })
	jf := fakeJellyfin(t)
	app := newApp(t)
	admin := loginAs(t, app, jf.URL, "alice", "secret")
	call(t, app, "PUT", "/api/v1/admin/settings", `{"jellyfinApiKey":"jfkey","tmdbApiKey":"k123"}`, admin)
	adminCall := func(m, p string) (*http.Response, string) { return call(t, app, m, p, "", admin) }
	runJobAndWait(t, app, adminCall, "library-sync", "ok")
	runJobAndWait(t, app, adminCall, "playback-sync", "ok")

	_, body := call(t, app, "GET", "/api/v1/stats", "", admin)
	var r statsResp
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		t.Fatal(err)
	}
	if r.Totals.Titles != 3 || r.Totals.Hours < 4.2 { // Movie One 2 h + Show Two 0.75 h + Gone Movie 1.5 h
		t.Fatalf("totals = %+v", r.Totals)
	}
	if len(r.Top) == 0 || r.Top[0].Title != "Movie One" || len(r.TopMovies) != 2 || r.TopMovies[1].Title != "Gone Movie" {
		t.Fatalf("top = %+v / movies %+v", r.Top, r.TopMovies)
	}
	var gone string
	for _, it := range r.Top {
		if it.Title == "Gone Movie" {
			gone = it.PosterURL
		}
	}
	if gone != "/api/v1/images/tmdb/w342/gone.jpg" {
		t.Fatalf("poster = %q", gone)
	}
	hasThriller := false
	for _, g := range r.Genres {
		hasThriller = hasThriller || g.Name == "Thriller"
	}
	if !hasThriller {
		t.Fatalf("genres of the removed title are missing: %+v", r.Genres)
	}
}
