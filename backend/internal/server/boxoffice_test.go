package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/boxoffice"
	"github.com/Rombond/tipsarr/backend/internal/store"
)

func getChart(t *testing.T, e *env, c *http.Cookie, query string) (boxoffice.Chart, int) {
	t.Helper()
	var cookies []*http.Cookie
	if c != nil {
		cookies = append(cookies, c)
	}
	resp, body := call(t, e.app, "GET", "/api/v1/boxoffice"+query, "", cookies...)
	var ch boxoffice.Chart
	if resp.StatusCode == 200 {
		if err := json.Unmarshal([]byte(body), &ch); err != nil {
			t.Fatalf("decode %s: %v", body, err)
		}
	}
	return ch, resp.StatusCode
}

func TestBoxOffice(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)
	ctx := context.Background()

	// nothing stored yet: an empty chart, not an error
	if ch, code := getChart(t, e, bob, ""); code != 200 || len(ch.Entries) != 0 || ch.Region != "US" {
		t.Fatalf("empty chart = %d %+v", code, ch)
	}

	// regions: normalised, invalid codes dropped
	_, body := call(t, e.app, "PUT", "/api/v1/admin/settings", `{"boxofficeRegions":"us, gb ,xx!, GB"}`, admin)
	if !strings.Contains(body, `"boxofficeRegions":"US,GB"`) {
		t.Fatalf("regions = %s", body)
	}

	// admin refreshes (job); the page is fetched per region and per weekend
	if resp, _ := call(t, e.app, "POST", "/api/v1/admin/sync/boxoffice-refresh", "", admin); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("run job = %d", resp.StatusCode)
	}
	waitFor(t, "box office refresh", func() bool {
		_, b := call(t, e.app, "GET", "/api/v1/admin/sync", "", admin)
		return strings.Contains(b, `"name":"boxoffice-refresh"`) && strings.Contains(b, `charts stored"`)
	})
	bomHits.Lock()
	hitGB := false
	for _, u := range bomHits.urls {
		if strings.Contains(u, "/weekend/2026W40/") && strings.Contains(u, "area=GB") {
			hitGB = true
		}
		if strings.Contains(u, "/weekend/2026W40/") && !strings.Contains(u, "area") {
			continue
		}
	}
	bomHits.Unlock()
	if !hitGB {
		t.Fatal("the GB chart should be fetched with ?area=GB")
	}

	ch, code := getChart(t, e, bob, "")
	if code != 200 || ch.Region != "US" || ch.Week != "2026W40" || ch.Label != "October 2-4, 2026" || len(ch.Entries) != 5 || len(ch.Weeks) != 8 { // history is filled up to 8 weekends
		t.Fatalf("chart = %d region=%s week=%s label=%q entries=%d weeks=%d", code, ch.Region, ch.Week, ch.Label, len(ch.Entries), len(ch.Weeks))
	}
	if len(ch.Regions) != 2 {
		t.Fatalf("regions = %v", ch.Regions)
	}
	byTitle := map[string]boxoffice.ChartEntry{}
	for _, en := range ch.Entries {
		byTitle[en.Title] = en
	}
	v := byTitle["Verity"]
	if v.Position != 1 || v.WeekendGross != 32031011 || v.Item == nil || v.Item.TMDBID != 900 || v.Item.Type != "movie" {
		t.Fatalf("Verity = %+v", v)
	}
	if re := byTitle["Resident Evil"]; re.Item == nil || re.Item.TMDBID != 902 {
		t.Fatalf("Resident Evil should match the recent release: %+v", re.Item)
	}
	if hb := byTitle["Heart of the Beast"]; hb.Item == nil || hb.Item.TMDBID != 903 {
		t.Fatalf("recent partial title match: %+v", hb.Item)
	}
	unmatched := 0
	for _, en := range ch.Entries {
		if en.Item == nil {
			unmatched++
		}
	}
	if unmatched != 2 {
		t.Fatalf("expected 2 unmatched titles, got %d", unmatched)
	}

	// other region and an older stored week
	if gb, code := getChart(t, e, bob, "?region=GB"); code != 200 || gb.Region != "GB" || len(gb.Entries) != 5 {
		t.Fatalf("GB chart = %d %+v", code, gb.Region)
	}
	if old, code := getChart(t, e, bob, "?week=2026W39"); code != 200 || old.Week != "2026W39" {
		t.Fatalf("history week = %d %s", code, old.Week)
	}
	if _, code := getChart(t, e, bob, "?week=2026W01"); code != http.StatusNotFound {
		t.Fatalf("unknown week = %d", code)
	}
	if _, code := getChart(t, e, nil, ""); code != http.StatusUnauthorized {
		t.Fatalf("anonymous = %d", code)
	}

	// availability + Radarr tracking are applied at read time (read-only calls to Radarr)
	e.st.ReplaceLibrary(ctx, []store.LibraryItem{{MediaType: "movie", TMDBID: 902, JellyfinID: "j", Title: "Resident Evil"}}, nil)
	radarr := newFakeArr(t, "radarr")
	radarr.moviesJSON.Store(`[{"id":7,"tmdbId":900,"hasFile":false},{"id":8,"tmdbId":903,"hasFile":true}]`)
	addInstance(t, e, admin, "radarr", radarr)
	ch, _ = getChart(t, e, bob, "")
	for _, en := range ch.Entries {
		switch en.Title {
		case "Verity":
			if !en.InRadarr || en.HasFile {
				t.Fatalf("Verity radarr flags = %+v", en)
			}
		case "Heart of the Beast":
			if !en.InRadarr || !en.HasFile {
				t.Fatalf("Heart flags = %+v", en)
			}
		case "Resident Evil":
			if en.InRadarr || en.Item.Availability != "available" {
				t.Fatalf("Resident Evil = %+v", en)
			}
		}
	}
	if w := radarr.writes(); len(w) != 0 {
		t.Fatalf("box office must only read from Radarr: %v", w)
	}
}

