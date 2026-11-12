package demo

import (
	"strings"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/fakejf"
)

// The demo Jellyfin library: real TMDB titles, so detail pages, posters and suggestions come from TMDB.
type libEntry struct {
	tmdb    int
	name    string
	year    int
	runtime int
	genres  string
	rating  float64
	eps     []int // series: episodes per season (season 1 first)
}

var demoMovies = []libEntry{
	{862, "Toy Story", 1995, 81, "Animation,Adventure,Family,Comedy", 8.0, nil},
	{863, "Toy Story 2", 1999, 92, "Animation,Comedy,Family", 7.6, nil},
	{10193, "Toy Story 3", 2010, 103, "Animation,Family,Comedy", 7.8, nil},
	{129, "Spirited Away", 2001, 125, "Animation,Family,Fantasy", 8.5, nil},
	{8392, "My Neighbor Totoro", 1988, 86, "Fantasy,Animation,Family", 8.1, nil},
	{14160, "Up", 2009, 96, "Animation,Comedy,Family,Adventure", 7.9, nil},
	{8587, "The Lion King", 1994, 89, "Family,Animation,Drama", 8.3, nil},
	{12, "Finding Nemo", 2003, 100, "Animation,Family", 7.8, nil},
	{585, "Monsters, Inc.", 2001, 92, "Animation,Comedy,Family", 7.8, nil},
	{9806, "The Incredibles", 2004, 115, "Action,Adventure,Animation,Family", 7.7, nil},
	{10681, "WALL·E", 2008, 98, "Animation,Family,Science Fiction", 8.1, nil},
	{2062, "Ratatouille", 2007, 111, "Animation,Comedy,Family,Fantasy", 7.8, nil},
	{150540, "Inside Out", 2015, 95, "Animation,Family,Adventure,Drama,Comedy", 7.9, nil},
	{354912, "Coco", 2017, 105, "Family,Animation,Fantasy,Music,Comedy,Adventure", 8.2, nil},
	{109445, "Frozen", 2013, 102, "Animation,Family,Adventure,Fantasy", 7.2, nil},
	{502356, "The Super Mario Bros. Movie", 2023, 92, "Family,Comedy,Adventure,Fantasy,Animation", 7.6, nil},
	{671, "Harry Potter and the Philosopher's Stone", 2001, 152, "Adventure,Fantasy", 7.9, nil},
	{120, "The Lord of the Rings: The Fellowship of the Ring", 2001, 179, "Adventure,Fantasy,Action", 8.4, nil},
	{121, "The Lord of the Rings: The Two Towers", 2002, 179, "Adventure,Fantasy,Action", 8.4, nil},
	{122, "The Lord of the Rings: The Return of the King", 2003, 201, "Adventure,Fantasy,Action", 8.5, nil},
	{11, "Star Wars", 1977, 121, "Adventure,Action,Science Fiction", 8.2, nil},
	{105, "Back to the Future", 1985, 116, "Adventure,Comedy,Science Fiction", 8.3, nil},
	{329, "Jurassic Park", 1993, 127, "Adventure,Science Fiction", 7.9, nil},
	{27205, "Inception", 2010, 148, "Action,Science Fiction,Adventure", 8.4, nil},
	{157336, "Interstellar", 2014, 169, "Adventure,Drama,Science Fiction", 8.4, nil},
	{438631, "Dune", 2021, 155, "Science Fiction,Adventure", 7.8, nil},
	{13, "Forrest Gump", 1994, 142, "Comedy,Drama,Romance", 8.5, nil},
	{194, "Amélie", 2001, 122, "Comedy,Romance", 7.9, nil},
}

var demoShows = []libEntry{
	{2316, "The Office", 2005, 22, "Comedy", 8.6, []int{6, 22, 25, 19, 28, 26, 26, 24, 25}},
	{246, "Avatar: The Last Airbender", 2005, 24, "Animation,Action & Adventure,Sci-Fi & Fantasy", 8.7, []int{20, 20, 21}},
	{97546, "Ted Lasso", 2020, 35, "Comedy,Drama", 8.4, []int{10, 12, 12}},
	{82856, "The Mandalorian", 2019, 38, "Sci-Fi & Fantasy,Action & Adventure,Drama", 8.4, []int{8, 8, 8}},
	{95396, "Severance", 2022, 50, "Drama,Mystery,Sci-Fi & Fantasy", 8.4, []int{9}},
	{456, "The Simpsons", 1989, 22, "Family,Animation,Comedy", 8.0, []int{13, 22}},
}

// DemoUsers are the Jellyfin accounts of the demo (AdminUser is the Jellyfin and Tipsarr admin).
var DemoUsers = []fakejf.User{
	{ID: strings.Repeat("a", 31) + "1", Name: AdminUser, Admin: true},
	{ID: strings.Repeat("b", 31) + "2", Name: PlainUser},
	{ID: strings.Repeat("c", 31) + "3", Name: "carol"},
	{ID: strings.Repeat("d", 31) + "4", Name: "dave"},
}

// JellyfinAPIKey is the key the demo sandboxes use with the fake Jellyfin.
const JellyfinAPIKey = "demo-jellyfin-key"

// Catalog builds the fake Jellyfin catalogue of the demo. now is the day the history is relative to.
func Catalog(now time.Time) fakejf.Catalog {
	c := fakejf.Catalog{ServerName: "Tipsarr demo Jellyfin", APIKey: JellyfinAPIKey, Password: Password, Users: DemoUsers, Now: now,
		History: map[string]fakejf.Watched{
			AdminUser: {Movies: []int{27205, 157336, 438631, 120, 121, 122, 11, 329, 671, 10681}, Shows: []int{2316, 82856, 95396}},
			PlainUser: {Movies: []int{13, 862, 129, 14160, 8587, 105, 12, 585}, Shows: []int{246, 456}},
			"carol":   {Movies: []int{354912, 150540, 2062, 194, 8392}, Shows: []int{97546}},
			"dave":    {Movies: []int{9806, 502356, 109445}},
		},
		Art: func(t fakejf.Title) (string, string, float64) {
			return t.Name, "", float64(t.TMDB%20) / 20
		},
	}
	for i, m := range demoMovies {
		c.Titles = append(c.Titles, fakejf.Title{TMDB: m.tmdb, Name: m.name, Year: m.year, Genres: strings.Split(m.genres, ","), RuntimeMin: m.runtime,
			Rating: m.rating, AddedDaysAgo: 2 + (i*7)%60})
	}
	for i, s := range demoShows {
		eps := s.eps
		t := fakejf.Title{TMDB: s.tmdb, Series: true, Name: s.name, Year: s.year, Genres: strings.Split(s.genres, ","), RuntimeMin: s.runtime,
			Rating: s.rating, AddedDaysAgo: 5 + (i*11)%50, Episodes: func(n int) int {
				if n < 1 || n > len(eps) {
					return 0
				}
				return eps[n-1]
			}}
		for n := 1; n <= len(eps); n++ {
			t.Seasons = append(t.Seasons, n)
		}
		c.Titles = append(c.Titles, t)
	}
	return c
}
