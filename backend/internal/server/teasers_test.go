package server_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

type teaserFeed struct {
	Items []struct {
		Type       string `json:"type"`
		TMDBID     int    `json:"tmdbId"`
		TrailerKey string `json:"trailerKey"`
	} `json:"items"`
	Done bool `json:"done"`
}

func getTeasers(t *testing.T, e *env, query string, c *http.Cookie) teaserFeed {
	t.Helper()
	resp, body := call(t, e.app, "GET", "/api/v1/teasers"+query, "", c)
	if resp.StatusCode != 200 {
		t.Fatalf("teasers = %d %s", resp.StatusCode, body)
	}
	var f teaserFeed
	if err := json.Unmarshal([]byte(body), &f); err != nil {
		t.Fatalf("decode %s: %v", body, err)
	}
	return f
}

func TestTeasersFeed(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)

	// only titles with a trailer: movie 1 has one, show 2 and the recommendations have none
	f := getTeasers(t, e, "", bob)
	if len(f.Items) != 1 || f.Items[0].Type != "movie" || f.Items[0].TMDBID != 1 || f.Items[0].TrailerKey != "OFFICIAL1" || f.Done {
		t.Fatalf("first feed = %+v", f)
	}

	// a title the app already holds is not sent again
	if f = getTeasers(t, e, "?exclude=movie:1", bob); len(f.Items) != 0 || !f.Done {
		t.Fatalf("excluded feed = %+v", f)
	}

	// swiping past it removes it for good, for that person only
	if resp, body := call(t, e.app, "POST", "/api/v1/teasers/seen", `{"type":"movie","tmdbId":1}`, bob); resp.StatusCode != 204 {
		t.Fatalf("seen = %d %s", resp.StatusCode, body)
	}
	if f = getTeasers(t, e, "", bob); len(f.Items) != 0 || !f.Done {
		t.Fatalf("feed after seen = %+v", f)
	}
	if f = getTeasers(t, e, "", admin); len(f.Items) != 1 {
		t.Fatalf("other person's feed = %+v", f)
	}

	// not signed in
	if resp, _ := call(t, e.app, "GET", "/api/v1/teasers", ""); resp.StatusCode != 401 {
		t.Fatalf("anonymous = %d", resp.StatusCode)
	}
}
