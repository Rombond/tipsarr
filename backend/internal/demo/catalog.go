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
	{550, "Fight Club", 1999, 139, "Drama", 8.4, nil},
	{603, "The Matrix", 1999, 136, "Action,Science Fiction", 8.2, nil},
	{155, "The Dark Knight", 2008, 152, "Drama,Action,Crime", 8.5, nil},
	{27205, "Inception", 2010, 148, "Action,Science Fiction,Adventure", 8.4, nil},
	{157336, "Interstellar", 2014, 169, "Adventure,Drama,Science Fiction", 8.4, nil},
	{13, "Forrest Gump", 1994, 142, "Comedy,Drama,Romance", 8.5, nil},
	{680, "Pulp Fiction", 1994, 154, "Thriller,Crime", 8.5, nil},
	{278, "The Shawshank Redemption", 1994, 142, "Drama,Crime", 8.7, nil},
	{238, "The Godfather", 1972, 175, "Drama,Crime", 8.7, nil},
	{120, "The Lord of the Rings: The Fellowship of the Ring", 2001, 179, "Adventure,Fantasy,Action", 8.4, nil},
	{121, "The Lord of the Rings: The Two Towers", 2002, 179, "Adventure,Fantasy,Action", 8.4, nil},
	{122, "The Lord of the Rings: The Return of the King", 2003, 201, "Adventure,Fantasy,Action", 8.5, nil},
	{11, "Star Wars", 1977, 121, "Adventure,Action,Science Fiction", 8.2, nil},
	{105, "Back to the Future", 1985, 116, "Adventure,Comedy,Science Fiction", 8.3, nil},
	{329, "Jurassic Park", 1993, 127, "Adventure,Science Fiction", 7.9, nil},
	{862, "Toy Story", 1995, 81, "Animation,Adventure,Family,Comedy", 8.0, nil},
	{129, "Spirited Away", 2001, 125, "Animation,Family,Fantasy", 8.5, nil},
	{497, "The Green Mile", 1999, 189, "Fantasy,Drama,Crime", 8.5, nil},
	{98, "Gladiator", 2000, 155, "Action,Drama,Adventure", 8.2, nil},
	{78, "Blade Runner", 1982, 117, "Science Fiction,Drama,Thriller", 7.9, nil},
	{438631, "Dune", 2021, 155, "Science Fiction,Adventure", 7.8, nil},
	{872585, "Oppenheimer", 2023, 180, "Drama,History", 8.1, nil},
	{496243, "Parasite", 2019, 133, "Comedy,Thriller,Drama", 8.5, nil},
	{76341, "Mad Max: Fury Road", 2015, 121, "Action,Adventure,Science Fiction", 7.6, nil},
	{14160, "Up", 2009, 96, "Animation,Comedy,Family,Adventure", 7.9, nil},
	{194, "Amélie", 2001, 122, "Comedy,Romance", 7.9, nil},
	{8587, "The Lion King", 1994, 89, "Family,Animation,Drama", 8.3, nil},
	{1124, "The Prestige", 2006, 130, "Drama,Mystery,Science Fiction", 8.2, nil},
}

var demoShows = []libEntry{
	{1396, "Breaking Bad", 2008, 47, "Drama,Crime", 8.9, []int{7, 13, 13, 13, 16}},
	{1399, "Game of Thrones", 2011, 60, "Sci-Fi & Fantasy,Drama,Action & Adventure", 8.5, []int{10, 10, 10, 10, 10, 10}},
	{66732, "Stranger Things", 2016, 50, "Drama,Sci-Fi & Fantasy,Mystery", 8.6, []int{8, 9, 8}},
	{94605, "Arcane", 2021, 40, "Animation,Drama,Sci-Fi & Fantasy", 8.7, []int{9}},
	{100088, "The Last of Us", 2023, 55, "Drama", 8.6, []int{9}},
	{95396, "Severance", 2022, 50, "Drama,Mystery,Sci-Fi & Fantasy", 8.4, []int{9}},
	{87108, "Chernobyl", 2019, 65, "Drama", 8.7, []int{5}},
	{60059, "Better Call Saul", 2015, 46, "Crime,Drama", 8.7, []int{10, 10, 10}},
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
			AdminUser: {Movies: []int{550, 603, 155, 27205, 157336, 680, 278, 120, 121, 122, 438631, 1124}, Shows: []int{1396, 1399, 94605}},
			PlainUser: {Movies: []int{13, 862, 129, 14160, 8587, 105, 329, 11}, Shows: []int{66732, 94605}},
			"carol":   {Movies: []int{238, 497, 98, 496243, 194}, Shows: []int{60059}},
			"dave":    {Movies: []int{78, 76341, 872585}},
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
