package server_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

type teaserFeed struct {
	Items []struct {
		Type        string `json:"type"`
		TMDBID      int    `json:"tmdbId"`
		TrailerKey  string `json:"trailerKey"`
		Player      string `json:"player"`
		VideoURL    string `json:"videoUrl"`
		VideoHeight int    `json:"videoHeight"`
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
	f := newFakeResolver(t, e.files.Dir(), false)
	enableFiles(t, e, admin, f)

	// alice has watched movie 1 and show 2, so her feed is the recommended titles that have a trailer (10 and 11)
	call(t, e.app, "PUT", "/api/v1/admin/settings", `{"jellyfinApiKey":"jfkey"}`, admin)
	call(t, e.app, "POST", "/api/v1/admin/sync/history-sync", "", admin)
	waitFor(t, "history", func() bool {
		v, _ := e.st.HistoryVersion(t.Context(), "aaaaaaaabbbbccccddddeeeeeeeeeeee")
		return v == 1
	})

	feed := getTeasers(t, e, "", admin)
	if len(feed.Items) != 2 || feed.Done {
		t.Fatalf("first feed = %+v", feed)
	}
	for _, it := range feed.Items {
		if it.Player != "youtube" || it.VideoURL != "" || it.TrailerKey == "" {
			t.Fatalf("nothing is downloaded yet: %+v", it)
		}
	}
	// the resolver was asked for every card of the batch
	asked := map[string]bool{}
	for _, k := range f.resolvedKeys() {
		asked[k] = true
	}
	for _, it := range feed.Items {
		if !asked[it.TrailerKey] {
			t.Fatalf("the resolver was not asked for %s (asked: %v)", it.TrailerKey, f.resolvedKeys())
		}
	}

	// once a file is ready its card plays natively and comes first within the batch
	last := feed.Items[len(feed.Items)-1]
	f.finish(t, last.TrailerKey)
	if _, err := e.files.Sync(t.Context()); err != nil {
		t.Fatal(err)
	}
	feed = getTeasers(t, e, "", admin)
	first := feed.Items[0]
	if first.TrailerKey != last.TrailerKey || first.Player != "file" || first.VideoURL != "/api/v1/teasers/video/"+last.TrailerKey || first.VideoHeight != 720 {
		t.Fatalf("ready card first = %+v (last was %s)", first, last.TrailerKey)
	}
	for _, it := range feed.Items[1:] {
		if it.Player != "youtube" {
			t.Fatalf("a card that is not ready claims a file: %+v", it)
		}
	}
	// the file really plays through the video endpoint
	if resp, b := rangeGet(t, e, first.VideoURL, "bytes=0-9", admin); resp.StatusCode != 206 || len(b) != 10 {
		t.Fatalf("video = %d %d bytes", resp.StatusCode, len(b))
	}

	// bob sees the same two titles. Titles the app already holds are not sent again
	if got := getTeasers(t, e, "", bob); len(got.Items) != 2 {
		t.Fatalf("bob's feed = %+v", got)
	}
	if got := getTeasers(t, e, "?exclude=movie:10,movie:11", bob); len(got.Items) != 0 || !got.Done {
		t.Fatalf("excluded feed = %+v", got)
	}
	// swiping past a title removes it for good, for that person only
	if resp, body := call(t, e.app, "POST", "/api/v1/teasers/seen", `{"type":"movie","tmdbId":10}`, bob); resp.StatusCode != 204 {
		t.Fatalf("seen = %d %s", resp.StatusCode, body)
	}
	got := getTeasers(t, e, "", bob)
	if len(got.Items) != 1 || got.Items[0].TMDBID != 11 {
		t.Fatalf("bob after swiping past movie 10 = %+v", got)
	}
	if got := getTeasers(t, e, "", admin); len(got.Items) != 2 {
		t.Fatalf("alice's feed changed when bob swiped: %+v", got)
	}

	// not signed in
	if resp, _ := call(t, e.app, "GET", "/api/v1/teasers", ""); resp.StatusCode != 401 {
		t.Fatalf("anonymous = %d", resp.StatusCode)
	}
}

// With the resolver down the feed keeps working with the YouTube player.
func TestTeasersFeedSurvivesADeadResolver(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)
	f := newFakeResolver(t, e.files.Dir(), true)
	enableFiles(t, e, admin, f)
	f.srv.Close()
	feed := getTeasers(t, e, "", bob)
	if len(feed.Items) == 0 {
		t.Fatal("no cards while the resolver is down")
	}
	for _, it := range feed.Items {
		if it.Player != "youtube" || it.TrailerKey == "" {
			t.Fatalf("card = %+v", it)
		}
	}
}
