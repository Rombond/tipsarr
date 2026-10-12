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

	// nothing is ready and no pool exists yet: the feed is empty but not finished, and the pool builds in the background
	if feed := getTeasers(t, e, "", bob); len(feed.Items) != 0 || feed.Done {
		t.Fatalf("cold feed = %+v", feed)
	}
	waitFor(t, "pool", func() bool {
		rows, _ := e.st.TeaserPool(t.Context(), "|")
		return len(rows) > 0
	})
	// the pool holds upcoming movie 7 (a trailer); movie 1 is in the library and show 2 has no trailer, so both are left out
	rows, _ := e.st.TeaserPool(t.Context(), "|")
	if len(rows) != 1 || rows[0].MediaType != "movie" || rows[0].TMDBID != 7 || rows[0].YoutubeKey != "UPCOMING007" {
		t.Fatalf("pool = %+v", rows)
	}
	waitFor(t, "resolver asked for the pool", func() bool {
		for _, k := range f.resolvedKeys() {
			if k == "UPCOMING007" {
				return true
			}
		}
		return false
	})

	// still downloading: not shown, not finished
	if feed := getTeasers(t, e, "", bob); len(feed.Items) != 0 || feed.Done {
		t.Fatalf("feed while downloading = %+v", feed)
	}
	f.finish(t, "UPCOMING007")
	if _, err := e.files.Sync(t.Context()); err != nil {
		t.Fatal(err)
	}
	feed := getTeasers(t, e, "", bob)
	if len(feed.Items) != 1 || feed.Done {
		t.Fatalf("bob's feed = %+v", feed)
	}
	it := feed.Items[0]
	if it.TMDBID != 7 || it.TrailerKey != "UPCOMING007" || it.VideoURL != "/api/v1/teasers/video/UPCOMING007" || it.VideoHeight != 720 {
		t.Fatalf("card = %+v", it)
	}
	// the file really plays through the video endpoint
	if resp, b := rangeGet(t, e, it.VideoURL, "bytes=0-9", bob); resp.StatusCode != 206 || len(b) != 10 {
		t.Fatalf("video = %d %d bytes", resp.StatusCode, len(b))
	}

	// alice gets the pool title too; her recommendations (10, 11) are asked for and wait for their files
	if feed := getTeasers(t, e, "", admin); len(feed.Items) != 1 || feed.Done {
		t.Fatalf("alice before her files = %+v", feed)
	}
	asked := map[string]bool{}
	for _, k := range f.resolvedKeys() {
		asked[k] = true
	}
	if !asked["RECTRAILER0"] || !asked["RECTRAILER1"] {
		t.Fatalf("recommendation trailers were not asked for: %v", f.resolvedKeys())
	}
	f.finish(t, "RECTRAILER0")
	f.finish(t, "RECTRAILER1")
	if _, err := e.files.Sync(t.Context()); err != nil {
		t.Fatal(err)
	}
	if got := getTeasers(t, e, "", admin); len(got.Items) != 3 {
		t.Fatalf("alice's feed = %+v", got)
	}

	// titles the app already holds are not sent again, and when the person has nothing left the feed is done
	if got := getTeasers(t, e, "?exclude=movie:7,movie:10,movie:11", bob); len(got.Items) != 0 || !got.Done {
		t.Fatalf("excluded feed = %+v", got)
	}
	// swiping past a title removes it for good, for that person only
	if resp, body := call(t, e.app, "POST", "/api/v1/teasers/seen", `{"type":"movie","tmdbId":7}`, bob); resp.StatusCode != 204 {
		t.Fatalf("seen = %d %s", resp.StatusCode, body)
	}
	if got := getTeasers(t, e, "", bob); len(got.Items) != 2 || got.Done {
		t.Fatalf("bob after swiping past movie 7 = %+v", got)
	}
	if got := getTeasers(t, e, "", admin); len(got.Items) != 3 {
		t.Fatalf("alice's feed changed when bob swiped: %+v", got)
	}

	// not signed in
	if resp, _ := call(t, e.app, "GET", "/api/v1/teasers", ""); resp.StatusCode != 401 {
		t.Fatalf("anonymous = %d", resp.StatusCode)
	}
}

// The pool job keeps the files of the pool warm and asks again for a file the resolver lost.
func TestTeaserPoolJob(t *testing.T) {
	e := newEnv(t, true)
	_, admin, _ := setupUsers(t, e)
	f := newFakeResolver(t, e.files.Dir(), true)

	// off: skipped
	if _, err := e.teasers.RefreshPool(t.Context()); err == nil {
		t.Fatal("the pool job ran with trailer files off")
	}
	enableFiles(t, e, admin, f)
	if _, err := e.teasers.RefreshPool(t.Context()); err != nil {
		t.Fatal(err)
	}
	// movie 1 (trending) and movie 7 (upcoming); show 2 has no trailer
	if rows, _ := e.st.TeaserPool(t.Context(), "|"); len(rows) != 2 {
		t.Fatalf("pool = %+v", rows)
	}
	if got := f.touchedKeys(); len(got) == 0 {
		t.Fatal("the pool files were not kept warm")
	}
}

// With the resolver down the feed answers, with nothing to play yet.
func TestTeasersFeedSurvivesADeadResolver(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)
	f := newFakeResolver(t, e.files.Dir(), true)
	enableFiles(t, e, admin, f)
	f.srv.Close()
	if feed := getTeasers(t, e, "", bob); len(feed.Items) != 0 || feed.Done {
		t.Fatalf("feed = %+v", feed)
	}
}
