package demo

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/fakejf"
)

// ArrKey is the API key the demo sandboxes use with the fake Radarr and Sonarr.
const ArrKey = "demo-arr-key"

// Internal paths of the gateway's loopback listener, as seen by a sandbox.
const (
	PathJellyfin  = "/jellyfin"
	PathRadarr    = "/radarr"
	PathSonarr    = "/sonarr"
	PathTMDB      = "/tmdb/3"      // TIPSARR_TMDB_URL
	PathTMDBImage = "/tmdbimg/t/p" // TIPSARR_TMDB_IMAGE_URL
	PathBoxOffice = "/boxoffice"   // TIPSARR_BOXOFFICE_URL
)

// Services is the handler of the gateway's internal listener: the fake Jellyfin, Radarr and Sonarr (read-only:
// sandboxes run in dry-run, so nothing is ever added) and the shared TMDB / box-office proxy.
func Services(now time.Time, proxy http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle(PathJellyfin+"/", http.StripPrefix(PathJellyfin, fakejf.New(Catalog(now))))
	mux.Handle(PathRadarr+"/", http.StripPrefix(PathRadarr, fakeArr("radarr")))
	mux.Handle(PathSonarr+"/", http.StripPrefix(PathSonarr, fakeArr("sonarr")))
	if proxy != nil {
		mux.Handle("/tmdb/", proxy)
		mux.Handle("/tmdbimg/", proxy)
		mux.Handle("/boxoffice/", proxy)
	}
	return mux
}

type obj = map[string]any

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// fakeArr is a stateless Radarr or Sonarr: it knows no title, lists profiles and folders, and refuses every write
// (a sandbox never writes: dry-run is forced in demo mode).
func fakeArr(kind string) http.Handler {
	mux := http.NewServeMux()
	h := func(pattern string, fn http.HandlerFunc) {
		mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Api-Key") != ArrKey {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			fn(w, r)
		})
	}
	empty := func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, []obj{}) }
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
		writeJSON(w, []obj{{"id": 1, "path": "/data/media/" + map[string]string{"radarr": "movies", "sonarr": "tv"}[kind], "freeSpace": 812345678901}})
	})
	h("GET /api/v3/tag", empty)
	h("GET /api/v3/languageprofile", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) })
	h("GET /api/v3/queue", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, obj{"records": []obj{}, "totalRecords": 0}) })
	h("GET /api/v3/movie", empty)
	h("GET /api/v3/series", empty)
	h("GET /api/v3/movie/lookup/tmdb", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.URL.Query().Get("tmdbId"))
		writeJSON(w, obj{"title": "", "tmdbId": id})
	})
	h("GET /api/v3/series/lookup", func(w http.ResponseWriter, r *http.Request) {
		tvdb, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Query().Get("term"), "tvdb:"))
		seasons := []obj{}
		for s := 0; s <= 12; s++ {
			seasons = append(seasons, obj{"seasonNumber": s, "monitored": false})
		}
		writeJSON(w, []obj{{"title": "", "tvdbId": tvdb, "seasons": seasons, "seriesType": "standard"}})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden) // writes, and anything else
	})
	return mux
}
