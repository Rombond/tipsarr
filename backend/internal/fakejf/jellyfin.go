// Package fakejf is a fake Jellyfin server: the endpoints Tipsarr calls, answered from a fixed catalogue
// (users, library, watch history, the Playback Reporting plugin). It is used by the dev environment (cmd/devenv) and
// by the public demo (cmd/demo-gateway). Never use it in production.
package fakejf

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// User is a Jellyfin account.
type User struct {
	ID    string // 32 hex characters
	Name  string
	Admin bool
}

// Title is a movie or a series of the library.
type Title struct {
	TMDB         int
	Series       bool
	Name         string
	Year         int
	Genres       []string
	RuntimeMin   int
	Rating       float64
	AddedDaysAgo int
	Seasons      []int         // series: the season numbers in the library
	Episodes     func(int) int // series: episodes in a season
}

// Watched is what one person has seen: whole movies, and the first library season of each show.
type Watched struct {
	Movies []int
	Shows  []int
}

// Catalog is everything the fake answers from.
type Catalog struct {
	ServerName string
	APIKey     string // the token Tipsarr must send
	Password   string // every account's password
	Users      []User
	Titles     []Title
	History    map[string]Watched // by user name
	Now        time.Time          // the fixed "today" of the history
	// Art returns the label, subtitle and hue (0..1) of a generated poster for a title.
	Art func(t Title) (label, sub string, hue float64)
}

type obj = map[string]any

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

const (
	kindMovie   = 1
	kindSeries  = 2
	kindEpisode = 3
)

// ID is the Jellyfin item id of a movie (kind 1), series (2) or episode (3, n from EpisodeN).
func ID(kind int, n uint64) string { return fmt.Sprintf("%032x", uint64(kind)<<56|n) }

// EpisodeN packs a show, season and episode into an item number.
func EpisodeN(show, season, ep int) uint64 { return uint64(show)<<16 | uint64(season)<<8 | uint64(ep) }

func parseID(id string) (kind int, n uint64, ok bool) {
	if len(id) != 32 {
		return 0, 0, false
	}
	raw, err := strconv.ParseUint(id[16:], 16, 64)
	if err != nil {
		return 0, 0, false
	}
	return int(raw >> 56), raw & (1<<56 - 1), true
}

type play struct {
	user, item, kind, name string
	at                     time.Time
	secs                   int
}

type fake struct {
	c      Catalog
	byTMDB map[[2]int]Title // {series 0/1, tmdb}
	byID   map[string]User
}

func (f *fake) title(series bool, tmdb int) (Title, bool) {
	k := 0
	if series {
		k = 1
	}
	t, ok := f.byTMDB[[2]int{k, tmdb}]
	return t, ok
}

func (f *fake) plays(u User) []play {
	h := f.c.History[u.Name]
	out := []play{}
	for k, m := range h.Movies {
		t, _ := f.title(false, m)
		out = append(out, play{u.ID, ID(kindMovie, uint64(m)), "Movie", t.Name, f.c.Now.AddDate(0, 0, -(2 + k*3)), t.RuntimeMin * 60})
	}
	for k, s := range h.Shows {
		t, ok := f.title(true, s)
		if !ok || len(t.Seasons) == 0 {
			continue
		}
		sn := t.Seasons[0]
		for e := 1; e <= t.Episodes(sn); e++ {
			out = append(out, play{u.ID, ID(kindEpisode, EpisodeN(s, sn, e)), "Episode", fmt.Sprintf("%s - s%02de%02d - Episode %d", t.Name, sn, e, e),
				f.c.Now.AddDate(0, 0, -(1 + k*5 + e)), (40 + s%20) * 60})
		}
	}
	return out
}

func (f *fake) item(t Title) obj {
	it := obj{"Id": ID(kindMovie, uint64(t.TMDB)), "Name": t.Name, "Type": "Movie", "ProviderIds": obj{"Tmdb": strconv.Itoa(t.TMDB)}, "Genres": t.Genres,
		"ProductionYear": t.Year, "RunTimeTicks": int64(t.RuntimeMin) * 60 * 10000000, "CommunityRating": t.Rating,
		"DateCreated": f.c.Now.AddDate(0, 0, -t.AddedDaysAgo).Format("2006-01-02T15:04:05Z"), "ImageTags": obj{"Primary": "tag" + strconv.Itoa(t.TMDB)}}
	if t.Series {
		it["Id"], it["Type"] = ID(kindSeries, uint64(t.TMDB)), "Series"
	}
	return it
}