func TestWeekKey(t *testing.T) {
	cases := map[string]string{
		"2026-10-06": "2026W40", // Tuesday -> Sunday Oct 4
		"2026-10-04": "2026W40", // Sunday itself
		"2026-10-03": "2026W39", // Saturday -> previous Sunday Sep 27
		"2026-01-01": "2025W52", // across the year boundary
	}
	for day, want := range cases {
		d, _ := parseDay(day)
		if got := boxoffice.WeekKey(d, 0); got != want {
			t.Fatalf("WeekKey(%s) = %s, want %s", day, got, want)
		}
	}
	d, _ := parseDay("2026-10-06")
	if got := boxoffice.WeekKey(d, 1); got != "2026W39" {
		t.Fatalf("back=1 -> %s", got)
	}
}

func TestGenreRootFolder(t *testing.T) {
	e := newEnv(t, false)
	_, admin, _ := setupUsers(t, e)
	radarr := newFakeArr(t, "radarr")
	addInstance(t, e, admin, "radarr", radarr)
	_, body := call(t, e.app, "GET", "/api/v1/admin/servarr", "", admin)
	var list []struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal([]byte(body), &list)

	if resp, _ := call(t, e.app, "PUT", "/api/v1/admin/servarr/"+list[0].ID,
		`{"kind":"radarr","name":"radarr","url":"`+radarr.srv.URL+`","qualityProfileId":4,"rootFolder":"/data/media","isDefault":true,"genreRoots":{"abc":"/x"}}`, admin); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("non-numeric genre id = %d", resp.StatusCode)
	}
	resp, body := call(t, e.app, "PUT", "/api/v1/admin/servarr/"+list[0].ID,
		`{"kind":"radarr","name":"radarr","url":"`+radarr.srv.URL+`","qualityProfileId":4,"rootFolder":"/data/media","isDefault":true,"genreRoots":{"28":"/data/action"}}`, admin)
	if resp.StatusCode != 200 || !strings.Contains(body, `"/data/action"`) {
		t.Fatalf("update = %d %s", resp.StatusCode, body)
	}

	// movie 1 is an Action movie (genre 28): admin requests it -> goes to the Action folder
	if resp, b := call(t, e.app, "POST", "/api/v1/requests", `{"type":"movie","tmdbId":1}`, admin); resp.StatusCode != http.StatusCreated {
		t.Fatalf("request = %d %s", resp.StatusCode, b)
	}
	if radarr.posted["rootFolderPath"] != "/data/action" {
		t.Fatalf("genre root not applied: %v", radarr.posted["rootFolderPath"])
	}
	// a movie without a mapped genre uses the instance default
	if resp, _ := call(t, e.app, "POST", "/api/v1/requests", `{"type":"movie","tmdbId":5}`, admin); resp.StatusCode != http.StatusCreated {
		t.Fatalf("request 5 = %d", resp.StatusCode)
	}
	if radarr.posted["rootFolderPath"] != "/data/media" {
		t.Fatalf("default root = %v", radarr.posted["rootFolderPath"])
	}
	// omitting genreRoots on update keeps the saved map
	_, body = call(t, e.app, "PUT", "/api/v1/admin/servarr/"+list[0].ID,
		`{"kind":"radarr","name":"radarr","url":"`+radarr.srv.URL+`","qualityProfileId":4,"rootFolder":"/data/media","isDefault":true}`, admin)
	if !strings.Contains(body, `"/data/action"`) {
		t.Fatalf("genre roots lost on update: %s", body)
	}
}

