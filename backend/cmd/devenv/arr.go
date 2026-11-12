package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Radarr and Sonarr stand-ins. A title added to them "downloads" on a clock: it waits a few seconds, then a
// movie downloads for ~80 s (a show season by season, ~30 s each), then the files are there.
const (
	arrKey       = "devkey"
	arrWait      = 8 * time.Second
	movieDownSec = 80.0
	seasonSec    = 30.0
)

type arrEntry struct {
	id      int
	tmdb    int
	tvdb    int
	title   string
	added   time.Time
	seasons []int // monitored seasons (shows)
}

type arr struct {
	kind  string
	mu    sync.Mutex
	items map[int]*arrEntry
	next  int
}

func clamp(f float64) float64 { return math.Max(0, math.Min(1, f)) }

func (e *arrEntry) since() float64 { return time.Since(e.added).Seconds() }

func hhmmss(sec float64) string {
	s := int(sec)
	return fmt.Sprintf("%02d:%02d:%02d", s/3600, s%3600/60, s%60)
}

func (a *arr) movieJSON(e *arrEntry) obj {
	done := e.since() >= arrWait.Seconds()+movieDownSec
	return obj{"id": e.id, "title": e.title, "tmdbId": e.tmdb, "hasFile": done, "monitored": true, "year": year(e.tmdb)}
}

// season progress 0..1 for the k-th monitored season
func (e *arrEntry) frac(k int) float64 {
	return clamp((e.since() - arrWait.Seconds() - float64(k)*seasonSec) / seasonSec)
}

// files is how many episodes of season s are downloaded (0 when the season is not monitored)
func (e *arrEntry) files(s int) int {
	k := sort.SearchInts(e.seasons, s)
	if k >= len(e.seasons) || e.seasons[k] != s {
		return 0
	}
	ec := tvdbEpisodeCount(e.tmdb, s)
	f := e.frac(k)
	if f >= 1 {
		return ec
	}
	return int(f * float64(ec))
}

func (a *arr) seriesJSON(e *arrEntry) obj {
	monitored := map[int]bool{}
	for _, s := range e.seasons {
		monitored[s] = true
	}
	seasons := []obj{}
	totalEps, totalFiles := 0, 0
	for s := 0; s <= tvdbSeasonCount(e.tmdb); s++ {
		total := tvdbEpisodeCount(e.tmdb, s)
		ec, files := 0, e.files(s)
		if monitored[s] {
			ec = total
			totalEps, totalFiles = totalEps+ec, totalFiles+files
		}
		seasons = append(seasons, obj{"seasonNumber": s, "monitored": monitored[s], "statistics": obj{"episodeCount": ec, "episodeFileCount": files, "totalEpisodeCount": total}})
	}
	return obj{"id": e.id, "title": e.title, "tvdbId": e.tvdb, "tmdbId": e.tmdb, "monitored": true, "year": year(e.tmdb), "seasons": seasons,
		"statistics": obj{"episodeCount": totalEps, "episodeFileCount": totalFiles}}
}

func (a *arr) queue() []obj {
	out := []obj{}
	for _, e := range a.items {
		if a.kind == "radarr" {
			t := e.since() - arrWait.Seconds()
			if t > 0 && t < movieDownSec {
				f := t / movieDownSec
				out = append(out, obj{"movieId": e.id, "title": e.title, "size": 1000.0, "sizeleft": 1000 * (1 - f), "timeleft": hhmmss(movieDownSec - t), "status": "downloading"})
			}
			continue
		}
		for k, s := range e.seasons {
			f := e.frac(k)
			if f <= 0 || f >= 1 {
				continue
			}
			ec := tvdbEpisodeCount(e.tmdb, s)
			cur := int(f * float64(ec))
			for ep := cur; ep < ec; ep++ {
				rec := obj{"seriesId": e.id, "title": e.title, "size": 1000.0, "sizeleft": 1000.0, "timeleft": hhmmss((1 - f) * seasonSec), "status": "queued", "seasonNumber": s}
				if ep == cur {
					rec["status"], rec["sizeleft"] = "downloading", 1000*(1-(f*float64(ec)-float64(cur)))
				}
				out = append(out, rec)
			}
		}
	}
	return out
}

