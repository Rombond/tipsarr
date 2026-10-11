package main

import (
	"fmt"
	"strings"
	"time"
)

// The fake catalogue: ids decide everything, so every service agrees on the same titles.
// Movies 1..120, shows 201..240, people 501..515, collections 701..703.
const (
	movieMax   = 120
	showFirst  = 201
	showLast   = 240
	personFrst = 501
	personLast = 515
)

// A fixed "today", so release dates, "upcoming" and the box office stay in step whenever the env is started.
var today = time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)

type genre struct {
	ID   int
	EN   string
	FR   string
	TV   bool
	Movi bool
}

var genres = []genre{
	{28, "Action", "Action", false, true}, {12, "Adventure", "Aventure", false, true}, {16, "Animation", "Animation", true, true},
	{35, "Comedy", "Comédie", true, true}, {80, "Crime", "Crime", true, true}, {18, "Drama", "Drame", true, true},
	{10751, "Family", "Familial", true, true}, {14, "Fantasy", "Fantastique", false, true}, {27, "Horror", "Horreur", false, true},
	{9648, "Mystery", "Mystère", true, true}, {10749, "Romance", "Romance", false, true}, {878, "Science Fiction", "Science-Fiction", false, true},
	{53, "Thriller", "Thriller", false, true}, {99, "Documentary", "Documentaire", true, true},
	{10759, "Action & Adventure", "Action & Adventure", true, false}, {10765, "Sci-Fi & Fantasy", "Science-Fiction & Fantastique", true, false},
}

func genreByID(id int) genre {
	for _, g := range genres {
		if g.ID == id {
			return g
		}
	}
	return genres[0]
}

var (
	adjEN      = []string{"Silent", "Crimson", "Hidden", "Last", "Broken", "Golden", "Midnight", "Wild", "Frozen", "Burning", "Distant", "Lost", "Electric", "Hollow", "Rising"}
	nounEN     = []string{"Harbor", "Empire", "Garden", "Signal", "Horizon", "Kingdom", "Witness", "Tide", "Citadel", "Promise", "Orchard", "Frontier", "Echo", "Lantern", "River"}
	adjFR      = []string{"silencieux", "écarlate", "caché", "dernier", "brisé", "doré", "de minuit", "sauvage", "gelé", "ardent", "lointain", "perdu", "électrique", "creux", "naissant"}
	nounFR     = []string{"Port", "Empire", "Jardin", "Signal", "Horizon", "Royaume", "Témoin", "Marée", "Citadelle", "Serment", "Verger", "Frontière", "Écho", "Lanterne", "Fleuve"}
	firstNames = []string{"Ana", "Marc", "Léa", "Tomás", "Ines", "Hugo", "Nora", "Pablo", "Chloé", "Ivan", "Sara", "Jules", "Mina", "Omar", "Eva"}
	lastNames  = []string{"Moreau", "Silva", "Brandt", "Okafor", "Lindqvist", "Dubois", "Haddad", "Novak", "Rossi", "Tanaka", "Weber", "Costa", "Petit", "Aziz", "Berg"}
	keywords   = map[int]string{1: "heist", 2: "shark", 3: "time travel", 4: "revenge", 5: "road trip", 6: "space"}
)

func isFR(lang string) bool { return strings.HasPrefix(strings.ToLower(lang), "fr") }

func isMovie(id int) bool { return id >= 1 && id <= movieMax }
func isShow(id int) bool  { return id >= showFirst && id <= showLast }

func title(id int, lang string) string {
	a, n := id%15, (id*7)%15
	if isFR(lang) {
		article := "Le"
		if n == 2 || n == 4 || n == 5 || n == 7 { // gendered by eye, good enough for a fake
			article = "La"
		}
		if n == 0 || n == 6 || n == 8 || n == 3 || n == 12 {
			article = "Le"
		}
		return fmt.Sprintf("%s %s %s", article, nounFR[n], adjFR[a])
	}
	return fmt.Sprintf("%s %s", adjEN[a], nounEN[n])
}

func tagline(id int, lang string) string {
	if isFR(lang) {
		return fmt.Sprintf("Rien ne sera plus comme avant (%d).", id)
	}
	return fmt.Sprintf("Nothing will ever be the same (%d).", id)
}

func overview(id int, lang string) string {
	if isFR(lang) {
		return fmt.Sprintf("Un récit haletant autour de « %s ». Quand tout bascule, personne n’est à l’abri.", title(id, lang))
	}
	return fmt.Sprintf("A gripping story around \"%s\". When everything changes, nobody is safe.", title(id, lang))
}

// genreIDs are two genres chosen by id (shows use the TV ones where they differ).
func genreIDs(id int) []int {
	if isShow(id) {
		tv := []int{10759, 16, 35, 80, 18, 10751, 9648, 10765, 99}
		return []int{tv[id%len(tv)], tv[(id*3+1)%len(tv)]}
	}
	mv := []int{28, 12, 16, 35, 80, 18, 10751, 14, 27, 9648, 10749, 878, 53, 99}
	return []int{mv[id%len(mv)], mv[(id*5+2)%len(mv)]}
}

func year(id int) int { return 1995 + (id*3)%31 }

// releaseDate: ids 100..110 are upcoming, the rest are in the past.
func releaseDate(id int) string {
	if isMovie(id) && id >= 100 && id <= 110 {
		return today.AddDate(0, 0, (id-99)*9).Format("2006-01-02")
	}
	y := year(id)
	if id >= 90 && id < 100 { // a few from this year
		return time.Date(2026, time.Month(1+id%9), 1+id%27, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	}
	return time.Date(y, time.Month(1+id%12), 1+id%27, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
}

func vote(id int) float64       { return float64(50+(id*13)%45) / 10 }
func runtime(id int) int        { return 80 + (id*7)%80 }
func popularity(id int) float64 { return float64((id*37)%100) + 0.5 }

func seasonCount(id int) int          { return id%4 + 1 }
func episodeCount(id, season int) int { return 8 + (id+season)%5 }

func imdbID(id int) string { return fmt.Sprintf("tt%07d", 1000000+id*13) }
func tvdbID(id int) int    { return 90000 + id }

func collectionOf(id int) int {
	if id >= 1 && id <= 9 {
		return 701 + (id-1)/3
	}
	return 0
}

func personName(id int) string {
	k := id - personFrst
	return firstNames[k%15] + " " + lastNames[(k*4+1)%15]
}

func castOf(id int) []int {
	out := []int{}
	for k := 0; k < 5; k++ {
		out = append(out, personFrst+(id+k*3)%15)
	}
	return out
}

// all ids of one kind, in a stable order
func allMovies() []int {
	out := make([]int, 0, movieMax)
	for i := 1; i <= movieMax; i++ {
		out = append(out, i)
	}
	return out
}

func allShows() []int {
	out := make([]int, 0, showLast-showFirst+1)
	for i := showFirst; i <= showLast; i++ {
		out = append(out, i)
	}
	return out
}
