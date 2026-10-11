package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type obj = map[string]any

// img is a poster file name long enough for Tipsarr's image proxy (real TMDB names are 27 characters).
func img(prefix string, id int) string { return fmt.Sprintf("/%s%06d-fake.jpg", prefix, id) }

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func listItem(id int, lang string) obj {
	if isShow(id) {
		return obj{"id": id, "media_type": "tv", "name": title(id, lang), "original_name": title(id, "en"), "first_air_date": releaseDate(id), "overview": overview(id, lang),
			"poster_path": img("t", id), "backdrop_path": img("bt", id), "vote_average": vote(id), "genre_ids": genreIDs(id), "popularity": popularity(id)}
	}
	return obj{"id": id, "media_type": "movie", "title": title(id, lang), "original_title": title(id, "en"), "release_date": releaseDate(id), "overview": overview(id, lang),
		"poster_path": img("m", id), "backdrop_path": img("b", id), "vote_average": vote(id), "genre_ids": genreIDs(id), "popularity": popularity(id)}
}

func personItem(id int) obj {
	return obj{"id": id, "media_type": "person", "name": personName(id), "profile_path": img("pp", id), "known_for_department": "Acting"}
}

func listOf(ids []int, lang string, mediaType string) []obj {
	out := make([]obj, 0, len(ids))
	for _, id := range ids {
		it := listItem(id, lang)
		if mediaType != "" {
			delete(it, "media_type")
		}
		out = append(out, it)
	}
	return out
}

func pageOf(ids []int, r *http.Request, mediaType string) obj {
	lang := r.URL.Query().Get("language")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	const per = 20
	total := (len(ids) + per - 1) / per
	if total < 1 {
		total = 1
	}
	lo, hi := (page-1)*per, page*per
	if lo > len(ids) {
		lo = len(ids)
	}
	if hi > len(ids) {
		hi = len(ids)
	}
	return obj{"page": page, "total_pages": total, "total_results": len(ids), "results": listOf(ids[lo:hi], lang, mediaType)}
}

func kwIDs(id int) []int { return []int{id%6 + 1, (id+2)%6 + 1} }

func hasAny(have []int, want map[int]bool) bool {
	for _, h := range have {
		if want[h] {
			return true
		}
	}
	return false
}

func numSet(s string) map[int]bool {
	out := map[int]bool{}
	for _, p := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == '|' }) {
		if n, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
			out[n] = true
		}
	}
	return out
}

func sortIDs(ids []int, by string) []int {
	out := append([]int(nil), ids...)
	switch {
	case strings.HasPrefix(by, "vote_average"):
		sort.SliceStable(out, func(i, j int) bool { return vote(out[i]) > vote(out[j]) })
	case strings.HasPrefix(by, "primary_release_date"), strings.HasPrefix(by, "first_air_date"), strings.HasPrefix(by, "release_date"):
		sort.SliceStable(out, func(i, j int) bool { return releaseDate(out[i]) > releaseDate(out[j]) })
	default:
		sort.SliceStable(out, func(i, j int) bool { return popularity(out[i]) > popularity(out[j]) })
	}
	return out
}

// pastOnly drops titles that are not out yet (they only show on Upcoming).
func pastOnly(ids []int) []int {
	out := []int{}
	for _, id := range ids {
		if releaseDate(id) <= today.Format("2006-01-02") {
			out = append(out, id)
		}
	}
	return out
}