func newArr(kind string) http.Handler {
	a := &arr{kind: kind, items: map[int]*arrEntry{}, next: 100}
	mux := http.NewServeMux()
	h := func(pattern string, fn http.HandlerFunc) {
		mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Api-Key") != arrKey {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			a.mu.Lock()
			defer a.mu.Unlock()
			fn(w, r)
		})
	}
	h("GET /api/v3/system/status", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, obj{"appName": strings.ToUpper(kind[:1]) + kind[1:], "version": "5.0"})
	})
	h("GET /api/v3/qualityprofile", func(w http.ResponseWriter, _ *http.Request) {
		quality := func(name string, res int) []obj {
			return []obj{{"quality": obj{"name": name, "resolution": res}, "items": []obj{}, "allowed": true}}
		}
		writeJSON(w, []obj{{"id": 4, "name": "HD-1080p", "items": quality("Bluray-1080p", 1080)}, {"id": 6, "name": "UHD-2160p", "items": quality("Bluray-2160p", 2160)}})
	})
	h("GET /api/v3/rootfolder", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, []obj{{"id": 1, "path": "/data/media/" + map[string]string{"radarr": "movies", "sonarr": "tv"}[kind], "freeSpace": 812345678901}, {"id": 2, "path": "/data/media/other", "freeSpace": 99345678901}})
	})
	h("GET /api/v3/tag", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, []obj{}) })
	h("GET /api/v3/languageprofile", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) })
	h("GET /api/v3/queue", func(w http.ResponseWriter, _ *http.Request) {
		q := a.queue()
		writeJSON(w, obj{"records": q, "totalRecords": len(q)})
	})
	list := func(w http.ResponseWriter, r *http.Request) {
		out := []obj{}
		ids := []int{}
		for id := range a.items {
			ids = append(ids, id)
		}
		sort.Ints(ids)
		for _, id := range ids {
			e := a.items[id]
			if t := r.URL.Query().Get("tmdbId"); t != "" && strconv.Itoa(e.tmdb) != t {
				continue
			}
			if t := r.URL.Query().Get("tvdbId"); t != "" && strconv.Itoa(e.tvdb) != t {
				continue
			}
			if kind == "radarr" {
				out = append(out, a.movieJSON(e))
			} else {
				out = append(out, a.seriesJSON(e))
			}
		}
		writeJSON(w, out)
	}
	one := func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.PathValue("id"))
		e, ok := a.items[id]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Method == http.MethodDelete {
			delete(a.items, id)
			w.WriteHeader(http.StatusOK)
			return
		}
		if kind == "radarr" {
			writeJSON(w, a.movieJSON(e))
		} else {
			writeJSON(w, a.seriesJSON(e))
		}
	}
	if kind == "radarr" {
		h("GET /api/v3/movie", list)
		h("GET /api/v3/movie/{id}", one)
		h("DELETE /api/v3/movie/{id}", one)
		h("GET /api/v3/movie/lookup/tmdb", func(w http.ResponseWriter, r *http.Request) {
			id, _ := strconv.Atoi(r.URL.Query().Get("tmdbId"))
			if !isMovie(id) {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			writeJSON(w, obj{"title": title(id, "en"), "tmdbId": id, "year": year(id), "imdbId": imdbID(id), "ratings": obj{
				"imdb":           obj{"votes": 1000 + id*37, "value": vote(id) + 0.3, "type": "user"},
				"tmdb":           obj{"votes": 500 + id, "value": vote(id), "type": "user"},
				"metacritic":     obj{"votes": 0, "value": float64(40 + id%50), "type": "user"},
				"rottenTomatoes": obj{"votes": 0, "value": float64(30 + id%65), "type": "user"}}})
		})
		h("POST /api/v3/movie", func(w http.ResponseWriter, r *http.Request) {
			var b struct {
				TMDBID int    `json:"tmdbId"`
				Title  string `json:"title"`
			}
			_ = json.NewDecoder(r.Body).Decode(&b)
			if b.TMDBID == 50 { // the one title Radarr refuses, to see a failed request
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("dev fake: Radarr refuses this movie"))
				return
			}
			a.next++
			e := &arrEntry{id: a.next, tmdb: b.TMDBID, title: b.Title, added: time.Now()}
			a.items[e.id] = e
			w.WriteHeader(http.StatusCreated)
			writeJSON(w, obj{"id": e.id, "tmdbId": e.tmdb})
		})
		return mux
	}
	h("GET /api/v3/series", list)
	h("GET /api/v3/series/{id}", one)
	h("DELETE /api/v3/series/{id}", one)
	h("GET /api/v3/series/lookup", func(w http.ResponseWriter, r *http.Request) {
		tvdb, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Query().Get("term"), "tvdb:"))
		tmdb := tvdb - 90000
		if !isShow(tmdb) {
			writeJSON(w, []obj{})
			return
		}
		seasons := []obj{}
		for s := 0; s <= tvdbSeasonCount(tmdb); s++ {
			seasons = append(seasons, obj{"seasonNumber": s, "monitored": false})
		}
		writeJSON(w, []obj{{"title": title(tmdb, "en"), "tvdbId": tvdb, "tmdbId": tmdb, "year": year(tmdb), "seasons": seasons, "seriesType": "standard"}})
	})
	h("GET /api/v3/episode", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.URL.Query().Get("seriesId"))
		e, ok := a.items[id]
		if !ok {
			writeJSON(w, []obj{})
			return
		}
		only, filtered := 0, r.URL.Query().Get("seasonNumber") != ""
		if filtered {
			only, _ = strconv.Atoi(r.URL.Query().Get("seasonNumber"))
		}
		out := []obj{}
		for s := 0; s <= tvdbSeasonCount(e.tmdb); s++ {
			if filtered && s != only {
				continue
			}
			k := sort.SearchInts(e.seasons, s)
			monitored := k < len(e.seasons) && e.seasons[k] == s
			files, first := e.files(s), tvdbFirstAired(e.tmdb, s)
			for ep := 1; ep <= tvdbEpisodeCount(e.tmdb, s); ep++ {
				out = append(out, obj{"id": id*10000 + s*100 + ep, "seriesId": id, "seasonNumber": s, "episodeNumber": ep,
					"title": fmt.Sprintf("Chapter %d.%d", s, ep), "overview": overview(e.tmdb, "en"), "airDate": first.AddDate(0, 0, 7*(ep-1)).Format("2006-01-02"),
					"runtime": 40 + e.tmdb%20, "hasFile": ep <= files, "monitored": monitored, "images": nil})
			}
		}
		writeJSON(w, out)
	})
	// MonitorSeasons on a series Sonarr already has: PUT the series back with more seasons monitored, then search.
	h("PUT /api/v3/series/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.PathValue("id"))
		e, ok := a.items[id]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var b struct {
			Seasons []struct {
				SeasonNumber int  `json:"seasonNumber"`
				Monitored    bool `json:"monitored"`
			} `json:"seasons"`
		}
		_ = json.NewDecoder(r.Body).Decode(&b)
		for _, s := range b.Seasons {
			k := sort.SearchInts(e.seasons, s.SeasonNumber)
			if s.Monitored && (k >= len(e.seasons) || e.seasons[k] != s.SeasonNumber) {
				e.seasons = append(e.seasons, s.SeasonNumber)
				sort.Ints(e.seasons)
			}
		}
		w.WriteHeader(http.StatusAccepted)
		writeJSON(w, a.seriesJSON(e))
	})
	h("POST /api/v3/command", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, obj{"id": 1, "name": "SeasonSearch", "status": "queued"})
	})
	h("POST /api/v3/series", func(w http.ResponseWriter, r *http.Request) {
		var b struct {
			TVDBID  int    `json:"tvdbId"`
			Title   string `json:"title"`
			Seasons []struct {
				SeasonNumber int  `json:"seasonNumber"`
				Monitored    bool `json:"monitored"`
			} `json:"seasons"`
		}
		_ = json.NewDecoder(r.Body).Decode(&b)
		a.next++
		e := &arrEntry{id: a.next, tvdb: b.TVDBID, tmdb: b.TVDBID - 90000, title: b.Title, added: time.Now()}
		for _, s := range b.Seasons {
			if s.Monitored {
				e.seasons = append(e.seasons, s.SeasonNumber)
			}
		}
		sort.Ints(e.seasons)
		a.items[e.id] = e
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, obj{"id": e.id, "tvdbId": e.tvdb})
	})
	return mux
}