func parseDay(s string) (time.Time, error) { return time.Parse("2006-01-02", s) }

func TestBoxOfficeAlias(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)
	call(t, e.app, "POST", "/api/v1/admin/sync/boxoffice-refresh", "", admin)
	waitFor(t, "refresh", func() bool {
		ch, code := getChart(t, e, bob, "")
		return code == 200 && len(ch.Entries) == 5
	})
	entry := func(week, title string) boxoffice.ChartEntry {
		ch, _ := getChart(t, e, bob, "?week="+week)
		for _, en := range ch.Entries {
			if en.Title == title {
				return en
			}
		}
		t.Fatalf("%s not in %s", title, week)
		return boxoffice.ChartEntry{}
	}
	if entry("2026W40", "Digger").Item != nil {
		t.Fatal("Digger should start unmatched")
	}

	// only admins may pin matches
	if resp, _ := call(t, e.app, "PUT", "/api/v1/admin/boxoffice/alias", `{"title":"Digger","tmdbId":904}`, bob); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("user alias = %d", resp.StatusCode)
	}
	if resp, _ := call(t, e.app, "PUT", "/api/v1/admin/boxoffice/alias", `{"title":"Digger","tmdbId":999999}`, admin); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown movie = %d", resp.StatusCode)
	}
	if resp, _ := call(t, e.app, "PUT", "/api/v1/admin/boxoffice/alias", `{"title":"digger","tmdbId":904}`, admin); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("alias = %d", resp.StatusCode) // case-insensitive title
	}
	// applied to every stored week at once
	for _, wk := range []string{"2026W40", "2026W39"} {
		if it := entry(wk, "Digger").Item; it == nil || it.TMDBID != 904 || it.Overview != "Pinned by hand." {
			t.Fatalf("%s Digger = %+v", wk, it)
		}
	}
	// ... and survives the next refresh (the alias wins over the search)
	call(t, e.app, "POST", "/api/v1/admin/sync/boxoffice-refresh", "", admin)
	time.Sleep(300 * time.Millisecond)
	if it := entry("2026W40", "Digger").Item; it == nil || it.TMDBID != 904 {
		t.Fatalf("alias lost after refresh: %+v", it)
	}

	// removing it goes back to the automatic result (no match for this title)
	if resp, _ := call(t, e.app, "DELETE", "/api/v1/admin/boxoffice/alias?title=Digger", "", admin); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete alias = %d", resp.StatusCode)
	}
	if entry("2026W40", "Digger").Item != nil {
		t.Fatal("Digger should be unmatched again")
	}
}
