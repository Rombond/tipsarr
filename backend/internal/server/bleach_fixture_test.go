package server_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// Bleach as Sonarr (TVDB 74796) numbers it: 17 arc seasons plus specials, where TMDB (30984) has S1 (366 episodes)
// and S2 "Thousand-Year Blood War" (50 episodes, first aired 2022-10-11 like Sonarr's S17).
var bleachCounts = []int{7, 20, 21, 22, 28, 18, 22, 20, 16, 22, 16, 7, 17, 36, 51, 26, 24, 50} // index = season

func bleachFirstAired(n int) string {
	switch n {
	case 1:
		return "2004-10-05"
	case 17:
		return "2022-10-11"
	case 0:
		return "2006-12-16"
	}
	return fmt.Sprintf("%d-01-10", 2004+n/2) // anything not matching a TMDB season
}

// fakeBleachSonarr is a Sonarr knowing only Bleach; tracked says whether it already has the series (id 86).
type fakeBleachSonarr struct {
	srv       *httptest.Server
	tracked   atomic.Bool
	down      atomic.Bool
	monitored atomic.Bool // series monitored + S1 with missing files (for the import)
	mu        sync.Mutex
	calls     []string
	put       map[string]any
	commands  []map[string]any
	added     map[string]any
}

func (f *fakeBleachSonarr) seasonsJSON(withStats bool) string {
	var parts []string
	for n, c := range bleachCounts {
		mon := f.monitored.Load() && n == 1
		if withStats {
			ec := 0
			if mon {
				ec = c
			}
			parts = append(parts, fmt.Sprintf(`{"seasonNumber":%d,"monitored":%t,"statistics":{"episodeFileCount":0,"episodeCount":%d,"totalEpisodeCount":%d}}`, n, mon, ec, c))
		} else {
			parts = append(parts, fmt.Sprintf(`{"seasonNumber":%d,"monitored":false}`, n))
		}
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func (f *fakeBleachSonarr) seriesJSON() string {
	return fmt.Sprintf(`{"id":86,"title":"Bleach","tvdbId":74796,"tmdbId":30984,"titleSlug":"bleach","monitored":%t,"seasons":%s}`, f.monitored.Load(), f.seasonsJSON(true))
}

func newFakeBleachSonarr(t *testing.T) *fakeBleachSonarr {
	f := &fakeBleachSonarr{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/api/v3")
		f.mu.Lock()
		f.calls = append(f.calls, r.Method+" "+p)
		f.mu.Unlock()
		if f.down.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		out := func(s string) { _, _ = w.Write([]byte(s)) }
		decode := func() map[string]any {
			var b map[string]any
			_ = json.NewDecoder(r.Body).Decode(&b)
			return b
		}
		switch {
		case p == "/system/status":
			out(`{"appName":"Sonarr","version":"4.0.20"}`)
		case p == "/qualityprofile":
			out(`[{"id":4,"name":"HD-1080p","items":[{"quality":{"name":"Bluray-1080p","resolution":1080},"allowed":true}]}]`)
		case p == "/rootfolder":
			out(`[{"id":1,"path":"/data/media","freeSpace":123}]`)
		case p == "/queue":
			out(`{"records":[],"totalRecords":0}`)
		case p == "/languageprofile":
			w.WriteHeader(http.StatusNotFound)
		case r.Method == http.MethodPost && p == "/command":
			b := decode()
			f.mu.Lock()
			f.commands = append(f.commands, b)
			f.mu.Unlock()
			w.WriteHeader(http.StatusCreated)
			out(`{"id":1}`)
		case r.Method == http.MethodPost && p == "/series":
			b := decode()
			f.mu.Lock()
			f.added = b
			f.mu.Unlock()
			w.WriteHeader(http.StatusCreated)
			out(`{"id":86,"tvdbId":74796}`)
		case r.Method == http.MethodPut && p == "/series/86":
			b := decode()
			f.mu.Lock()
			f.put = b
			f.mu.Unlock()
			out(f.seriesJSON())
		case p == "/series/86":
			out(f.seriesJSON())
		case p == "/series/lookup":
			if r.URL.Query().Get("term") != "tvdb:74796" {
				out(`[]`)
				return
			}
			id := "null"
			if f.tracked.Load() {
				id = "86"
			}
			out(`[{"title":"Bleach","tvdbId":74796,"id":` + id + `,"statistics":{"seasonCount":17,"totalEpisodeCount":0},"seasons":` + f.seasonsJSON(false) + `}]`)
		case p == "/series":
			if !f.tracked.Load() || (r.URL.Query().Get("tvdbId") != "" && r.URL.Query().Get("tvdbId") != "74796") {
				out(`[]`)
				return
			}
			out("[" + f.seriesJSON() + "]")
		case p == "/episode":
			if r.URL.Query().Get("seriesId") != "86" || !f.tracked.Load() {
				out(`[]`)
				return
			}
			only := -1
			if v := r.URL.Query().Get("seasonNumber"); v != "" {
				only, _ = strconv.Atoi(v)
			}
			var eps []string
			for n := range bleachCounts {
				if only >= 0 && n != only {
					continue
				}
				for e := 1; e <= 2; e++ {
					eps = append(eps, fmt.Sprintf(`{"id":%d,"seasonNumber":%d,"episodeNumber":%d,"title":"Arc %d episode %d","overview":"o","airDate":"%s","runtime":24,"hasFile":%t,"monitored":false}`,
						n*100+e, n, e, n, e, bleachFirstAired(n), e == 1))
				}
			}
			out("[" + strings.Join(eps, ",") + "]")
		default:
			out(`{}`)
		}
	}))
	t.Cleanup(f.srv.Close)
	return f
}

// writes lists every call that is not a read.
func (f *fakeBleachSonarr) writes() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var w []string
	for _, c := range f.calls {
		if !strings.HasPrefix(c, "GET ") {
			w = append(w, c)
		}
	}
	return w
}

func addBleachSonarr(t *testing.T, e *env, admin *http.Cookie, f *fakeBleachSonarr) {
	t.Helper()
	resp, body := call(t, e.app, "POST", "/api/v1/admin/servarr",
		`{"kind":"sonarr","name":"sonarr","url":"`+f.srv.URL+`","apiKey":"rk","qualityProfileId":4,"rootFolder":"/data/media","isDefault":true}`, admin)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create instance = %d %s", resp.StatusCode, body)
	}
}
