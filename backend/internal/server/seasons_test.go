package server_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/store"
)

type seasonsDetail struct {
	SeasonNumbering        string `json:"seasonNumbering"`
	SeasonNumberingDiffers bool   `json:"seasonNumberingDiffers"`
	NumberOfSeasons        int    `json:"numberOfSeasons"`
	Seasons                []struct {
		Number       int    `json:"number"`
		Name         string `json:"name"`
		EpisodeCount int    `json:"episodeCount"`
		PosterPath   string `json:"posterPath"`
	} `json:"seasons"`
}

func bleachDetail(t *testing.T, e *env, c *http.Cookie) seasonsDetail {
	t.Helper()
	resp, body := call(t, e.app, "GET", "/api/v1/media/tv/30984", "", c)
	var d seasonsDetail
	if resp.StatusCode != 200 || json.Unmarshal([]byte(body), &d) != nil {
		t.Fatalf("detail = %d %s", resp.StatusCode, body)
	}
	return d
}

type seasonDetailBody struct {
	Number   int    `json:"number"`
	Name     string `json:"name"`
	Episodes []struct {
		Number int    `json:"number"`
		Name   string `json:"name"`
		State  string `json:"state"`
	} `json:"episodes"`
}

func bleachSeason(t *testing.T, e *env, c *http.Cookie, n int) seasonDetailBody {
	t.Helper()
	resp, body := call(t, e.app, "GET", fmt.Sprintf("/api/v1/media/tv/30984/seasons/%d", n), "", c)
	var sd seasonDetailBody
	if resp.StatusCode != 200 || json.Unmarshal([]byte(body), &sd) != nil || sd.Episodes == nil {
		t.Fatalf("season %d = %d %s", n, resp.StatusCode, body)
	}
	return sd
}

func regular(d seasonsDetail) int {
	n := 0
	for _, s := range d.Seasons {
		if s.Number > 0 {
			n++
		}
	}
	return n
}

// Sonarr only knows Bleach from its lookup: 17 TVDB seasons of unknown size, nothing to list yet.
func TestBleachSeasonsLookupOnly(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)
	sonarr := newFakeBleachSonarr(t)
	addBleachSonarr(t, e, admin, sonarr)

	d := bleachDetail(t, e, bob)
	if d.SeasonNumbering != "tvdb" || !d.SeasonNumberingDiffers || regular(d) != 17 || d.NumberOfSeasons != 17 {
		t.Fatalf("detail = %+v", d)
	}
	for _, s := range d.Seasons {
		if s.Name == "" || s.EpisodeCount != 0 {
			t.Fatalf("season without counterpart = %+v", s)
		}
	}
	if d.Seasons[5].Name != "Season 5" {
		t.Fatalf("S5 name = %q", d.Seasons[5].Name)
	}
	if sd := bleachSeason(t, e, bob, 5); sd.Number != 5 || len(sd.Episodes) != 0 {
		t.Fatalf("untracked season = %+v", sd)
	}
	if w := sonarr.writes(); len(w) != 0 {
		t.Fatalf("reads only: %v", w)
	}
}

// Sonarr tracks Bleach: its S17 is TMDB's S2, other arcs get Sonarr's episodes.
func TestBleachSeasonsTracked(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)
	sonarr := newFakeBleachSonarr(t)
	sonarr.tracked.Store(true)
	addBleachSonarr(t, e, admin, sonarr)

	d := bleachDetail(t, e, bob)
	if d.SeasonNumbering != "tvdb" || !d.SeasonNumberingDiffers || regular(d) != 17 {
		t.Fatalf("detail = %+v", d)
	}
	s17 := d.Seasons[len(d.Seasons)-1]
	if s17.Number != 17 || s17.Name != "Thousand-Year Blood War" || s17.EpisodeCount != 50 || s17.PosterPath != "/tybw.jpg" {
		t.Fatalf("S17 = %+v", s17)
	}
	if s1 := d.Seasons[1]; s1.Number != 1 || s1.EpisodeCount != 20 || s1.Name != "Season 1" {
		t.Fatalf("S1 = %+v", s1)
	}

	sd := bleachSeason(t, e, bob, 17)
	if sd.Number != 17 || sd.Name != "Thousand-Year Blood War" || len(sd.Episodes) != 3 || sd.Episodes[0].Name != "The Blood Warfare" || sd.Episodes[0].State != "available" || sd.Episodes[1].State != "missing" {
		t.Fatalf("S17 episodes = %+v", sd)
	}
	sd = bleachSeason(t, e, bob, 5)
	if sd.Number != 5 || len(sd.Episodes) != 2 || sd.Episodes[0].Name != "Arc 5 episode 1" || sd.Episodes[0].State != "available" {
		t.Fatalf("S5 episodes = %+v", sd)
	}
	if resp, _ := call(t, e.app, "GET", "/api/v1/media/tv/30984/seasons/18", "", bob); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("S18 = %d", resp.StatusCode)
	}
}

