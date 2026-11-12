package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Rombond/tipsarr/backend/internal/fakejf"
)

func idOf(letter string) string { return strings.Repeat(letter, 31) }

var jfUsers = []fakejf.User{
	{ID: idOf("a") + "1", Name: "alice", Admin: true}, {ID: idOf("b") + "2", Name: "bob"}, {ID: idOf("c") + "3", Name: "carol"}, {ID: idOf("d") + "4", Name: "dave"},
	{ID: idOf("e") + "5", Name: "erin"},
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

// seasons a library show has: all of them, except the newest for every third show (those show as partial)
func libSeasons(show int) []int {
	n := tvdbSeasonCount(show) // Jellyfin folders follow Sonarr (TVDB) numbering
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

func libTitle(id int) fakejf.Title {
	t := fakejf.Title{TMDB: id, Name: jfName(id), Year: year(id), Genres: jfGenres(id), RuntimeMin: runtime(id), Rating: vote(id), AddedDaysAgo: 3 + (id*11)%70}
	if isShow(id) {
		t.Series, t.RuntimeMin, t.Seasons = true, 40+id%20, libSeasons(id)
		t.Episodes = func(s int) int { return tvdbEpisodeCount(id, s) }
	}
	return t
}

func newJellyfin() http.Handler {
	c := fakejf.Catalog{ServerName: "Tipsarr dev Jellyfin", APIKey: "jfkey", Password: "test", Users: jfUsers, Now: today,
		History: map[string]fakejf.Watched{
			"alice": {Movies: []int{1, 2, 3, 4, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 25}, Shows: []int{201, 202, 203}},
			"bob":   {Movies: []int{2, 4, 6, 8, 10, 12, 14, 16}, Shows: []int{201, 205}},
			"carol": {Movies: []int{1, 3, 5, 7, 9, 11}, Shows: []int{204}},
			"dave":  {Movies: []int{2, 3, 4}},
		},
		Art: func(t fakejf.Title) (string, string, float64) {
			p := "m"
			if t.Series {
				p = "t"
			}
			label, sub, _, hue := artFor(p + strconv.Itoa(t.TMDB))
			return label, sub, hue
		},
	}
	for _, m := range libMovies() {
		c.Titles = append(c.Titles, libTitle(m))
	}
	for _, s := range libShows() {
		c.Titles = append(c.Titles, libTitle(s))
	}
	return fakejf.New(c)
}
