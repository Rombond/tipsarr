package server_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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
