package server_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

type libResp struct {
	Total int `json:"total"`
	Items []struct {
		Type      string   `json:"type"`
		TMDBID    int      `json:"tmdbId"`
		Title     string   `json:"title"`
		Year      int      `json:"year"`
		Genres    []string `json:"genres"`
		Rating    float64  `json:"rating"`
		PosterURL string   `json:"posterUrl"`
		Plays     int      `json:"plays"`
		Watched   bool     `json:"watched"`
	} `json:"items"`
}

func TestLibraryBrowse(t *testing.T) {
	jf := fakeJellyfin(t)
	app := newApp(t)
	admin := loginAs(t, app, jf.URL, "alice", "secret")
	bob := loginAs(t, app, jf.URL, "bob", "hunter2")
	call(t, app, "PUT", "/api/v1/admin/settings", `{"jellyfinApiKey":"jfkey"}`, admin)
	for _, job := range []string{"library-sync", "history-sync"} {
		if resp, _ := call(t, app, "POST", "/api/v1/admin/sync/"+job, "", admin); resp.StatusCode != http.StatusAccepted {
			t.Fatalf("%s = %d", job, resp.StatusCode)
		}
		waitFor(t, job, func() bool {
			_, b := call(t, app, "GET", "/api/v1/admin/sync", "", admin)
			i := strings.Index(b, `"name":"`+job+`"`)
			return i >= 0 && strings.Contains(b[i:min(len(b), i+300)], `"status":"ok"`)
		})
	}
	list := func(q string, who *http.Cookie) libResp {
		t.Helper()
		resp, body := call(t, app, "GET", "/api/v1/library"+q, "", who)
		var out libResp
		if resp.StatusCode != 200 || json.Unmarshal([]byte(body), &out) != nil {
			t.Fatalf("library%s = %d %s", q, resp.StatusCode, body)
		}
		return out
	}
	titles := func(r libResp) string {
		var n []string
		for _, i := range r.Items {
			n = append(n, i.Title)
		}
		return strings.Join(n, ",")
	}

	if resp, _ := call(t, app, "GET", "/api/v1/library", ""); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous = %d", resp.StatusCode)
	}
	all := list("", admin)
	if all.Total != 2 || titles(all) != "Show Two,Movie One" { // newest added first
		t.Fatalf("default = %d %s", all.Total, titles(all))
	}
	m := all.Items[1]
	if m.Year != 2024 || m.Rating != 7.5 || len(m.Genres) != 2 || m.PosterURL != "/api/v1/images/jellyfin/aaaa0000000000000000000000000001?tag=tg1" || m.Plays != 2 || !m.Watched {
		t.Fatalf("movie row = %+v", m)
	}
	if got := titles(list("?type=movie", admin)); got != "Movie One" {
		t.Fatalf("type = %s", got)
	}
	if got := titles(list("?genre=Drama", admin)); got != "Show Two,Movie One" {
		t.Fatalf("one genre = %s", got)
	}
	if got := titles(list("?genre=Drama&genre=Action", admin)); got != "Movie One" {
		t.Fatalf("two genres must both match = %s", got)
	}
	if got := titles(list("?q=show", admin)); got != "Show Two" {
		t.Fatalf("search = %s", got)
	}
	if got := titles(list("?q=100%25", admin)); got != "" {
		t.Fatalf("a percent sign is literal, got %s", got)
	}
	if got := titles(list("?minRating=8", admin)); got != "Show Two" {
		t.Fatalf("minRating = %s", got)
	}
	if got := titles(list("?yearTo=2023", admin)); got != "Show Two" {
		t.Fatalf("yearTo = %s", got)
	}
	if got := titles(list("?maxRuntime=60", admin)); got != "Show Two" {
		t.Fatalf("maxRuntime = %s", got)
	}
	if got := titles(list("?sort=title&dir=asc", admin)); got != "Movie One,Show Two" {
		t.Fatalf("sort title = %s", got)
	}
	if got := titles(list("?sort=popular", admin)); got != "Movie One,Show Two" { // 2 plays vs 2 episodes: tie broken by title... movie first
		t.Fatalf("sort popular = %s", got)
	}
	// "watched" is per user: alice watched both, bob nothing
	if got := titles(list("?watched=no", admin)); got != "" {
		t.Fatalf("alice unwatched = %s", got)
	}
	if got := titles(list("?watched=no", bob)); got != "Show Two,Movie One" {
		t.Fatalf("bob unwatched = %s", got)
	}
	if got := list("?pageSize=1&page=2", admin); len(got.Items) != 1 || got.Total != 2 {
		t.Fatalf("paging = %+v", got)
	}

	_, body := call(t, app, "GET", "/api/v1/library/facets", "", bob)
	if !strings.Contains(body, `"name":"Drama","count":2`) || !strings.Contains(body, `"yearMin":2023`) || !strings.Contains(body, `"movies":1`) {
		t.Fatalf("facets = %s", body)
	}

	// poster proxy: needs a login, validates the id, caches
	resp, body := call(t, app, "GET", "/api/v1/images/jellyfin/aaaa0000000000000000000000000001?tag=tg1", "", bob)
	if resp.StatusCode != 200 || body != "JFPOSTER" || !strings.Contains(resp.Header.Get("Cache-Control"), "immutable") {
		t.Fatalf("poster = %d %q", resp.StatusCode, body)
	}
	if resp, _ := call(t, app, "GET", "/api/v1/images/jellyfin/aaaa0000000000000000000000000001?tag=tg1", ""); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous poster = %d", resp.StatusCode)
	}
	if resp, _ := call(t, app, "GET", "/api/v1/images/jellyfin/..%2F..%2Fetc?tag=x", "", bob); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("bad id = %d", resp.StatusCode)
	}
}
