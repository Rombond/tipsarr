package demo

import (
	"bytes"
	"encoding/json"
	"strings"
)

// The public demo is used for store screenshots and app review (13+ age rating), so the TMDB answers it serves are
// cleaned of mature titles: adult entries, horror, and a denylist of titles with gory or explicit artwork that
// TMDB does not flag. Only the demo proxy applies this; a real Tipsarr talks to TMDB directly.

// unsafeGenres are TMDB genre ids never shown in the demo (27 = Horror).
var unsafeGenres = map[int]bool{27: true}

// unsafeMovies and unsafeShows are TMDB ids of titles kept out of the demo.
var (
	unsafeMovies = map[int]bool{
		764: true, 765: true, 766: true, 109428: true, 713704: true, // Evil Dead films
		550: true, 680: true, 807: true, 274: true, 24: true, 393: true, 68718: true, // Fight Club, Pulp Fiction, Se7en, Silence of the Lambs, Kill Bill, Django
		293660: true, 383498: true, 533535: true, // Deadpool films
		475557: true, 889737: true, // Joker films
		1034541: true, 663712: true, 423108: true, // Terrifier 3, Terrifier 2, The Conjuring 3
		933260: true, 1100782: true, // The Substance, Smile 2
		76341: true, 98: true, 1124: true, 496243: true, // Mad Max: Fury Road, Gladiator, The Prestige, Parasite (old demo library)
	}
	unsafeShows = map[int]bool{
		10545: true,               // True Blood
		76479: true, 205715: true, // The Boys, Gen V
		86423: true,                           // Locke & Key
		62264: true,                           // Ash vs Evil Dead
		1402:  true, 62286: true, 94305: true, // The Walking Dead and spin-offs
		1413: true, 40008: true, 1405: true, 90462: true, 72844: true, 1622: true, // AHS, Hannibal, Dexter, Chucky, Hill House, Supernatural
		1399: true, 94997: true, 93405: true, 85552: true, 95557: true, 46296: true, 63247: true, // GoT, HotD, Squid Game, Euphoria, Invincible, Spartacus, Westworld
		100088: true, 42009: true, 105248: true, 1396: true, 60059: true, 87108: true, 71912: true, // TLOU, Black Mirror, Edgerunners, BB, BCS, Chernobyl, The Witcher
	}
)

// SafeTMDB removes mature titles from a TMDB JSON answer (any list of movies or series at any depth: results,
// recommendations, similar, credits, known_for...). It returns the body unchanged when nothing is removed or the
// body is not JSON.
func SafeTMDB(body []byte) []byte {
	t := bytes.TrimSpace(body)
	if len(t) == 0 || (t[0] != '{' && t[0] != '[') {
		return body
	}
	d := json.NewDecoder(bytes.NewReader(body))
	d.UseNumber()
	var v any
	if d.Decode(&v) != nil {
		return body
	}
	v, changed := clean(v, "")
	if !changed {
		return body
	}
	out, err := json.Marshal(v)
	if err != nil {
		return body
	}
	return out
}

// clean walks v; kind is the media type implied by the enclosing key ("movie", "tv" or "").
func clean(v any, kind string) (any, bool) {
	changed := false
	switch x := v.(type) {
	case map[string]any:
		for k, c := range x {
			ck := kind
			switch {
			case k == "movie_results":
				ck = "movie"
			case k == "tv_results":
				ck = "tv"
			}
			n, ch := clean(c, ck)
			if ch {
				x[k] = n
				changed = true
			}
		}
		return x, changed
	case []any:
		out := x[:0:0]
		for _, c := range x {
			if m, ok := c.(map[string]any); ok && unsafeTitle(m, kind) {
				changed = true
				continue
			}
			n, ch := clean(c, kind)
			changed = changed || ch
			out = append(out, n)
		}
		if !changed {
			return x, false
		}
		return out, true
	}
	return v, false
}

func num(v any) (int, bool) {
	n, ok := v.(json.Number)
	if !ok {
		return 0, false
	}
	i, err := n.Int64()
	return int(i), err == nil
}

// unsafeTitle reports whether m is a movie or series entry of a list that must not be shown.
func unsafeTitle(m map[string]any, kind string) bool {
	id, ok := num(m["id"])
	if !ok {
		return false
	}
	mt, _ := m["media_type"].(string)
	if mt == "" {
		mt = kind
	}
	if mt == "" {
		if _, person := m["known_for_department"]; person {
			return false
		}
		_, title := m["title"]
		_, first := m["first_air_date"]
		_, orig := m["original_name"]
		switch {
		case title:
			mt = "movie"
		case first || orig:
			mt = "tv"
		default:
			return false // not a title (a person, a genre, a video...)
		}
	}
	if mt != "movie" && mt != "tv" {
		return false
	}
	if a, _ := m["adult"].(bool); a {
		return true
	}
	if gs, ok := m["genre_ids"].([]any); ok {
		for _, g := range gs {
			if n, ok := num(g); ok && unsafeGenres[n] {
				return true
			}
		}
	}
	if mt == "movie" {
		return unsafeMovies[id]
	}
	return unsafeShows[id]
}

// isTMDBJSON reports whether a proxied answer is a TMDB API JSON body.
func isTMDBJSON(contentType string) bool { return strings.Contains(contentType, "json") }
