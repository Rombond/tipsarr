package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/store"
	"github.com/Rombond/tipsarr/backend/internal/suggestions"
)

func getSuggestions(t *testing.T, e *env, c *http.Cookie) (suggestions.Result, int) {
	t.Helper()
	resp, body := call(t, e.app, "GET", "/api/v1/suggestions", "", c)
	var res suggestions.Result
	if resp.StatusCode == 200 {
		if err := json.Unmarshal([]byte(body), &res); err != nil {
			t.Fatalf("decode %s: %v", body, err)
		}
	}
	return res, resp.StatusCode
}

func TestSuggestions(t *testing.T) {
	e := newEnv(t, true)
	jfURL, admin, bob := setupUsers(t, e)
	_ = jfURL
	ctx := context.Background()

	// without a TMDB key the rows cannot be built
	e2 := newEnv(t, true)
	jf2 := fakeJellyfin(t)
	c2 := loginAs(t, e2.app, jf2.URL, "bob", "hunter2")
	if _, code := getSuggestions(t, e2, c2); code != http.StatusServiceUnavailable {
		t.Fatalf("no TMDB key = %d", code)
	}

	// alice (admin) has history: sync it from the fake Jellyfin
	call(t, e.app, "PUT", "/api/v1/admin/settings", `{"jellyfinApiKey":"jfkey"}`, admin)
	call(t, e.app, "POST", "/api/v1/admin/sync/history-sync", "", admin)
	const alice = "aaaaaaaabbbbccccddddeeeeeeeeeeee"
	waitFor(t, "history", func() bool { v, _ := e.st.HistoryVersion(ctx, alice); return v == 1 })

	res, code := getSuggestions(t, e, admin)
	if code != 200 || len(res.Rows) != 3 {
		t.Fatalf("alice rows = %d (%d): %+v", len(res.Rows), code, res)
	}
	acc := res.Rows[0]
	if acc.Kind != "account" || acc.Title != "Recommended for you" || !acc.Personal {
		t.Fatalf("account row = %+v", acc)
	}
	ids := map[string]bool{}
	for _, it := range acc.Items {
		ids[it.Type+":"+itoa(it.TMDBID)] = true
	}
	if ids["movie:1"] || ids["tv:2"] {
		t.Fatalf("watched titles must be hidden: %v", ids)
	}
	if !ids["movie:11"] || !ids["tv:20"] || len(acc.Items) < 8 {
		t.Fatalf("account items = %v", ids)
	}
	// movie 11 is recommended AND similar for the first seed, so it outranks plain picks
	if acc.Items[0].TMDBID != 11 && acc.Items[0].TMDBID != 20 {
		t.Fatalf("expected multi-signal title first, got %d", acc.Items[0].TMDBID)
	}
	var because []string
	for _, r := range res.Rows[1:] {
		if r.Kind != "because" || r.Seed == nil || !strings.HasPrefix(r.Title, "Because you watched ") {
			t.Fatalf("because row = %+v", r)
		}
		because = append(because, r.Seed.Title)
		for _, it := range r.Items {
			if it.TMDBID == 1 || it.TMDBID == 2 {
				t.Fatalf("seed/watched in row: %+v", it)
			}
		}
	}
	if len(because) != 2 || !(contains(because, "Movie One") && contains(because, "Show Two")) {
		t.Fatalf("because seeds = %v", because)
	}

	// served from the DB: no extra TMDB calls
	before := tmdbHits.Load()
	if again, _ := getSuggestions(t, e, admin); len(again.Rows) != 3 || again.Generating {
		t.Fatalf("second read = %+v", again)
	}
	if tmdbHits.Load() != before {
		t.Fatal("reading stored suggestions must not call TMDB")
	}

	// bob has no history: seeded from what the server watches, no "because" rows
	bres, _ := getSuggestions(t, e, bob)
	if len(bres.Rows) != 1 || bres.Rows[0].Title != "Popular on this server" || bres.Rows[0].Personal {
		t.Fatalf("bob rows = %+v", bres)
	}

	// availability and requests are applied at read time
	call(t, e.app, "POST", "/api/v1/requests", `{"type":"movie","tmdbId":5}`, bob) // unrelated to rows: just must not break
	e.st.ReplaceLibrary(ctx, []store.LibraryItem{{MediaType: "movie", TMDBID: 11, JellyfinID: "x", Title: "R11"}}, nil)
	res, _ = getSuggestions(t, e, admin)
	found := false
	for _, it := range res.Rows[0].Items {
		if it.TMDBID == 11 {
			found = it.Availability == "available"
		}
	}
	if !found {
		t.Fatal("library availability not applied to suggestion items")
	}

	// new watch -> history version changes -> stale rows served + background refresh
	rows, _ := e.st.UserHistory(ctx, alice, 0)
	rows = append(rows, store.WatchHistory{MediaType: "movie", TMDBID: 10, LastPlayedAt: time.Now().Unix(), PlayCount: 1})
	if changed, err := e.st.ReplaceUserHistory(ctx, alice, rows); err != nil || !changed {
		t.Fatalf("history change: %v %v", changed, err)
	}
	res, _ = getSuggestions(t, e, admin)
	if !res.Generating {
		t.Fatal("expected generating=true after history changed")
	}
	waitFor(t, "background refresh", func() bool {
		r, _ := getSuggestions(t, e, admin)
		if r.Generating || len(r.Rows) == 0 {
			return false
		}
		for _, it := range r.Rows[0].Items {
			if it.TMDBID == 10 {
				return false // movie 10 is watched now and must disappear
			}
		}
		return true
	})

	// forcing a refresh is rate limited
	if resp, _ := call(t, e.app, "POST", "/api/v1/suggestions/refresh", "", admin); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("refresh = %d", resp.StatusCode)
	}
	if resp, _ := call(t, e.app, "POST", "/api/v1/suggestions/refresh", "", admin); resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("second refresh = %d", resp.StatusCode)
	}
	if resp, _ := call(t, e.app, "GET", "/api/v1/suggestions", ""); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous = %d", resp.StatusCode)
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func itoa(n int) string { return strconv.Itoa(n) }
