package server_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/store"
)

type refreshCounter struct{ n atomic.Int32 }

func (r *refreshCounter) QueueLibrary() { r.n.Add(1) }

// The "now available" webhook (and push) must not go out before the library shows the title, but must
// never be lost: after the hold it goes out anyway, and each alert goes out once.
func TestAvailableAlertWaitsForTheLibrary(t *testing.T) {
	t.Parallel()
	e := newEnv(t, false)
	_, admin, bob := setupUsers(t, e)
	radarr, sonarr := newFakeArr(t, "radarr"), newFakeArr(t, "sonarr")
	addInstance(t, e, admin, "radarr", radarr)
	addInstance(t, e, admin, "sonarr", sonarr)
	ctx := context.Background()

	refresh := &refreshCounter{}
	e.reqs.SetLibrary(refresh)
	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	e.reqs.SetClock(func() time.Time { return now })

	var mu sync.Mutex
	var titles []string
	recv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var body struct {
			Request struct {
				Title string `json:"title"`
				ID    string `json:"id"`
			} `json:"request"`
		}
		_ = json.Unmarshal(b, &body)
		mu.Lock()
		titles = append(titles, body.Request.ID)
		mu.Unlock()
	}))
	defer recv.Close()
	call(t, e.app, "POST", "/api/v1/admin/webhooks", `{"name":"w","url":"`+recv.URL+`","events":["media.available"],"enabled":true}`, admin)
	delivered := func() int { mu.Lock(); defer mu.Unlock(); return len(titles) }

	// movie: the file arrives, the request is available at once, the alert waits
	_, body := call(t, e.app, "POST", "/api/v1/requests", `{"type":"movie","tmdbId":5}`, bob)
	movie := decodeReq(t, body)
	call(t, e.app, "POST", "/api/v1/requests/"+movie.ID+"/approve", `{}`, admin)
	radarr.hasFile.Store(true)
	e.reqs.Poll(ctx)
	if _, b := call(t, e.app, "GET", "/api/v1/requests/"+movie.ID, "", bob); decodeReq(t, b).Status != "available" {
		t.Fatalf("movie not available: %s", b)
	}
	if refresh.n.Load() == 0 {
		t.Fatal("finished download did not ask for a library refresh")
	}
	now = now.Add(time.Minute)
	e.reqs.Poll(ctx)
	time.Sleep(100 * time.Millisecond)
	if n := delivered(); n != 0 {
		t.Fatalf("alert sent before the library had the title (%d)", n)
	}

	// the library sync sees the movie: the alert goes out, once
	if err := e.st.ReplaceLibrary(ctx, []store.LibraryItem{{MediaType: "movie", TMDBID: 5, JellyfinID: "j5", Title: "M"}}, nil); err != nil {
		t.Fatal(err)
	}
	e.reqs.Poll(ctx)
	waitFor(t, "alert after the library has the title", func() bool { return delivered() == 1 })
	e.reqs.Poll(ctx)
	time.Sleep(100 * time.Millisecond)
	if n := delivered(); n != 1 {
		t.Fatalf("alert sent %d times", n)
	}

	// show: the library never shows it, the alert still goes out after the hold
	_, body = call(t, e.app, "POST", "/api/v1/requests", `{"type":"tv","tmdbId":2,"seasons":[2]}`, admin)
	tv := decodeReq(t, body)
	sonarr.hasFile.Store(true)
	e.reqs.Poll(ctx)
	now = now.Add(9 * time.Minute)
	e.reqs.Poll(ctx)
	time.Sleep(100 * time.Millisecond)
	if n := delivered(); n != 1 {
		t.Fatalf("alert sent before the hold ran out (%d)", n)
	}
	now = now.Add(2 * time.Minute)
	e.reqs.Poll(ctx)
	waitFor(t, "alert after the hold", func() bool { return delivered() == 2 })
	_ = tv
}

