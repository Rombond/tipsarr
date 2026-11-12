package demo

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBlocked(t *testing.T) {
	for _, c := range []struct {
		m, p string
		want bool
	}{
		{"POST", "/api/v1/admin/webhooks/abc/test", true},
		{"POST", "/api/v1/admin/webhooks", false},
		{"POST", "/api/v1/admin/push/register", true},
		{"POST", "/api/v1/admin/sync/teaser-pool", true},
		{"POST", "/api/v1/admin/sync/library-sync", false},
		{"GET", "/api/v1/auth/oidc/callback", true},
		{"GET", "/api/v1/requests", false},
	} {
		if got := Blocked(c.m, c.p); got != c.want {
			t.Errorf("Blocked(%s %s) = %v", c.m, c.p, got)
		}
	}
}

// Only the allowed address can be reached, whatever the client.
func TestRestrictDial(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer ok.Close()
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer other.Close()
	saved := http.DefaultTransport
	defer func() { http.DefaultTransport = saved }()
	Restrict([]string{strings.TrimPrefix(ok.URL, "http://")})
	c := &http.Client{Timeout: 2 * time.Second}
	if resp, err := c.Get(ok.URL); err != nil {
		t.Fatalf("allowed: %v", err)
	} else {
		resp.Body.Close()
	}
	if _, err := c.Get(other.URL); err == nil || !strings.Contains(err.Error(), "refused") {
		t.Fatalf("other address = %v", err)
	}
	if _, err := c.Get("http://localhost:" + other.URL[strings.LastIndex(other.URL, ":")+1:]); err == nil {
		t.Fatal("a host name to another port went through")
	}
}

func TestLimiter(t *testing.T) {
	l := NewLimiter(20, 2)
	start := time.Now()
	for i := 0; i < 6; i++ {
		if err := l.Wait(context.Background(), time.Second); err != nil {
			t.Fatal(err)
		}
	}
	if d := time.Since(start); d < 150*time.Millisecond {
		t.Fatalf("6 calls at 20/s with burst 2 took only %s", d)
	}
	slow := NewLimiter(0.1, 1)
	_ = slow.Wait(context.Background(), time.Second)
	if err := slow.Wait(context.Background(), 50*time.Millisecond); err != ErrRateLimited {
		t.Fatalf("exhausted = %v", err)
	}
}

