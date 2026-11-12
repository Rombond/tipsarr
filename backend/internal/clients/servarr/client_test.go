package servarr

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// fakeServarr records every request and answers just enough for add flows.
type fakeServarr struct {
	mu   sync.Mutex
	seen []string // "METHOD path"
	last map[string]any
}

func (f *fakeServarr) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.seen = append(f.seen, r.Method+" "+r.URL.Path)
		f.mu.Unlock()
		if r.Header.Get("X-Api-Key") != "key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v3/movie":
			var b map[string]any
			_ = json.NewDecoder(r.Body).Decode(&b)
			f.mu.Lock()
			f.last = b
			f.mu.Unlock()
			_, _ = w.Write([]byte(`{"id":42,"title":"M","tmdbId":603}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v3/series":
			var b map[string]any
			_ = json.NewDecoder(r.Body).Decode(&b)
			f.mu.Lock()
			f.last = b
			f.mu.Unlock()
			_, _ = w.Write([]byte(`{"id":77,"title":"S","tvdbId":81189}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v3/command":
			var b map[string]any
			_ = json.NewDecoder(r.Body).Decode(&b)
			f.mu.Lock()
			f.last = b
			f.mu.Unlock()
			_, _ = w.Write([]byte(`{"id":1}`))
		case r.URL.Path == "/api/v3/movie":
			_, _ = w.Write([]byte(`[]`))
		case r.URL.Path == "/api/v3/series":
			_, _ = w.Write([]byte(`[]`))
		case r.URL.Path == "/api/v3/movie/lookup/tmdb":
			_, _ = w.Write([]byte(`{"title":"M","tmdbId":603,"year":1999}`))
		case r.URL.Path == "/api/v3/series/lookup":
			_, _ = w.Write([]byte(`[{"title":"S","tvdbId":81189,"seasons":[{"seasonNumber":0,"monitored":false},{"seasonNumber":1,"monitored":false},{"seasonNumber":2,"monitored":false}]}]`))
		case r.URL.Path == "/api/v3/languageprofile":
			w.WriteHeader(http.StatusNotFound)
		case r.URL.Path == "/api/v3/queue":
			_, _ = w.Write([]byte(`{"records":[{"movieId":42,"title":"M","size":1000,"sizeleft":250,"timeleft":"00:10:30","status":"downloading"},{"seriesId":77,"size":0}],"totalRecords":2}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	})
}

func (f *fakeServarr) writes() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var w []string
	for _, s := range f.seen {
		if len(s) >= 3 && s[:3] != "GET" {
			w = append(w, s)
		}
	}
	return w
}

func TestDryRunNeverWrites(t *testing.T) {
	f := &fakeServarr{}
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	ctx := context.Background()

	radarr := New(KindRadarr, srv.URL, "key", true)
	if _, err := radarr.AddMovie(ctx, 603, 1, "/movies"); !errors.Is(err, ErrDryRun) {
		t.Fatalf("AddMovie dry-run err = %v", err)
	}
	sonarr := New(KindSonarr, srv.URL, "key", true)
	if _, err := sonarr.AddSeries(ctx, 81189, 1, "/tv", []int{1}, ""); !errors.Is(err, ErrDryRun) {
		t.Fatalf("AddSeries dry-run err = %v", err)
	}
	// every other write verb is blocked at the choke point too
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		if err := radarr.send(ctx, m, "/movie/1", map[string]any{}, nil); !errors.Is(err, ErrDryRun) {
			t.Fatalf("%s not blocked: %v", m, err)
		}
	}
	if w := f.writes(); len(w) != 0 {
		t.Fatalf("dry-run leaked writes to the server: %v", w)
	}
	// reads still work in dry-run
	if _, err := radarr.Queue(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestAddMovieAndSeriesWhenLive(t *testing.T) {
	f := &fakeServarr{}
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	ctx := context.Background()

	id, err := New(KindRadarr, srv.URL, "key", false).AddMovie(ctx, 603, 4, "/movies")
	if err != nil || id != 42 {
		t.Fatalf("AddMovie = %d %v", id, err)
	}
	if f.last["qualityProfileId"] != float64(4) || f.last["rootFolderPath"] != "/movies" || f.last["monitored"] != true {
		t.Fatalf("movie body = %v", f.last)
	}
	if opts, _ := f.last["addOptions"].(map[string]any); opts["searchForMovie"] != true {
		t.Fatalf("addOptions = %v", f.last["addOptions"])
	}

	id, err = New(KindSonarr, srv.URL, "key", false).AddSeries(ctx, 81189, 5, "/tv", []int{2}, "anime")
	if err != nil || id != 77 {
		t.Fatalf("AddSeries = %d %v", id, err)
	}
	monitored := map[int]bool{}
	for _, s := range f.last["seasons"].([]any) {
		m := s.(map[string]any)
		monitored[int(m["seasonNumber"].(float64))] = m["monitored"].(bool)
	}
	if monitored[0] || monitored[1] || !monitored[2] {
		t.Fatalf("only season 2 should be monitored: %v", monitored)
	}
	if f.last["seriesType"] != "anime" {
		t.Fatalf("seriesType = %v", f.last["seriesType"])
	}
}

func TestQueueParsing(t *testing.T) {
	f := &fakeServarr{}
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	q, err := New(KindRadarr, srv.URL, "key", true).Queue(context.Background())
	if err != nil || len(q) != 1 { // the seriesId-only record is ignored for Radarr
		t.Fatalf("queue = %+v %v", q, err)
	}
	if q[0].MediaID != 42 || q[0].Percent() != 75 || q[0].ETASeconds() != 630 {
		t.Fatalf("item = %+v pct=%d eta=%d", q[0], q[0].Percent(), q[0].ETASeconds())
	}
	if (QueueItem{TimeLeft: "1.02:00:00"}).ETASeconds() != 86400+7200 {
		t.Fatal("day-prefixed ETA")
	}
}

func TestBadKeyIsError(t *testing.T) {
	f := &fakeServarr{}
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	if _, err := New(KindRadarr, srv.URL, "wrong", true).Status(context.Background()); err == nil {
		t.Fatal("expected 401 error")
	}
}

func TestRefreshDownloads(t *testing.T) {
	f := &fakeServarr{}
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	ctx := context.Background()

	if err := New(KindRadarr, srv.URL, "key", true).RefreshDownloads(ctx); !errors.Is(err, ErrDryRun) {
		t.Fatalf("dry-run err = %v", err)
	}
	if w := f.writes(); len(w) != 0 {
		t.Fatalf("dry-run leaked writes: %v", w)
	}
	if err := New(KindSonarr, srv.URL, "key", false).RefreshDownloads(ctx); err != nil {
		t.Fatal(err)
	}
	if f.last["name"] != "RefreshMonitoredDownloads" {
		t.Fatalf("command body = %v", f.last)
	}
}

// Shapes taken from the owner's Radarr (2026-11-06): an HD profile, an Ultra-HD one, and a mixed one whose
// allowed group holds a 2160p quality.
func TestQualityProfilesKnow4K(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[
 {"id":4,"name":"HD-1080p","items":[
   {"quality":{"id":19,"name":"Bluray-2160p","resolution":2160},"items":[],"allowed":false},
   {"quality":{"id":7,"name":"Bluray-1080p","resolution":1080},"items":[],"allowed":true},
   {"name":"WEB 1080p","items":[{"quality":{"id":3,"name":"WEBDL-1080p","resolution":1080},"items":[],"allowed":true}],"allowed":true,"id":1001}]},
 {"id":5,"name":"Ultra-HD","items":[
   {"quality":{"id":7,"name":"Bluray-1080p","resolution":1080},"items":[],"allowed":false},
   {"name":"WEB 2160p","items":[{"quality":{"id":18,"name":"WEBDL-2160p","resolution":2160},"items":[],"allowed":false}],"allowed":true,"id":1002}]},
 {"id":8,"name":"Any","items":[
   {"name":"WEB 2160p","items":[{"quality":{"id":18,"name":"WEBDL-2160p","resolution":2160},"items":[],"allowed":true}],"allowed":true,"id":1002},
   {"quality":{"id":7,"name":"Bluray-1080p","resolution":1080},"items":[],"allowed":true}]},
 {"id":9,"name":"Bare"}]`))
	}))
	defer srv.Close()
	got, err := New(KindRadarr, srv.URL, "key", true).QualityProfiles(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := map[int]struct {
		res int
		uhd bool
	}{4: {1080, false}, 5: {2160, true}, 8: {2160, true}, 9: {0, false}}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for _, p := range got {
		if w := want[p.ID]; p.MaxResolution != w.res || p.UHD != w.uhd {
			t.Errorf("%s: res %d uhd %v, want %+v", p.Name, p.MaxResolution, p.UHD, w)
		}
	}
	if !IsUHD(got, 5) || IsUHD(got, 4) || IsUHD(got, 99) {
		t.Error("IsUHD")
	}
}