// A season of a multi-season request is announced on its own, and the final alert is skipped when
// every season already was.
func TestSeasonAlerts(t *testing.T) {
	t.Parallel()
	e := newEnv(t, false)
	_, admin, _ := setupUsers(t, e)
	sonarr := newFakeArr(t, "sonarr")
	addInstance(t, e, admin, "sonarr", sonarr)
	ctx := context.Background()
	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	e.reqs.SetClock(func() time.Time { return now })
	e.reqs.SetAlertHold(10*time.Minute, 3*time.Minute)

	_, body := call(t, e.app, "POST", "/api/v1/requests", `{"type":"tv","tmdbId":2,"seasons":[1,2]}`, admin)
	tv := decodeReq(t, body)
	if tv.Status != "approved" {
		t.Fatalf("tv = %s", body)
	}
	if first, err := e.st.MarkSeasonDone(ctx, tv.ID, 1, now.Unix()); err != nil || !first {
		t.Fatalf("mark done = %v %v", first, err)
	}
	if again, _ := e.st.MarkSeasonDone(ctx, tv.ID, 1, now.Unix()); again {
		t.Fatal("a season was stamped complete twice")
	}
	notified := func(season int) int64 {
		rows, _ := e.st.RequestSeasonRows(ctx, tv.ID)
		for _, r := range rows {
			if r.SeasonNumber == season {
				return r.NotifiedAt
			}
		}
		return -1
	}
	e.reqs.Poll(ctx)
	if notified(1) != 0 {
		t.Fatal("season alert sent before its hold")
	}
	now = now.Add(4 * time.Minute)
	e.reqs.Poll(ctx)
	if notified(1) == 0 {
		t.Fatal("season alert not handled after its hold")
	}
	if notified(2) != 0 {
		t.Fatal("season 2 was alerted although it is not complete")
	}
}

// A pending request whose title shows up in the library anyway (added outside Tipsarr) turns available
// on the next poll, and the requester gets the "now available" alert once (issue #44).
func TestPendingRequestClosedWhenInLibrary(t *testing.T) {
	t.Parallel()
	e := newEnv(t, false)
	_, admin, bob := setupUsers(t, e)
	addInstance(t, e, admin, "radarr", newFakeArr(t, "radarr"))
	ctx := context.Background()

	var n atomic.Int32
	recv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { n.Add(1) }))
	defer recv.Close()
	call(t, e.app, "POST", "/api/v1/admin/webhooks", `{"name":"w","url":"`+recv.URL+`","events":["media.available"],"enabled":true}`, admin)

	_, body := call(t, e.app, "POST", "/api/v1/requests", `{"type":"movie","tmdbId":5}`, bob)
	movie := decodeReq(t, body)
	if movie.Status != "pending" {
		t.Fatalf("movie = %s", body)
	}
	e.reqs.Poll(ctx)
	if _, b := call(t, e.app, "GET", "/api/v1/requests/"+movie.ID, "", bob); decodeReq(t, b).Status != "pending" {
		t.Fatalf("closed before the library had it: %s", b)
	}

	if err := e.st.ReplaceLibrary(ctx, []store.LibraryItem{{MediaType: "movie", TMDBID: 5, JellyfinID: "j5", Title: "M"}}, nil); err != nil {
		t.Fatal(err)
	}
	e.reqs.Poll(ctx)
	if _, b := call(t, e.app, "GET", "/api/v1/requests/"+movie.ID, "", bob); decodeReq(t, b).Status != "available" {
		t.Fatalf("not available after the library had it: %s", b)
	}
	waitFor(t, "available alert", func() bool { return n.Load() == 1 })
	e.reqs.Poll(ctx)
	time.Sleep(100 * time.Millisecond)
	if got := n.Load(); got != 1 {
		t.Fatalf("alert sent %d times", got)
	}
}

// Show requests close only when the library holds every episode of every requested season, and a
// request sent to Sonarr is left to the Sonarr poll (issue #44).
func TestShowRequestClosedOnlyWhenSeasonsComplete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	setup := func(t *testing.T) (*env, *http.Cookie, *http.Cookie, func(int), func(string) string) {
		e := newEnv(t, false)
		_, admin, bob := setupUsers(t, e)
		addInstance(t, e, admin, "sonarr", newFakeArr(t, "sonarr"))
		library := func(episodes int) {
			t.Helper()
			if err := e.st.ReplaceLibrary(ctx, []store.LibraryItem{{MediaType: "tv", TMDBID: 2, JellyfinID: "j2", Title: "Show Two"}},
				[]store.LibrarySeason{{TMDBID: 2, SeasonNumber: 1, EpisodeCount: episodes}, {TMDBID: 2, SeasonNumber: 2, EpisodeCount: episodes}}); err != nil {
				t.Fatal(err)
			}
		}
		status := func(id string) string {
			r, err := e.st.GetRequest(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			return r.Status
		}
		return e, admin, bob, library, status
	}

	t.Run("in flight", func(t *testing.T) {
		e, admin, _, library, status := setup(t)
		_, body := call(t, e.app, "POST", "/api/v1/requests", `{"type":"tv","tmdbId":2,"seasons":[2]}`, admin)
		sent := decodeReq(t, body)
		if r, _ := e.st.GetRequest(ctx, sent.ID); r.Status != store.StatusApproved || r.SentAt == 0 {
			t.Fatalf("admin request = %+v", r)
		}
		library(1) // Sonarr imported E01 only
		e.reqs.Poll(ctx)
		library(10)
		e.reqs.Poll(ctx)
		if s := status(sent.ID); s != store.StatusApproved {
			t.Fatalf("in-flight request closed by the library instead of Sonarr: %s", s)
		}
	})

	t.Run("pending", func(t *testing.T) {
		e, _, bob, library, status := setup(t)
		_, body := call(t, e.app, "POST", "/api/v1/requests", `{"type":"tv","tmdbId":2,"seasons":[1]}`, bob)
		pending := decodeReq(t, body)
		library(1)
		e.reqs.Poll(ctx)
		if s := status(pending.ID); s != store.StatusPending {
			t.Fatalf("pending request closed on a partial season: %s", s)
		}
		library(10)
		e.reqs.Poll(ctx)
		if s := status(pending.ID); s != store.StatusAvailable {
			t.Fatalf("pending request not closed with complete seasons: %s", s)
		}
	})
}

