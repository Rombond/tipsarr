package stats

import "testing"

// "Most watched" is ranked by time spent: a show with many short episode plays must not beat a
// movie that took longer in total, and a title never played is not listed.
func TestMostWatchedIsRankedByTimeSpent(t *testing.T) {
	titles := map[string]*title{
		"tv:1":    {typ: "tv", tmdb: 1, name: "Long show", plays: 40, secs: 40 * 1500},              // 16.7 h
		"movie:2": {typ: "movie", tmdb: 2, name: "Movie watched a lot", plays: 12, secs: 12 * 7200}, // 24 h
		"movie:3": {typ: "movie", tmdb: 3, name: "Movie once", plays: 1, secs: 6000},
		"tv:4":    {typ: "tv", tmdb: 4, name: "Show one episode", plays: 1, secs: 2400},
		"movie:5": {typ: "movie", tmdb: 5, name: "Never played", plays: 0, secs: 0},
	}
	res := &StatsReport{}
	aggregate(res, titles)
	var order []string
	for _, it := range res.Top {
		order = append(order, it.Title)
	}
	want := []string{"Movie watched a lot", "Long show", "Movie once", "Show one episode"}
	if len(order) != len(want) {
		t.Fatalf("top = %v", order)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("top = %v, want %v", order, want)
		}
	}
	if len(res.TopMovies) != 2 || res.TopMovies[0].Title != "Movie watched a lot" || len(res.TopShows) != 2 || res.TopShows[0].Title != "Long show" {
		t.Fatalf("per type = %+v / %+v", res.TopMovies, res.TopShows)
	}
}

// Genres are shown in the viewer's language, and two names that translate to the same genre merge.
func TestGenresAreTranslated(t *testing.T) {
	fr := map[string]string{"Adventure": "Aventure", "Science Fiction": "Science-Fiction"}
	name := func(g string) string {
		if n, ok := fr[g]; ok {
			return n
		}
		return g
	}
	titles := map[string]*title{
		"movie:1": {typ: "movie", tmdb: 1, plays: 1, secs: 7200, genres: []string{"Adventure", "Science Fiction", "Science-Fiction"}},
		"movie:2": {typ: "movie", tmdb: 2, plays: 1, secs: 3600, genres: []string{"Science-Fiction"}},
	}
	translateGenres(titles, name)
	res := &StatsReport{}
	aggregate(res, titles)
	if len(res.Genres) != 2 || res.Genres[0].Name != "Science-Fiction" || res.Genres[0].Titles != 2 || res.Genres[1].Name != "Aventure" {
		t.Fatalf("genres = %+v", res.Genres)
	}
}