// The proxy injects the key, drops the caller's, caches by URL (without the key) and fetches a title once even when
// many sandboxes ask at the same time.
func TestProxyCache(t *testing.T) {
	var calls atomic.Int32
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		time.Sleep(30 * time.Millisecond)
		if r.URL.Query().Get("api_key") != "real" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"path":"`+r.URL.Path+`","lang":"`+r.URL.Query().Get("language")+`"}`)
	}))
	defer up.Close()
	p := NewProxy(t.TempDir(), Upstream{Prefix: "/tmdb/", Base: up.URL, TTL: time.Hour, TMDBKey: "real", Limiter: NewLimiter(100, 10)})
	srv := httptest.NewServer(p)
	defer srv.Close()

	get := func(path string) string {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != 200 {
			t.Fatalf("%s = %d %s", path, resp.StatusCode, b)
		}
		return string(b)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); get("/tmdb/3/movie/550?api_key=sandbox&language=fr-FR") }()
	}
	wg.Wait()
	if body := get("/tmdb/3/movie/550?language=fr-FR&api_key=other"); !strings.Contains(body, `"/3/movie/550"`) || !strings.Contains(body, "fr-FR") {
		t.Fatalf("body = %s", body)
	}
	if n := calls.Load(); n != 1 {
		t.Fatalf("upstream called %d times, want 1", n)
	}
	get("/tmdb/3/movie/550?language=en-US")
	if n := calls.Load(); n != 2 {
		t.Fatalf("another language must be another entry: %d calls", n)
	}
	// a fresh proxy on the same directory reuses the cache (shared across restarts)
	p2 := NewProxy(p.Dir, p.Upstreams...)
	rec := httptest.NewRecorder()
	p2.ServeHTTP(rec, httptest.NewRequest("GET", "/tmdb/3/movie/550?language=fr-FR", nil))
	if rec.Code != 200 || calls.Load() != 2 {
		t.Fatalf("reuse = %d, calls %d", rec.Code, calls.Load())
	}
	rec = httptest.NewRecorder()
	p2.ServeHTTP(rec, httptest.NewRequest("POST", "/tmdb/3/movie/550", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST = %d", rec.Code)
	}
}

// The internal services answer as the fakes Tipsarr expects.
func TestServices(t *testing.T) {
	srv := httptest.NewServer(Services(time.Now(), nil))
	defer srv.Close()
	req, _ := http.NewRequest("GET", srv.URL+PathJellyfin+"/Items?IncludeItemTypes=Movie,Series", nil)
	req.Header.Set("Authorization", `MediaBrowser Token="`+JellyfinAPIKey+`"`)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(b), `"Tmdb":"27205"`) {
		t.Fatalf("library = %.200s", b)
	}
	req, _ = http.NewRequest("POST", srv.URL+PathRadarr+"/api/v3/movie", strings.NewReader("{}"))
	req.Header.Set("X-Api-Key", ArrKey)
	resp, _ = http.DefaultClient.Do(req)
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("radarr write = %d", resp.StatusCode)
	}
}

// The fake Sonarr numbers Bleach like TVDB (17 arcs) and leaves other shows to TMDB.
func TestFakeSonarrBleach(t *testing.T) {
	srv := httptest.NewServer(Services(time.Now(), nil))
	defer srv.Close()
	get := func(path string, v any) {
		t.Helper()
		req, _ := http.NewRequest("GET", srv.URL+PathSonarr+path, nil)
		req.Header.Set("X-Api-Key", ArrKey)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
	}
	type season struct {
		SeasonNumber int `json:"seasonNumber"`
		Statistics   struct {
			TotalEpisodeCount int `json:"totalEpisodeCount"`
		} `json:"statistics"`
	}
	var lookup []struct{ Seasons []season }
	get("/api/v3/series/lookup?term=tvdb:12345", &lookup)
	if len(lookup) != 1 || len(lookup[0].Seasons) != 0 {
		t.Fatalf("unknown lookup = %+v", lookup)
	}
	get("/api/v3/series/lookup?term=tvdb:74796", &lookup)
	if n := len(lookup[0].Seasons); n != 18 {
		t.Fatalf("bleach lookup seasons = %d", n)
	}
	var series []struct {
		ID        int  `json:"id"`
		Monitored bool `json:"monitored"`
		Seasons   []season
	}
	get("/api/v3/series?tvdbId=74796", &series)
	if len(series) != 1 || series[0].ID != 86 || series[0].Monitored || len(series[0].Seasons) != 18 ||
		series[0].Seasons[17].Statistics.TotalEpisodeCount != 50 || series[0].Seasons[0].Statistics.TotalEpisodeCount != 7 {
		t.Fatalf("series = %+v", series)
	}
	get("/api/v3/series?tvdbId=1", &series)
	if len(series) != 0 {
		t.Fatalf("other series = %+v", series)
	}
	var eps []struct {
		SeasonNumber  int    `json:"seasonNumber"`
		EpisodeNumber int    `json:"episodeNumber"`
		AirDate       string `json:"airDate"`
	}
	get("/api/v3/episode?seriesId=86", &eps)
	count := map[int]int{}
	for _, e := range eps {
		count[e.SeasonNumber]++
		if e.SeasonNumber == 17 && e.EpisodeNumber == 1 && e.AirDate != "2022-10-11" {
			t.Fatalf("S17E1 aired %s", e.AirDate)
		}
	}
	if count[17] != 50 || count[14] != 51 || count[0] != 7 {
		t.Fatalf("episode counts = %v", count)
	}
}
