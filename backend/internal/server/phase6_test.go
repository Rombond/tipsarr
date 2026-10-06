package server_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestAnimeFolder(t *testing.T) {
	e := newEnv(t, false)
	_, admin, bob := setupUsers(t, e)
	sonarr := newFakeArr(t, "sonarr")
	resp, body := call(t, e.app, "POST", "/api/v1/admin/servarr",
		`{"kind":"sonarr","name":"sonarr","url":"`+sonarr.srv.URL+`","apiKey":"rk","qualityProfileId":4,"rootFolder":"/data/tv","isDefault":true,"animeRoot":"/data/anime"}`, admin)
	if resp.StatusCode != http.StatusCreated || !strings.Contains(body, `"animeRoot":"/data/anime"`) {
		t.Fatalf("create sonarr = %d %s", resp.StatusCode, body)
	}

	// an anime show (Animation + Japanese) goes to the anime folder with series type anime
	if resp, b := call(t, e.app, "POST", "/api/v1/requests", `{"type":"tv","tmdbId":30}`, admin); resp.StatusCode != http.StatusCreated {
		t.Fatalf("anime request = %d %s", resp.StatusCode, b)
	}
	if sonarr.posted["rootFolderPath"] != "/data/anime" || sonarr.posted["seriesType"] != "anime" {
		t.Fatalf("anime placement = %v / %v", sonarr.posted["rootFolderPath"], sonarr.posted["seriesType"])
	}
	// a regular show uses the default folder and keeps Sonarr's own series type
	if resp, b := call(t, e.app, "POST", "/api/v1/requests", `{"type":"tv","tmdbId":2,"seasons":[1]}`, admin); resp.StatusCode != http.StatusCreated {
		t.Fatalf("regular request = %d %s", resp.StatusCode, b)
	}
	if sonarr.posted["rootFolderPath"] != "/data/tv" || sonarr.posted["seriesType"] != nil {
		t.Fatalf("regular placement = %v / %v", sonarr.posted["rootFolderPath"], sonarr.posted["seriesType"])
	}

	// the anime show now has an active request, so a second one is a duplicate
	if resp, b := call(t, e.app, "POST", "/api/v1/requests", `{"type":"tv","tmdbId":30}`, bob); resp.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate anime request = %d %s", resp.StatusCode, b)
	}
}

func TestAnimeFolderNeedsConfig(t *testing.T) {
	e := newEnv(t, false)
	_, admin, _ := setupUsers(t, e)
	sonarr := newFakeArr(t, "sonarr")
	addInstance(t, e, admin, "sonarr", sonarr) // no anime folder configured
	call(t, e.app, "POST", "/api/v1/requests", `{"type":"tv","tmdbId":30}`, admin)
	if sonarr.posted["rootFolderPath"] != "/data/media" || sonarr.posted["seriesType"] != nil {
		t.Fatalf("without an anime folder nothing special happens: %v / %v", sonarr.posted["rootFolderPath"], sonarr.posted["seriesType"])
	}
}

func TestWatchlistAndBlocklist(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)

	if resp, _ := call(t, e.app, "GET", "/api/v1/watchlist", ""); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous = %d", resp.StatusCode)
	}
	if resp, _ := call(t, e.app, "POST", "/api/v1/watchlist", `{"type":"movie","tmdbId":5}`, bob); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("add = %d", resp.StatusCode)
	}
	call(t, e.app, "POST", "/api/v1/watchlist", `{"type":"movie","tmdbId":5}`, bob) // idempotent
	if resp, _ := call(t, e.app, "POST", "/api/v1/watchlist", `{"type":"movie","tmdbId":999999}`, bob); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown title = %d", resp.StatusCode)
	}
	_, body := call(t, e.app, "GET", "/api/v1/watchlist", "", bob)
	var list []struct {
		Title string `json:"title"`
		TMDB  int    `json:"tmdbId"`
	}
	_ = json.Unmarshal([]byte(body), &list)
	if len(list) != 1 || list[0].Title != "Request Me" {
		t.Fatalf("watchlist = %s", body)
	}
	// per user: alice sees nothing
	if _, b := call(t, e.app, "GET", "/api/v1/watchlist", "", admin); strings.TrimSpace(b) != "[]" {
		t.Fatalf("alice watchlist = %s", b)
	}
	if _, b := call(t, e.app, "GET", "/api/v1/media/movie/5/flags", "", bob); !strings.Contains(b, `"watchlisted":true`) || !strings.Contains(b, `"blocklisted":false`) {
		t.Fatalf("flags = %s", b)
	}
	if resp, _ := call(t, e.app, "DELETE", "/api/v1/watchlist/movie/5", "", bob); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("remove = %d", resp.StatusCode)
	}
	if _, b := call(t, e.app, "GET", "/api/v1/watchlist", "", bob); strings.TrimSpace(b) != "[]" {
		t.Fatalf("after remove = %s", b)
	}

	// blocklist hides a title from the user's suggestions, immediately
	call(t, e.app, "PUT", "/api/v1/admin/settings", `{"jellyfinApiKey":"jfkey"}`, admin)
	call(t, e.app, "POST", "/api/v1/admin/sync/history-sync", "", admin)
	waitFor(t, "history", func() bool {
		v, _ := e.st.HistoryVersion(t.Context(), "aaaaaaaabbbbccccddddeeeeeeeeeeee")
		return v == 1
	})
	res, _ := getSuggestions(t, e, admin)
	has := func(id int) bool {
		for _, r := range getRows(t, e, admin) {
			for _, it := range r.Items {
				if it.TMDBID == id {
					return true
				}
			}
		}
		return false
	}
	_ = res
	if !has(11) {
		t.Fatal("movie 11 should be suggested before it is blocklisted")
	}
	if resp, _ := call(t, e.app, "POST", "/api/v1/blocklist", `{"type":"movie","tmdbId":11}`, admin); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("blocklist add = %d", resp.StatusCode)
	}
	if has(11) {
		t.Fatal("blocklisted title must disappear from suggestions immediately")
	}
	// and stays out when rows are regenerated
	if err := e.sugg.Generate(t.Context(), "aaaaaaaabbbbccccddddeeeeeeeeeeee"); err != nil {
		t.Fatal(err)
	}
	if has(11) {
		t.Fatal("blocklisted title came back after regeneration")
	}
	if _, b := call(t, e.app, "GET", "/api/v1/blocklist", "", admin); !strings.Contains(b, `"tmdbId":11`) {
		t.Fatalf("blocklist = %s", b)
	}
}