func detail(id int, lang string) obj {
	d := listItem(id, lang)
	delete(d, "genre_ids")
	delete(d, "media_type")
	gs := []obj{}
	for _, g := range genreIDs(id) {
		name := genreByID(g).EN
		if isFR(lang) {
			name = genreByID(g).FR
		}
		gs = append(gs, obj{"id": g, "name": name})
	}
	d["genres"] = gs
	d["tagline"] = tagline(id, lang)
	d["status"] = "Released"
	d["original_language"] = "en"
	d["homepage"] = ""
	d["vote_count"] = 100 + id*17
	cast := []obj{}
	for k, p := range castOf(id) {
		cast = append(cast, obj{"id": p, "name": personName(p), "character": "Role " + strconv.Itoa(k+1), "profile_path": img("pp", p)})
	}
	crew := []obj{{"id": personFrst + (id*2)%15, "name": personName(personFrst + (id*2)%15), "job": "Director"}, {"id": personFrst + (id*5)%15, "name": personName(personFrst + (id*5)%15), "job": "Writer"}}
	d["credits"] = obj{"cast": cast, "crew": crew}
	videos := []obj{{"key": fmt.Sprintf("TRLEN%06d", id), "site": "YouTube", "type": "Trailer", "official": true, "iso_639_1": "en"}}
	if id%2 == 0 {
		videos = append(videos, obj{"key": fmt.Sprintf("TRLFR%06d", id), "site": "YouTube", "type": "Trailer", "official": true, "iso_639_1": "fr"})
	}
	// TIPSARR_FAKE_TRAILER=<YouTube key> makes every trailer a real video, to try playback in the apps.
	if key := os.Getenv("TIPSARR_FAKE_TRAILER"); key != "" {
		videos = []obj{{"key": key, "site": "YouTube", "type": "Trailer", "official": true, "iso_639_1": "en"}}
	}
	d["videos"] = obj{"results": videos}
	d["recommendations"] = obj{"results": relatedOf(id, lang, 0)}
	d["similar"] = obj{"results": relatedOf(id, lang, 1)}
	kws := []obj{}
	for _, k := range kwIDs(id) {
		kws = append(kws, obj{"id": k, "name": keywords[k]})
	}
	if isShow(id) {
		d["keywords"] = obj{"results": kws}
	} else {
		d["keywords"] = obj{"keywords": kws}
	}
	d["reviews"] = obj{"results": []obj{{"author": "reviewer" + strconv.Itoa(id%5), "content": "A fine fake review for title " + strconv.Itoa(id) + ".", "url": "https://example.org/r/" + strconv.Itoa(id), "created_at": "2026-09-01T10:00:00Z", "author_details": obj{"rating": 7.0}}}}
	d["production_companies"] = []obj{{"name": "Fake Pictures"}, {"name": "Studio " + strconv.Itoa(id%7)}}
	d["production_countries"] = []obj{{"name": "United States of America"}}
	d["spoken_languages"] = []obj{{"english_name": "English"}}
	if isShow(id) {
		n := seasonCount(id)
		seasons := []obj{{"season_number": 0, "name": "Specials", "episode_count": 2}}
		total := 0
		for s := 1; s <= n; s++ {
			seasons = append(seasons, obj{"season_number": s, "name": "Season " + strconv.Itoa(s), "episode_count": episodeCount(id, s), "air_date": time.Date(year(id)+s-1, 1, 10, 0, 0, 0, 0, time.UTC).Format("2006-01-02"), "poster_path": img("t", id)})
			total += episodeCount(id, s)
		}
		d["seasons"], d["number_of_seasons"], d["number_of_episodes"] = seasons, n, total
		d["episode_run_time"] = []int{40 + id%20}
		d["external_ids"] = obj{"imdb_id": imdbID(id), "tvdb_id": tvdbID(id)}
		d["networks"] = []obj{{"name": "Fake Network"}}
		d["created_by"] = []obj{{"id": personFrst + id%15, "name": personName(personFrst + id%15)}}
	} else {
		d["runtime"] = runtime(id)
		d["imdb_id"] = imdbID(id)
		d["external_ids"] = obj{"imdb_id": imdbID(id)}
		d["budget"], d["revenue"] = int64(20000000+id*100000), int64(60000000+id*900000)
		if c := collectionOf(id); c != 0 {
			d["belongs_to_collection"] = obj{"id": c, "name": "Saga " + strconv.Itoa(c-700), "backdrop_path": img("b", 1+(c-701)*3)}
		}
	}
	return d
}

func relatedOf(id int, lang string, shift int) []obj {
	pool := allMovies()
	if isShow(id) {
		pool = allShows()
	}
	out := []int{}
	for k := 1; len(out) < 8; k++ {
		c := pool[(id*(3+shift)+k*5)%len(pool)]
		if c != id {
			out = append(out, c)
		}
	}
	return listOf(pastOnly(out), lang, "")
}

