package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type jfUser struct {
	ID    string
	Name  string
	Admin bool
}

func idOf(letter string) string { return strings.Repeat(letter, 31) }

var jfUsers = []jfUser{
	{idOf("a") + "1", "alice", true}, {idOf("b") + "2", "bob", false}, {idOf("c") + "3", "carol", false}, {idOf("d") + "4", "dave", false},
	{idOf("e") + "5", "erin", false},
}

// the library: movies 1..30 without two holes, and shows 201..210
func libMovies() []int {
	out := []int{}
	for i := 1; i <= 30; i++ {
		if i != 17 && i != 23 {
			out = append(out, i)
		}
	}
	return out
}

func libShows() []int {
	out := []int{}
	for i := 201; i <= 210; i++ {
		out = append(out, i)
	}
	return out
}

func jfID(kind, n int) string { return fmt.Sprintf("%032x", uint64(kind)<<32|uint64(n)) }

// seasons a library show has: all of them, except the newest for every third show (those show as partial)
func libSeasons(show int) []int {
	n := seasonCount(show)
	if show%3 == 0 && n > 1 {
		n--
	}
	out := []int{}
	for s := 1; s <= n; s++ {
		out = append(out, s)
	}
	return out
}

func jfName(id int) string {
	if id%5 == 0 {
		return title(id, "fr")
	}
	return title(id, "en")
}

func jfGenres(id int) []string {
	out := []string{}
	for _, g := range genreIDs(id) {
		name := genreByID(g).EN
		if id%7 == 0 {
			name = genreByID(g).FR // the franglais seen on real libraries
		}
		out = append(out, name)
	}
	return out
}

func daysAgo(d int) string { return today.AddDate(0, 0, -d).Format("2006-01-02T15:04:05Z") }

func libraryItem(id int) obj {
	it := obj{"Id": jfID(1, id), "Name": jfName(id), "Type": "Movie", "ProviderIds": obj{"Tmdb": strconv.Itoa(id)}, "Genres": jfGenres(id), "ProductionYear": year(id),
		"RunTimeTicks": int64(runtime(id)) * 60 * 10000000, "CommunityRating": vote(id), "DateCreated": daysAgo(3 + (id*11)%70), "ImageTags": obj{"Primary": "tag" + strconv.Itoa(id)}}
	if isShow(id) {
		it["Id"], it["Type"] = jfID(2, id), "Series"
		it["RunTimeTicks"] = int64(40+id%20) * 60 * 10000000
	}
	return it
}

// what each person watched
var history = map[string]struct {
	movies []int
	shows  []int
}{
	"alice": {[]int{1, 2, 3, 4, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 25}, []int{201, 202, 203}},
	"bob":   {[]int{2, 4, 6, 8, 10, 12, 14, 16}, []int{201, 205}},
	"carol": {[]int{1, 3, 5, 7, 9, 11}, []int{204}},
	"dave":  {[]int{2, 3, 4}, nil},
}

type play struct {
	user, item, kind, name string
	at                     time.Time
	secs                   int
}

func plays(user jfUser) []play {
	h := history[user.Name]
	out := []play{}
	for k, m := range h.movies {
		out = append(out, play{user.ID, jfID(1, m), "Movie", jfName(m), today.AddDate(0, 0, -(2 + k*3)), runtime(m) * 60})
	}
	for k, s := range h.shows {
		for _, sn := range libSeasons(s)[:1] { // the first season, every episode
			for e := 1; e <= episodeCount(s, sn); e++ {
				out = append(out, play{user.ID, jfID(3, s*10000+sn*100+e), "Episode", fmt.Sprintf("%s - s%02de%02d - Episode %d", title(s, "en"), sn, e, e),
					today.AddDate(0, 0, -(1 + k*5 + e)), (40 + s%20) * 60})
			}
		}
	}
	return out
}