func getRows(t *testing.T, e *env, c *http.Cookie) []struct {
	Items []struct {
		TMDBID int `json:"tmdbId"`
	} `json:"items"`
} {
	t.Helper()
	_, body := call(t, e.app, "GET", "/api/v1/suggestions", "", c)
	var out struct {
		Rows []struct {
			Items []struct {
				TMDBID int `json:"tmdbId"`
			} `json:"items"`
		} `json:"rows"`
	}
	_ = json.Unmarshal([]byte(body), &out)
	return out.Rows
}

func TestUsersAdminAndProfile(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)
	if resp, _ := call(t, e.app, "GET", "/api/v1/admin/users", "", bob); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("user list = %d", resp.StatusCode)
	}
	_, body := call(t, e.app, "GET", "/api/v1/admin/users", "", admin)
	var users []struct {
		ID, Name, Role string
	}
	_ = json.Unmarshal([]byte(body), &users)
	if len(users) != 2 {
		t.Fatalf("users = %s", body)
	}
	var aliceID, bobID string
	for _, u := range users {
		if u.Name == "alice" {
			aliceID = u.ID
		} else {
			bobID = u.ID
		}
	}

	// import from Jellyfin: carol is new, alice and bob keep their roles
	if resp, _ := call(t, e.app, "POST", "/api/v1/admin/users/import", "", admin); resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("import without key = %d", resp.StatusCode)
	}
	call(t, e.app, "PUT", "/api/v1/admin/settings", `{"jellyfinApiKey":"jfkey"}`, admin)
	resp, body := call(t, e.app, "POST", "/api/v1/admin/users/import", "", admin)
	if resp.StatusCode != 200 || !strings.Contains(body, `"created":1`) || !strings.Contains(body, `"total":3`) {
		t.Fatalf("import = %d %s", resp.StatusCode, body)
	}
	if _, b := call(t, e.app, "POST", "/api/v1/admin/users/import", "", admin); !strings.Contains(b, `"created":0`) {
		t.Fatalf("second import = %s", b)
	}

	// roles: promote, never demote yourself
	if resp, _ := call(t, e.app, "PATCH", "/api/v1/admin/users/"+aliceID, `{"role":"user"}`, admin); resp.StatusCode != http.StatusConflict {
		t.Fatalf("self demotion = %d", resp.StatusCode)
	}
	resp, body = call(t, e.app, "PATCH", "/api/v1/admin/users/"+bobID, `{"role":"admin","region":"FR","language":"fr-FR"}`, admin)
	if resp.StatusCode != 200 || !strings.Contains(body, `"role":"admin"`) || !strings.Contains(body, `"region":"FR"`) {
		t.Fatalf("promote = %d %s", resp.StatusCode, body)
	}
	if resp, _ := call(t, e.app, "PATCH", "/api/v1/admin/users/"+bobID, `{"region":"france"}`, admin); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("bad region = %d", resp.StatusCode)
	}
	if resp, _ := call(t, e.app, "PATCH", "/api/v1/admin/users/nobody", `{"role":"user"}`, admin); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown user = %d", resp.StatusCode)
	}

	// own profile: region and language, validated
	resp, body = call(t, e.app, "PATCH", "/api/v1/me", `{"region":"GB","language":"en"}`, admin)
	if resp.StatusCode != 200 || !strings.Contains(body, `"region":"GB"`) {
		t.Fatalf("profile = %d %s", resp.StatusCode, body)
	}
	if _, b := call(t, e.app, "GET", "/api/v1/me", "", admin); !strings.Contains(b, `"language":"en"`) {
		t.Fatalf("/me = %s", b)
	}
	if resp, _ := call(t, e.app, "PATCH", "/api/v1/me", `{"language":"klingon!"}`, admin); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("bad language = %d", resp.StatusCode)
	}
	// a user cannot touch admin endpoints even after editing their profile
	if resp, _ := call(t, e.app, "PATCH", "/api/v1/admin/users/"+aliceID, `{"role":"user"}`, bob); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bob's session ended when he was promoted, got %d", resp.StatusCode)
	}
}

func TestLoginRateLimit(t *testing.T) {
	e := newEnv(t, true)
	jf := fakeJellyfin(t)
	call(t, e.app, "POST", "/api/v1/setup", `{"jellyfinUrl":"`+jf.URL+`"}`)

	for i := 0; i < 8; i++ {
		if resp, _ := call(t, e.app, "POST", "/api/v1/auth/login", `{"username":"mallory","password":"guess`+itoa(i)+`"}`); resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("attempt %d = %d", i, resp.StatusCode)
		}
	}
	// blocked now, even with the right password for a real account from the same address
	if resp, _ := call(t, e.app, "POST", "/api/v1/auth/login", `{"username":"mallory","password":"guess"}`); resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", resp.StatusCode)
	}
	if resp, _ := call(t, e.app, "POST", "/api/v1/auth/login", `{"username":"alice","password":"secret"}`); resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("the address is blocked too: %d", resp.StatusCode)
	}
}