// An approval that is still waiting for Radarr when the library closes the request does not overwrite
// the close: the request stays available and the requester is told once (issue #44).
func TestApproveLosesToLibraryClose(t *testing.T) {
	t.Parallel()
	e := newEnv(t, false)
	_, admin, bob := setupUsers(t, e)
	arr := newFakeArr(t, "radarr")
	addInstance(t, e, admin, "radarr", arr)
	ctx := context.Background()

	var n atomic.Int32
	recv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { n.Add(1) }))
	defer recv.Close()
	call(t, e.app, "POST", "/api/v1/admin/webhooks", `{"name":"w","url":"`+recv.URL+`","events":["media.available"],"enabled":true}`, admin)

	_, body := call(t, e.app, "POST", "/api/v1/requests", `{"type":"movie","tmdbId":5}`, bob)
	req := decodeReq(t, body)
	e.reqs.Poll(ctx) // the library generation is now seen
	arr.beforeAdd.Store(func() {
		if err := e.st.ReplaceLibrary(ctx, []store.LibraryItem{{MediaType: "movie", TMDBID: 5, JellyfinID: "j5", Title: "M"}}, nil); err != nil {
			t.Error(err)
		}
		e.reqs.Poll(ctx)
	})
	if resp, b := call(t, e.app, "POST", "/api/v1/requests/"+req.ID+"/approve", `{}`, admin); resp.StatusCode != http.StatusConflict {
		t.Fatalf("approve = %d %s", resp.StatusCode, b)
	}
	r, err := e.st.GetRequest(ctx, req.ID)
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != store.StatusAvailable || r.NotifiedAt == 0 {
		t.Fatalf("request after the race = %+v", r)
	}
	waitFor(t, "available alert", func() bool { return n.Load() == 1 })
	arr.hasFile.Store(true)
	e.reqs.Poll(ctx)
	time.Sleep(100 * time.Millisecond)
	if got := n.Load(); got != 1 {
		t.Fatalf("alert sent %d times", got)
	}
}

// In dry-run an approved request is never sent, so the library closes it like a pending one.
func TestDryRunApprovedClosedWhenInLibrary(t *testing.T) {
	t.Parallel()
	e := newEnv(t, true)
	_, admin, _ := setupUsers(t, e)
	addInstance(t, e, admin, "radarr", newFakeArr(t, "radarr"))
	ctx := context.Background()

	_, body := call(t, e.app, "POST", "/api/v1/requests", `{"type":"movie","tmdbId":5}`, admin)
	req := decodeReq(t, body)
	if r, _ := e.st.GetRequest(ctx, req.ID); r.Status != store.StatusApproved || r.SentAt != 0 {
		t.Fatalf("dry-run admin request = %+v", r)
	}
	if err := e.st.ReplaceLibrary(ctx, []store.LibraryItem{{MediaType: "movie", TMDBID: 5, JellyfinID: "j5", Title: "M"}}, nil); err != nil {
		t.Fatal(err)
	}
	e.reqs.Poll(ctx)
	if r, _ := e.st.GetRequest(ctx, req.ID); r.Status != store.StatusAvailable {
		t.Fatalf("dry-run approved request not closed: %+v", r)
	}
}

