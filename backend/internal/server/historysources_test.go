package server_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// TestHistorySources: an admin lets bob (no history) use alice's history for his recommendations.
func TestHistorySources(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)
	ctx := context.Background()
	const alice, bobID = "aaaaaaaabbbbccccddddeeeeeeeeeeee", "bbbbbbbbbbbbccccddddeeeeeeeeeeee"
	path := "/api/v1/admin/users/" + bobID + "/history-sources"

	call(t, e.app, "PUT", "/api/v1/admin/settings", `{"jellyfinApiKey":"jfkey"}`, admin)
	call(t, e.app, "POST", "/api/v1/admin/sync/history-sync", "", admin)
	waitFor(t, "history", func() bool { v, _ := e.st.HistoryVersion(ctx, alice); return v == 1 })

	titles := func(c *http.Cookie) (first string, personal bool, because []string, ids map[string]bool) {
		res, code := getSuggestions(t, e, c)
		if code != 200 || len(res.Rows) == 0 {
			t.Fatalf("rows = %d (%d)", len(res.Rows), code)
		}
		ids = map[string]bool{}
		for _, r := range res.Rows {
			if r.Kind == "because" && r.Seed != nil {
				because = append(because, r.Seed.Title)
			}
			for _, it := range r.Items {
				ids[it.Type+":"+itoa(it.TMDBID)] = true
			}
		}
		return res.Rows[0].Title, res.Rows[0].Personal, because, ids
	}

	// before: bob has no history of his own, so the server's popular titles stand in
	if first, personal, because, _ := titles(bob); first != "Popular on this server" || personal || len(because) != 0 {
		t.Fatalf("bob before = %q personal=%v because=%v", first, personal, because)
	}
	// alice has no sources: her stored rows keep the plain history version (no regeneration storm at deploy)
	aliceVersion, _ := e.st.HistoryVersion(ctx, alice)
	titles(admin)
	if rows, _, _ := e.st.SuggestionRows(ctx, alice); len(rows) == 0 || rows[0].HistoryVersion != aliceVersion {
		t.Fatalf("alice row version = %+v, want %d", rows, aliceVersion)
	}
	if resp, body := call(t, e.app, "GET", "/api/v1/me/history-sources", "", bob); resp.StatusCode != 200 || !strings.Contains(body, `"sources":[]`) {
		t.Fatalf("my sources before = %d %s", resp.StatusCode, body)
	}

	// who may set it, and what is refused
	if resp, _ := call(t, e.app, "PUT", path, `{"sources":["`+alice+`"]}`, bob); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("user sets sources = %d, want 403", resp.StatusCode)
	}
	_, body := call(t, e.app, "PUT", "/api/v1/admin/settings", `{"apiKeyEnabled":true}`, admin)
	key := body[strings.Index(body, `"apiKey":"`)+len(`"apiKey":"`):][:32]
	if resp, _ := keyCall(t, e.app.URL, "PUT", path, `{"sources":["`+alice+`"]}`, key); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("read-only key sets sources = %d, want 403", resp.StatusCode)
	}
	if resp, _ := call(t, e.app, "PUT", path, `{"sources":["`+bobID+`"]}`, admin); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("self source = %d, want 422", resp.StatusCode)
	}
	if resp, _ := call(t, e.app, "PUT", path, `{"sources":["nobody-here"]}`, admin); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("unknown source = %d, want 422", resp.StatusCode)
	}
	if resp, _ := call(t, e.app, "PUT", "/api/v1/admin/users/nobody-here/history-sources", `{"sources":[]}`, admin); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown person = %d, want 404", resp.StatusCode)
	}

	// alice blocks a title: that must stay hers
	if resp, _ := call(t, e.app, "POST", "/api/v1/blocklist", `{"type":"movie","tmdbId":11}`, admin); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("blocklist = %d", resp.StatusCode)
	}

	// map bob to alice (twice: deduplicated)
	resp, body := call(t, e.app, "PUT", path, `{"sources":["`+alice+`","`+alice+`"]}`, admin)
	if resp.StatusCode != 200 || !strings.Contains(body, `"id":"`+alice+`"`) || strings.Count(body, alice) != 1 {
		t.Fatalf("set sources = %d %s", resp.StatusCode, body)
	}
	if _, b := call(t, e.app, "GET", "/api/v1/me/history-sources", "", bob); !strings.Contains(b, `"name":"alice"`) {
		t.Fatalf("my sources after = %s", b)
	}
	var ids map[string]bool
	var because []string
	waitFor(t, "bob's rows built from alice's history", func() bool {
		first, personal, b, i := titles(bob)
		because, ids = b, i
		return first == "Recommended for you" && personal
	})
	if !contains(because, "Movie One") || !contains(because, "Show Two") {
		t.Fatalf("bob because rows = %v, want alice's seeds", because)
	}
	if ids["movie:1"] || ids["tv:2"] {
		t.Fatalf("titles alice watched must be hidden from bob: %v", ids)
	}
	if !ids["movie:11"] {
		t.Fatal("alice's blocklist must not hide titles from bob")
	}

	// two accounts mapped to each other do not loop
	if resp, _ := call(t, e.app, "PUT", "/api/v1/admin/users/"+alice+"/history-sources", `{"sources":["`+bobID+`"]}`, admin); resp.StatusCode != 200 {
		t.Fatalf("alice to bob = %d", resp.StatusCode)
	}
	if _, code := getSuggestions(t, e, admin); code != 200 {
		t.Fatalf("alice after mutual mapping = %d", code)
	}
	call(t, e.app, "PUT", "/api/v1/admin/users/"+alice+"/history-sources", `{"sources":[]}`, admin)

	// remove the mapping: bob is back to the server's popular titles
	if resp, _ := call(t, e.app, "PUT", path, `{"sources":[]}`, admin); resp.StatusCode != 200 {
		t.Fatalf("clear = %d", resp.StatusCode)
	}
	waitFor(t, "bob back on server titles", func() bool {
		first, personal, _, _ := titles(bob)
		return first == "Popular on this server" && !personal
	})
}