// Sonarr down: TMDB seasons quickly, unless an earlier answer is cached.
func TestBleachSeasonsSonarrDown(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)
	sonarr := newFakeBleachSonarr(t)
	addBleachSonarr(t, e, admin, sonarr)

	sonarr.down.Store(true)
	start := time.Now()
	d := bleachDetail(t, e, bob)
	if time.Since(start) > 5*time.Second || d.SeasonNumbering != "tmdb" || d.SeasonNumberingDiffers || regular(d) != 2 {
		t.Fatalf("down = %+v in %v", d, time.Since(start))
	}

	sonarr.down.Store(false)
	sonarr.tracked.Store(true)
	if d := bleachDetail(t, e, bob); d.SeasonNumbering != "tvdb" {
		t.Fatalf("up = %+v", d)
	}
	sonarr.down.Store(true)
	if d := bleachDetail(t, e, bob); d.SeasonNumbering != "tvdb" || regular(d) != 17 || d.Seasons[17].Name != "Thousand-Year Blood War" {
		t.Fatalf("stale = %+v", d)
	}
}

func bleachRequest(t *testing.T, e *env, id string) (*store.Request, []int) {
	t.Helper()
	ctx := context.Background()
	r, err := e.st.GetRequest(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	s, err := e.st.RequestSeasons(ctx, []string{id})
	if err != nil {
		t.Fatal(err)
	}
	return r, s[id]
}

// Asking for S17 sends Sonarr's S17, never S2 (the TMDB number).
func TestBleachRequestLive(t *testing.T) {
	e := newEnv(t, false)
	_, admin, bob := setupUsers(t, e)
	sonarr := newFakeBleachSonarr(t)
	sonarr.tracked.Store(true)
	addBleachSonarr(t, e, admin, sonarr)

	resp, body := call(t, e.app, "POST", "/api/v1/requests", `{"type":"tv","tmdbId":30984,"seasons":[17],"seasonNumbering":"tmdb"}`, bob)
	if resp.StatusCode != http.StatusConflict || !strings.Contains(body, "season_numbering_changed") {
		t.Fatalf("stale numbering = %d %s", resp.StatusCode, body)
	}
	resp, body = call(t, e.app, "POST", "/api/v1/requests", `{"type":"tv","tmdbId":30984,"seasons":[17],"seasonNumbering":"tvdb"}`, admin)
	v := decodeReq(t, body)
	if resp.StatusCode != http.StatusCreated || v.Status != "approved" || v.DryRun || !strings.Contains(body, `"seasonNumbering":"tvdb"`) {
		t.Fatalf("create = %d %s", resp.StatusCode, body)
	}
	r, seasons := bleachRequest(t, e, v.ID)
	if r.SeasonNumbering != "tvdb" || fmt.Sprint(seasons) != "[17]" {
		t.Fatalf("stored = %q %v", r.SeasonNumbering, seasons)
	}
	sonarr.mu.Lock()
	defer sonarr.mu.Unlock()
	if sonarr.put == nil || len(sonarr.commands) != 1 || sonarr.commands[0]["seasonNumber"] != float64(17) || sonarr.commands[0]["name"] != "SeasonSearch" {
		t.Fatalf("put = %v commands = %v", sonarr.put != nil, sonarr.commands)
	}
	for _, s := range sonarr.put["seasons"].([]any) {
		m := s.(map[string]any)
		if n := m["seasonNumber"].(float64); (n == 17) != (m["monitored"] == true) {
			t.Fatalf("season %v monitored = %v", n, m["monitored"])
		}
	}
}

// Old TMDB-numbered requests: one season cannot be translated, all of them become every Sonarr season.
func TestBleachLegacyTMDBRequests(t *testing.T) {
	e := newEnv(t, false)
	_, admin, bob := setupUsers(t, e)
	sonarr := newFakeBleachSonarr(t)
	addBleachSonarr(t, e, admin, sonarr) // not tracked: an approval adds the series
	ctx := context.Background()
	mk := func(seasons ...int) string {
		r := &store.Request{ID: store.NewID(), MediaType: "tv", TMDBID: 30984, Title: "Bleach", Status: store.StatusPending, RequestedBy: "x", SeasonNumbering: "tmdb"}
		if err := e.st.CreateRequest(ctx, r, seasons); err != nil {
			t.Fatal(err)
		}
		return r.ID
	}

	one := mk(2)
	_, body := call(t, e.app, "POST", "/api/v1/requests/"+one+"/approve", `{}`, admin)
	if v := decodeReq(t, body); v.Status != "failed" || !strings.Contains(v.Error, "number this show's seasons differently") {
		t.Fatalf("one season = %s", body)
	}
	if w := sonarr.writes(); len(w) != 0 {
		t.Fatalf("wrong seasons sent: %v", w)
	}
	call(t, e.app, "DELETE", "/api/v1/requests/"+one, "", admin)

	all := mk(1, 2)
	_, body = call(t, e.app, "POST", "/api/v1/requests/"+all+"/approve", `{}`, admin)
	if v := decodeReq(t, body); v.Status != "approved" || v.DryRun {
		t.Fatalf("every season = %s", body)
	}
	r, seasons := bleachRequest(t, e, all)
	if r.SeasonNumbering != "tvdb" || len(seasons) != 17 || seasons[0] != 1 || seasons[16] != 17 {
		t.Fatalf("translated = %q %v", r.SeasonNumbering, seasons)
	}
	sonarr.mu.Lock()
	defer sonarr.mu.Unlock()
	for _, s := range sonarr.added["seasons"].([]any) {
		m := s.(map[string]any)
		if n := m["seasonNumber"].(float64); (n > 0) != (m["monitored"] == true) {
			t.Fatalf("added season %v monitored = %v", n, m["monitored"])
		}
	}
	_ = bob
}

// Dry-run: a request for a series Sonarr has is approved without any write.
func TestBleachRequestDryRun(t *testing.T) {
	e := newEnv(t, true)
	_, admin, _ := setupUsers(t, e)
	sonarr := newFakeBleachSonarr(t)
	sonarr.tracked.Store(true)
	addBleachSonarr(t, e, admin, sonarr)
	resp, body := call(t, e.app, "POST", "/api/v1/requests", `{"type":"tv","tmdbId":30984,"seasons":[17]}`, admin)
	if v := decodeReq(t, body); resp.StatusCode != http.StatusCreated || v.Status != "approved" || !v.DryRun {
		t.Fatalf("dry-run = %d %s", resp.StatusCode, body)
	}
	if w := sonarr.writes(); len(w) != 0 {
		t.Fatalf("DRY-RUN LEAK: %v", w)
	}
}

// Imported requests carry Sonarr's own numbers.
func TestBleachImportIsTVDB(t *testing.T) {
	e := newEnv(t, true)
	_, admin, _ := setupUsers(t, e)
	sonarr := newFakeBleachSonarr(t)
	sonarr.tracked.Store(true)
	sonarr.monitored.Store(true)
	addBleachSonarr(t, e, admin, sonarr)
	if _, err := e.reqs.ImportFromServarr(context.Background()); err != nil {
		t.Fatal(err)
	}
	_, body := call(t, e.app, "GET", "/api/v1/requests", "", admin)
	if !strings.Contains(body, `"total":1`) || !strings.Contains(body, `"seasonNumbering":"tvdb"`) || !strings.Contains(body, `"seasons":[1]`) {
		t.Fatalf("imported = %s", body)
	}
}