// A season still airing (issue #50): Sonarr's episodeCount only counts the aired episodes, so the
// request stays open with the real count until the whole season is on disk, undated episodes included,
// and the "now available" alert waits until the library holds the last episode.
func TestAiringSeasonStaysOpen(t *testing.T) {
	t.Parallel()
	e := newEnv(t, false)
	_, admin, _ := setupUsers(t, e)
	sonarr := newFakeArr(t, "sonarr")
	addInstance(t, e, admin, "sonarr", sonarr)
	ctx := context.Background()

	var alerts atomic.Int32
	recv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { alerts.Add(1) }))
	defer recv.Close()
	call(t, e.app, "POST", "/api/v1/admin/webhooks", `{"name":"w","url":"`+recv.URL+`","events":["media.available"],"enabled":true}`, admin)

	series := func(files, count int, nextAiring string) string {
		na := ""
		if nextAiring != "" {
			na = `,"nextAiring":"` + nextAiring + `"`
		}
		return fmt.Sprintf(`{"id":77,"tvdbId":81189,"monitored":true,"statistics":{"episodeFileCount":%d,"episodeCount":%d},`+
			`"seasons":[{"seasonNumber":1,"monitored":true,"statistics":{"episodeFileCount":%d,"episodeCount":%d,"totalEpisodeCount":8%s}}]}`,
			files, count, files, count, na)
	}
	episodes := func(withFile int, undatedMonitored bool) string {
		var eps []string
		for i := 1; i <= 8; i++ {
			switch {
			case i <= withFile:
				eps = append(eps, fmt.Sprintf(`{"id":%d,"seasonNumber":1,"episodeNumber":%d,"airDate":"2026-11-0%d","monitored":true,"hasFile":true}`, i, i, i))
			default:
				eps = append(eps, fmt.Sprintf(`{"id":%d,"seasonNumber":1,"episodeNumber":%d,"monitored":%t,"hasFile":false}`, i, i, undatedMonitored))
			}
		}
		return "[" + strings.Join(eps, ",") + "]"
	}
	library := func(n int) {
		t.Helper()
		if err := e.st.ReplaceLibrary(ctx, []store.LibraryItem{{MediaType: "tv", TMDBID: 2, JellyfinID: "j2", Title: "Show Two"}},
			[]store.LibrarySeason{{TMDBID: 2, SeasonNumber: 1, EpisodeCount: n}}); err != nil {
			t.Fatal(err)
		}
	}
	type seasonProgress struct {
		Season, Percent, Episodes, Total int
		Airing                           bool
	}
	view := func(id string) (string, string, []seasonProgress) {
		t.Helper()
		_, b := call(t, e.app, "GET", "/api/v1/requests/"+id, "", admin)
		var v struct {
			Status, Stage  string
			SeasonProgress []seasonProgress
		}
		if err := json.Unmarshal([]byte(b), &v); err != nil {
			t.Fatal(err)
		}
		return v.Status, v.Stage, v.SeasonProgress
	}

	sonarr.series77.Store(series(3, 3, "2026-11-27T05:00:00Z"))
	_, body := call(t, e.app, "POST", "/api/v1/requests", `{"type":"tv","tmdbId":2,"seasons":[1]}`, admin)
	req := decodeReq(t, body)
	if req.Status != "approved" {
		t.Fatalf("request = %s", body)
	}
	// Sonarr tracks it now: the detail sizes season 1 with its 8 episodes
	sonarr.seriesJSON.Store(`[{"id":77,"tvdbId":81189,"monitored":true,"seasons":[{"seasonNumber":1,"monitored":true,"statistics":{"totalEpisodeCount":8}}]}]`)
	library(3)

	// 3 aired, 3 on disk, 5 to air
	e.reqs.Poll(ctx)
	if st, stage, sp := view(req.ID); st != "approved" || stage != "searching" || len(sp) != 1 || sp[0] != (seasonProgress{1, 37, 3, 8, true}) {
		t.Fatalf("airing season = %s %s %+v", st, stage, sp)
	}

	// the 5 others have no air date yet: Sonarr's statistics forget them, the episode list does not
	sonarr.series77.Store(series(3, 3, ""))
	sonarr.episodeJSON.Store(episodes(3, true))
	e.reqs.Poll(ctx)
	if st, _, sp := view(req.ID); st != "approved" || len(sp) != 1 || sp[0] != (seasonProgress{1, 37, 3, 8, true}) {
		t.Fatalf("season with undated episodes = %s %+v", st, sp)
	}

	// every episode on disk: available, but the alert waits until the library has the last one
	sonarr.series77.Store(series(8, 8, ""))
	sonarr.episodeJSON.Store(episodes(8, true))
	e.reqs.Poll(ctx)
	if st, _, _ := view(req.ID); st != "available" {
		t.Fatalf("finished season not available: %s", st)
	}
	e.reqs.Poll(ctx)
	time.Sleep(100 * time.Millisecond)
	if n := alerts.Load(); n != 0 {
		t.Fatalf("alert sent while the library had 3 of 8 episodes (%d)", n)
	}
	library(8)
	e.reqs.Poll(ctx)
	waitFor(t, "alert once the library has the season", func() bool { return alerts.Load() == 1 })
}