func newJellyfin() http.Handler {
	mux := http.NewServeMux()
	byID := map[string]jfUser{}
	for _, u := range jfUsers {
		byID[u.ID] = u
	}
	policy := func(u jfUser) obj { return obj{"IsAdministrator": u.Admin, "IsDisabled": false} }
	keyed := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.Header.Get("Authorization"), `Token="jfkey"`) || r.Header.Get("X-Emby-Token") != "" {
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
		writeJSON(w, obj{"ServerName": "Tipsarr dev Jellyfin", "Version": "10.10.0", "Id": "devenv"})
	})
	mux.HandleFunc("POST /Users/AuthenticateByName", func(w http.ResponseWriter, r *http.Request) {
		var in struct{ Username, Pw string }
		_ = json.NewDecoder(r.Body).Decode(&in)
		for _, u := range jfUsers {
			if strings.EqualFold(in.Username, u.Name) && in.Pw == "test" {
				writeJSON(w, obj{"AccessToken": "t", "User": obj{"Id": u.ID, "Name": u.Name, "Policy": policy(u)}})
				return
			}
		}
		w.WriteHeader(http.StatusUnauthorized)
	})
	mux.HandleFunc("/Users", keyed(func(w http.ResponseWriter, _ *http.Request) {
		out := []obj{}
		for _, u := range jfUsers {
			out = append(out, obj{"Id": u.ID, "Name": u.Name, "Policy": policy(u)})
		}
		writeJSON(w, out)
	}))
	mux.HandleFunc("GET /Users/{id}", keyed(func(w http.ResponseWriter, r *http.Request) {
		u, ok := byID[r.PathValue("id")]
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
		for _, u := range jfUsers {
			for _, p := range plays(u) {
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
		raw, err := strconv.ParseUint(r.PathValue("id"), 16, 64)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		n := int(raw & 0xffffffff)
		name := "m" + strconv.Itoa(n)
		if raw>>32 == 2 {
			name = "t" + strconv.Itoa(n)
		}
		label, sub, _, hue := artFor(name)
		writeJPEG(w, poster(300, 450, hue, label, sub))
	}))
	mux.HandleFunc("/Items", keyed(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if ids := q.Get("Ids"); ids != "" { // episode -> its show
			list := []obj{}
			for _, id := range strings.Split(ids, ",") {
				raw, _ := strconv.ParseUint(id, 16, 64)
				show := int(raw&0xffffffff) / 10000
				list = append(list, obj{"Id": id, "Type": "Episode", "SeriesId": jfID(2, show)})
			}
			items(w, list)
			return
		}
		switch q.Get("IncludeItemTypes") {
		case "Movie,Series":
			list := []obj{}
			for _, m := range libMovies() {
				list = append(list, libraryItem(m))
			}
			for _, s := range libShows() {
				list = append(list, libraryItem(s))
			}
			items(w, list)
		case "Episode":
			list := []obj{}
			for _, s := range libShows() {
				for _, sn := range libSeasons(s) {
					for e := 1; e <= episodeCount(s, sn); e++ {
						list = append(list, obj{"Id": jfID(3, s*10000+sn*100+e), "Type": "Episode", "SeriesId": jfID(2, s), "ParentIndexNumber": sn, "IndexNumber": e})
					}
				}
			}
			items(w, list)
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	mux.HandleFunc("/Users/{id}/Items", keyed(func(w http.ResponseWriter, r *http.Request) {
		u, ok := byID[r.PathValue("id")]
		if !ok {
			items(w, nil)
			return
		}
		list := []obj{}
		seenMovie := map[string]bool{}
		for _, p := range plays(u) {
			ud := obj{"Played": true, "PlayCount": 1, "LastPlayedDate": p.at.Format("2006-01-02T15:04:05Z")}
			if p.kind == "Movie" {
				if seenMovie[p.item] {
					continue
				}
				seenMovie[p.item] = true
				raw, _ := strconv.ParseUint(p.item, 16, 64)
				list = append(list, obj{"Id": p.item, "Type": "Movie", "ProviderIds": obj{"Tmdb": strconv.Itoa(int(raw & 0xffffffff))}, "UserData": ud})
				continue
			}
			raw, _ := strconv.ParseUint(p.item, 16, 64)
			list = append(list, obj{"Id": p.item, "Type": "Episode", "SeriesId": jfID(2, int(raw&0xffffffff)/10000), "UserData": ud})
		}
		items(w, list)
	}))
	return mux
}
