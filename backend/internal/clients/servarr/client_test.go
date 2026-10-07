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

func TestAggregateQueue(t *testing.T) {
	got := AggregateQueue([]QueueItem{
		{MediaID: 7, Size: 1000, SizeLeft: 0, TimeLeft: "00:00:00"},
		{MediaID: 7, Size: 1000, SizeLeft: 500, TimeLeft: "00:10:00"},
		{MediaID: 7, Size: 1000, SizeLeft: 1000, TimeLeft: "00:30:00"}, // just started: used to win and show 0%
		{MediaID: 8, Size: 10, SizeLeft: 5},
	})
	if g := got[7]; g.Percent() != 50 || g.ETASeconds() != 1800 {
		t.Fatalf("series = %d%% eta %d", g.Percent(), g.ETASeconds())
	}
	if got[8].Percent() != 50 {
		t.Fatalf("movie = %d%%", got[8].Percent())
	}
}