func newTMDB() http.Handler {
	mux := http.NewServeMux()
	hit := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("api_key") != "devkey" {
				w.WriteHeader(http.StatusUnauthorized)
				writeJSON(w, obj{"status_code": 7, "status_message": "Invalid API key"})
				return
			}
			h(w, r)
		}
	}
	mux.HandleFunc("/trending/all/week", hit(func(w http.ResponseWriter, r *http.Request) {
		ids := []int{}
		for _, id := range sortIDs(pastOnly(append(allMovies(), allShows()...)), "popularity") {
			ids = append(ids, id)
		}
		writeJSON(w, pageOf(ids[:60], r, ""))
	}))
	discover := func(shows bool) http.HandlerFunc {
		return hit(func(w http.ResponseWriter, r *http.Request) {
			q := r.URL.Query()
			ids := pastOnly(allMovies())
			if shows {
				ids = allShows()
			}
			if g := q.Get("with_genres"); g != "" {
				want, out := numSet(g), []int{}
				for _, id := range ids {
					if hasAny(genreIDs(id), want) {
						out = append(out, id)
					}
				}
				ids = out
			}
			if k := q.Get("with_keywords"); k != "" {
				want, out := numSet(k), []int{}
				for _, id := range ids {
					if hasAny(kwIDs(id), want) {
						out = append(out, id)
					}
				}
				ids = out
			}
			writeJSON(w, pageOf(sortIDs(ids, q.Get("sort_by")), r, map[bool]string{true: "tv", false: "movie"}[shows]))
		})
	}
	mux.HandleFunc("/discover/movie", discover(false))
	mux.HandleFunc("/discover/tv", discover(true))
	mux.HandleFunc("/movie/upcoming", hit(func(w http.ResponseWriter, r *http.Request) {
		ids := []int{}
		for id := 100; id <= 110; id++ {
			ids = append(ids, id)
		}
		writeJSON(w, pageOf(ids, r, "movie"))
	}))
	mux.HandleFunc("/genre/{kind}/list", hit(func(w http.ResponseWriter, r *http.Request) {
		kind, out := r.PathValue("kind"), []obj{}
		for _, g := range genres {
			if (kind == "tv" && g.TV) || (kind == "movie" && g.Movi) {
				name := g.EN
				if isFR(r.URL.Query().Get("language")) {
					name = g.FR
				}
				out = append(out, obj{"id": g.ID, "name": name})
			}
		}
		writeJSON(w, obj{"genres": out})
	}))
	search := func(movies, shows, people bool) http.HandlerFunc {
		return hit(func(w http.ResponseWriter, r *http.Request) {
			q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("query")))
			ids := []int{}
			pool := []int{}
			if movies {
				pool = append(pool, allMovies()...)
			}
			if shows {
				pool = append(pool, allShows()...)
			}
			for _, id := range pool {
				if q != "" && (strings.Contains(strings.ToLower(title(id, "en")), q) || strings.Contains(strings.ToLower(title(id, "fr")), q)) {
					ids = append(ids, id)
				}
			}
			if len(ids) == 0 && q != "" && movies && !shows && !people && r.URL.Query().Get("year") == "" {
				// box-office chart titles are real film names: answer with one deterministic stand-in so the chart shows posters
				h := 0
				for _, c := range q {
					h = (h*31 + int(c)) % 90
				}
				id := 1 + h
				it := listItem(id, r.URL.Query().Get("language"))
				it["title"] = strings.ToUpper(q[:1]) + q[1:]
				delete(it, "media_type")
				writeJSON(w, obj{"page": 1, "total_pages": 1, "results": []obj{it}})
				return
			}
			mt := ""
			if movies && !shows {
				mt = "movie"
			} else if shows && !movies {
				mt = "tv"
			}
			res := pageOf(sortIDs(ids, ""), r, mt)
			if people && q != "" {
				items := res["results"].([]obj)
				for p := personFrst; p <= personLast; p++ {
					if strings.Contains(strings.ToLower(personName(p)), q) {
						items = append(items, personItem(p))
					}
				}
				res["results"] = items
			}
			writeJSON(w, res)
		})
	}
	mux.HandleFunc("/search/multi", search(true, true, true))
	mux.HandleFunc("/search/movie", search(true, false, false))
	mux.HandleFunc("/search/tv", search(false, true, false))
	mux.HandleFunc("/search/keyword", hit(func(w http.ResponseWriter, r *http.Request) {
		q, out := strings.ToLower(r.URL.Query().Get("query")), []obj{}
		for id, name := range keywords {
			if q != "" && strings.Contains(name, q) {
				out = append(out, obj{"id": id, "name": name})
			}
		}
		writeJSON(w, obj{"page": 1, "total_pages": 1, "results": out})
	}))
	mux.HandleFunc("/keyword/{id}", hit(func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.PathValue("id"))
		if name, ok := keywords[id]; ok {
			writeJSON(w, obj{"id": id, "name": name})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	mux.HandleFunc("/movie/{id}", hit(func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.PathValue("id"))
		if !isMovie(id) {
			w.WriteHeader(http.StatusNotFound)
			writeJSON(w, obj{"status_code": 34, "status_message": "not found"})
			return
		}
		writeJSON(w, detail(id, r.URL.Query().Get("language")))
	}))
	mux.HandleFunc("/tv/{id}", hit(func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.PathValue("id"))
		if !isShow(id) {
			w.WriteHeader(http.StatusNotFound)
			writeJSON(w, obj{"status_code": 34, "status_message": "not found"})
			return
		}
		writeJSON(w, detail(id, r.URL.Query().Get("language")))
	}))
	related := func(kind, which string) http.HandlerFunc {
		shift := map[string]int{"recommendations": 0, "similar": 1}[which]
		return hit(func(w http.ResponseWriter, r *http.Request) {
			id, _ := strconv.Atoi(r.PathValue("id"))
			writeJSON(w, obj{"page": 1, "total_pages": 1, "results": relatedOf(id, r.URL.Query().Get("language"), shift)})
		})
	}
	mux.HandleFunc("/movie/{id}/recommendations", related("movie", "recommendations"))
	mux.HandleFunc("/movie/{id}/similar", related("movie", "similar"))
	mux.HandleFunc("/tv/{id}/recommendations", related("tv", "recommendations"))
	mux.HandleFunc("/tv/{id}/similar", related("tv", "similar"))
	mux.HandleFunc("/tv/{id}/season/{n}", hit(func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.PathValue("id"))
		n, _ := strconv.Atoi(r.PathValue("n"))
		lang := r.URL.Query().Get("language")
		if !isShow(id) || n < 0 || n > seasonCount(id) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		eps := []obj{}
		for e := 1; e <= episodeCount(id, n); e++ {
			name := "Episode " + strconv.Itoa(e)
			if isFR(lang) {
				name = "Épisode " + strconv.Itoa(e)
			}
			eps = append(eps, obj{"episode_number": e, "name": name, "overview": overview(id, lang), "air_date": time.Date(year(id)+n-1, 1, 10+e, 0, 0, 0, 0, time.UTC).Format("2006-01-02"),
				"still_path": img("s", id), "runtime": 40 + (id+e)%20, "vote_average": vote(id + e)})
		}
		writeJSON(w, obj{"season_number": n, "name": "Season " + strconv.Itoa(n), "overview": overview(id, lang), "air_date": time.Date(year(id)+n-1, 1, 10, 0, 0, 0, 0, time.UTC).Format("2006-01-02"),
			"poster_path": img("t", id), "episodes": eps})
	}))
	mux.HandleFunc("/person/{id}", hit(func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.PathValue("id"))
		if id < personFrst || id > personLast {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		lang := r.URL.Query().Get("language")
		credits := []obj{}
		for _, t := range append(allMovies(), allShows()...) {
			for _, p := range castOf(t) {
				if p == id && releaseDate(t) <= today.Format("2006-01-02") {
					credits = append(credits, listItem(t, lang))
				}
			}
		}
		bio := "A fake actor, born for tests."
		if isFR(lang) {
			bio = "Un acteur fictif, né pour les tests."
		}
		writeJSON(w, obj{"id": id, "name": personName(id), "profile_path": img("pp", id), "known_for_department": "Acting", "biography": bio,
			"birthday": "1980-0" + strconv.Itoa(1+id%9) + "-15", "place_of_birth": "Lyon, France", "combined_credits": obj{"cast": credits}})
	}))
	mux.HandleFunc("/collection/{id}", hit(func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.PathValue("id"))
		if id < 701 || id > 703 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		lang := r.URL.Query().Get("language")
		first := 1 + (id-701)*3
		writeJSON(w, obj{"id": id, "name": "Saga " + strconv.Itoa(id-700), "overview": "A fake saga.", "poster_path": img("m", first), "backdrop_path": img("b", first),
			"parts": listOf([]int{first, first + 1, first + 2}, lang, "movie")})
	}))
	return mux
}