// trackedSonarr serves one tracked series (id 86, tvdb 74796) with S1 monitored and S2, S17 not.
func trackedSonarr(t *testing.T, puts *[]map[string]any, cmds *[]map[string]any) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/api/v3/series/86":
			var b map[string]any
			_ = json.NewDecoder(r.Body).Decode(&b)
			*puts = append(*puts, b)
			_, _ = w.Write([]byte(`{}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v3/command":
			var b map[string]any
			_ = json.NewDecoder(r.Body).Decode(&b)
			*cmds = append(*cmds, b)
			_, _ = w.Write([]byte(`{"id":1}`))
		case r.Method != http.MethodGet:
			t.Errorf("unexpected write %s %s", r.Method, r.URL.Path)
		case r.URL.Path == "/api/v3/series" || r.URL.Path == "/api/v3/series/86":
			s := `{"id":86,"title":"Bleach","tvdbId":74796,"monitored":false,"extra":"kept","seasons":[{"seasonNumber":1,"monitored":true},{"seasonNumber":2,"monitored":false},{"seasonNumber":17,"monitored":false}]}`
			if r.URL.Path == "/api/v3/series" {
				s = "[" + s + "]"
			}
			_, _ = w.Write([]byte(s))
		case r.URL.Path == "/api/v3/series/lookup":
			_, _ = w.Write([]byte(`[{"id":null,"title":"Other","tvdbId":1},{"id":null,"title":"Bleach","tvdbId":74796,"statistics":{"seasonCount":17,"totalEpisodeCount":0},"seasons":[{"seasonNumber":0,"monitored":false},{"seasonNumber":17,"monitored":true}]}]`))
		case r.URL.Path == "/api/v3/episode":
			_, _ = w.Write([]byte(`[{"id":5,"seasonNumber":17,"episodeNumber":1,"title":"The Blood Warfare","overview":"o","airDate":"2022-10-11","runtime":24,"absoluteEpisodeNumber":367,"hasFile":true,"monitored":true,"images":null}]`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
}

func TestLookupSeriesUntracked(t *testing.T) {
	var puts, cmds []map[string]any
	srv := trackedSonarr(t, &puts, &cmds)
	defer srv.Close()
	c := New(KindSonarr, srv.URL, "key", false)
	s, err := c.LookupSeries(context.Background(), 74796)
	if err != nil || s == nil {
		t.Fatalf("lookup = %+v %v", s, err)
	}
	if s.ID != 0 || s.Title != "Bleach" || len(s.Seasons) != 2 || s.Seasons[1].SeasonNumber != 17 || !s.Seasons[1].Monitored {
		t.Fatalf("lookup = %+v", s)
	}
	if s, err := c.LookupSeries(context.Background(), 999); s != nil || err != nil {
		t.Fatalf("unknown = %+v %v", s, err)
	}
}

func TestSeriesEpisodesFields(t *testing.T) {
	var puts, cmds []map[string]any
	srv := trackedSonarr(t, &puts, &cmds)
	defer srv.Close()
	eps, err := New(KindSonarr, srv.URL, "key", false).SeriesEpisodes(context.Background(), 86)
	if err != nil || len(eps) != 1 {
		t.Fatalf("episodes = %+v %v", eps, err)
	}
	e := eps[0]
	if e.Title != "The Blood Warfare" || e.Overview != "o" || e.AirDate != "2022-10-11" || e.Runtime != 24 || e.AbsoluteEpisodeNumber != 367 || !e.HasFile {
		t.Fatalf("episode = %+v", e)
	}
}

func TestSeasonTotalEpisodeCount(t *testing.T) {
	var s Series
	if err := json.Unmarshal([]byte(`{"seasons":[{"seasonNumber":17,"statistics":{"episodeCount":0,"totalEpisodeCount":50}}]}`), &s); err != nil {
		t.Fatal(err)
	}
	if s.Seasons[0].Statistics.TotalEpisodeCount != 50 {
		t.Fatalf("total = %+v", s.Seasons[0].Statistics)
	}
}

// Asking for a season of a show Sonarr already tracks monitors and searches it, and never unmonitors anything.
func TestAddSeriesExistingMonitorsSeasons(t *testing.T) {
	var puts, cmds []map[string]any
	srv := trackedSonarr(t, &puts, &cmds)
	defer srv.Close()
	id, err := New(KindSonarr, srv.URL, "key", false).AddSeries(context.Background(), 74796, 1, "/tv", []int{17}, "")
	if err != nil || id != 86 {
		t.Fatalf("add = %d %v", id, err)
	}
	if len(puts) != 1 {
		t.Fatalf("puts = %v", puts)
	}
	b := puts[0]
	if b["monitored"] != true || b["extra"] != "kept" {
		t.Fatalf("series body = %v", b)
	}
	got := map[float64]bool{}
	for _, s := range b["seasons"].([]any) {
		m := s.(map[string]any)
		got[m["seasonNumber"].(float64)] = m["monitored"].(bool)
	}
	if !got[1] || got[2] || !got[17] {
		t.Fatalf("seasons = %v (S1 stays, S2 untouched, S17 on)", got)
	}
	if len(cmds) != 1 || cmds[0]["name"] != "SeasonSearch" || cmds[0]["seriesId"] != float64(86) || cmds[0]["seasonNumber"] != float64(17) {
		t.Fatalf("commands = %v", cmds)
	}
}

func TestMonitorSeasonsDryRun(t *testing.T) {
	var puts, cmds []map[string]any
	srv := trackedSonarr(t, &puts, &cmds)
	defer srv.Close()
	c := New(KindSonarr, srv.URL, "key", true)
	if err := c.MonitorSeasons(context.Background(), 86, []int{17}); !errors.Is(err, ErrDryRun) {
		t.Fatalf("err = %v", err)
	}
	if _, err := c.AddSeries(context.Background(), 74796, 1, "/tv", []int{17}, ""); !errors.Is(err, ErrDryRun) {
		t.Fatalf("add err = %v", err)
	}
	if len(puts)+len(cmds) != 0 {
		t.Fatalf("dry-run wrote: %v %v", puts, cmds)
	}
}