// New returns the fake Jellyfin.
func New(c Catalog) http.Handler {
	f := &fake{c: c, byTMDB: map[[2]int]Title{}, byID: map[string]User{}}
	for _, t := range c.Titles {
		k := 0
		if t.Series {
			k = 1
		}
		f.byTMDB[[2]int{k, t.TMDB}] = t
	}
	for _, u := range c.Users {
		f.byID[u.ID] = u
	}
	mux := http.NewServeMux()
	policy := func(u User) obj { return obj{"IsAdministrator": u.Admin, "IsDisabled": false} }
	keyed := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.Header.Get("Authorization"), `Token="`+c.APIKey+`"`) || r.Header.Get("X-Emby-Token") != "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			h(w, r)
		}
	}
	items := func(w http.ResponseWriter, list []obj) {
		writeJSON(w, obj{"Items": list, "TotalRecordCount": len(list)})
	}

	mux.HandleFunc("/System/Info/Public", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, obj{"ServerName": c.ServerName, "Version": "10.10.0", "Id": "fakejf"})
	})
	mux.HandleFunc("POST /Users/AuthenticateByName", func(w http.ResponseWriter, r *http.Request) {
		var in struct{ Username, Pw string }
		_ = json.NewDecoder(r.Body).Decode(&in)
		for _, u := range c.Users {
			if strings.EqualFold(in.Username, u.Name) && in.Pw == c.Password {
				writeJSON(w, obj{"AccessToken": "t", "User": obj{"Id": u.ID, "Name": u.Name, "Policy": policy(u)}})
				return
			}
		}
		w.WriteHeader(http.StatusUnauthorized)
	})
	mux.HandleFunc("/Users", keyed(func(w http.ResponseWriter, _ *http.Request) {
		out := []obj{}
		for _, u := range c.Users {
			out = append(out, obj{"Id": u.ID, "Name": u.Name, "Policy": policy(u)})
		}
		writeJSON(w, out)
	}))
	mux.HandleFunc("GET /Users/{id}", keyed(func(w http.ResponseWriter, r *http.Request) {
		u, ok := f.byID[r.PathValue("id")]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		writeJSON(w, obj{"Id": u.ID, "Name": u.Name, "Policy": policy(u)})
	}))
	mux.HandleFunc("GET /Plugins", keyed(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, []obj{{"Id": "p1", "Name": "Webhook", "Status": "Active"}, {"Id": "5c53438191a343cb907a35aa02eb9d2c", "Name": "Playback Reporting", "Status": "Active"}})
	}))
	// the Playback Reporting plugin: only the fixed queries Tipsarr sends
	afterRe := regexp.MustCompile(`rowid > (\d+)`)
	mux.HandleFunc("POST /user_usage_stats/submit_custom_query", keyed(func(w http.ResponseWriter, r *http.Request) {
		var in struct{ CustomQueryString string }
		_ = json.NewDecoder(r.Body).Decode(&in)
		q := in.CustomQueryString
		if !strings.HasPrefix(q, "SELECT ") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		all := [][]string{}
		for _, u := range c.Users {
			for _, p := range f.plays(u) {
				all = append(all, []string{"", p.at.Format("2006-01-02 15:04:05.0000000"), p.user, p.item, p.kind, p.name, strconv.Itoa(p.secs)})
			}
		}
		for i := range all {
			all[i][0] = strconv.Itoa(i + 1)
		}
		out := obj{"colums": []string{"x"}, "results": [][]string{}, "message": ""}
		switch {
		case strings.Contains(q, "MAX(rowid)"):
			out["results"] = [][]string{{strconv.Itoa(len(all))}}
		case afterRe.MatchString(q):
			after, _ := strconv.Atoi(afterRe.FindStringSubmatch(q)[1])
			if after < len(all) {
				out["results"] = all[after:]
			}
		}
		writeJSON(w, out)
	}))
	mux.HandleFunc("GET /Items/{id}/Images/Primary", keyed(func(w http.ResponseWriter, r *http.Request) {
		kind, n, ok := parseID(r.PathValue("id"))
		t, found := f.title(kind == kindSeries, int(n))
		if !ok || !found || c.Art == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		label, sub, hue := c.Art(t)
		WriteJPEG(w, Poster(300, 450, hue, label, sub))
	}))
	mux.HandleFunc("/Items", keyed(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if ids := q.Get("Ids"); ids != "" { // episode -> its show
			list := []obj{}
			for _, id := range strings.Split(ids, ",") {
				_, n, _ := parseID(id)
				list = append(list, obj{"Id": id, "Type": "Episode", "SeriesId": ID(kindSeries, n>>16)})
			}
			items(w, list)
			return
		}
		switch q.Get("IncludeItemTypes") {
		case "Movie,Series":
			list := []obj{}
			for _, t := range c.Titles {
				list = append(list, f.item(t))
			}
			items(w, list)
		case "Episode":
			list := []obj{}
			for _, t := range c.Titles {
				if !t.Series {
					continue
				}
				for _, sn := range t.Seasons {
					for e := 1; e <= t.Episodes(sn); e++ {
						list = append(list, obj{"Id": ID(kindEpisode, EpisodeN(t.TMDB, sn, e)), "Type": "Episode", "SeriesId": ID(kindSeries, uint64(t.TMDB)), "ParentIndexNumber": sn, "IndexNumber": e})
					}
				}
			}
			items(w, list)
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	mux.HandleFunc("/Users/{id}/Items", keyed(func(w http.ResponseWriter, r *http.Request) {
		u, ok := f.byID[r.PathValue("id")]
		if !ok {
			items(w, nil)
			return
		}
		list := []obj{}
		seenMovie := map[string]bool{}
		for _, p := range f.plays(u) {
			ud := obj{"Played": true, "PlayCount": 1, "LastPlayedDate": p.at.Format("2006-01-02T15:04:05Z")}
			_, n, _ := parseID(p.item)
			if p.kind == "Movie" {
				if seenMovie[p.item] {
					continue
				}
				seenMovie[p.item] = true
				list = append(list, obj{"Id": p.item, "Type": "Movie", "ProviderIds": obj{"Tmdb": strconv.FormatUint(n, 10)}, "UserData": ud})
				continue
			}
			list = append(list, obj{"Id": p.item, "Type": "Episode", "SeriesId": ID(kindSeries, n>>16), "UserData": ud})
		}
		items(w, list)
	}))
	return mux
}
