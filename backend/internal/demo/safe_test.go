package demo

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSafeTMDBFiltersMatureTitles(t *testing.T) {
	in := `{"page":1,"results":[
		{"id":10545,"name":"True Blood","first_air_date":"2008-09-07","genre_ids":[18]},
		{"id":76479,"media_type":"tv","name":"The Boys"},
		{"id":86423,"name":"Locke & Key","original_name":"Locke & Key"},
		{"id":713704,"media_type":"movie","title":"Evil Dead Rise"},
		{"id":1,"title":"Some horror","genre_ids":[35,27]},
		{"id":2,"title":"Adult","adult":true},
		{"id":862,"title":"Toy Story","genre_ids":[16,10751]},
		{"id":10545,"media_type":"movie","title":"Same id, a movie"},
		{"id":7,"media_type":"person","name":"Someone","known_for":[{"id":1399,"media_type":"tv","name":"Game of Thrones"},{"id":11,"media_type":"movie","title":"Star Wars"}]}
	],"total_results":9}`
	out := string(SafeTMDB([]byte(in)))
	for _, bad := range []string{"True Blood", "The Boys", "Locke", "Evil Dead", "Some horror", "Adult", "Game of Thrones"} {
		if strings.Contains(out, bad) {
			t.Errorf("%q kept: %s", bad, out)
		}
	}
	for _, good := range []string{"Toy Story", "Same id, a movie", "Someone", "Star Wars", `"total_results":9`} {
		if !strings.Contains(out, good) {
			t.Errorf("%q dropped: %s", good, out)
		}
	}
}

func TestSafeTMDBNested(t *testing.T) {
	in := `{"id":1399,"name":"x","recommendations":{"results":[{"id":10545,"name":"True Blood","first_air_date":"x"},{"id":246,"name":"Avatar","first_air_date":"x"}]},
		"credits":{"cast":[{"id":1399,"name":"Actor","original_name":"Actor","known_for_department":"Acting"}]}}`
	out := string(SafeTMDB([]byte(in)))
	if strings.Contains(out, "True Blood") || !strings.Contains(out, "Avatar") || !strings.Contains(out, "Actor") {
		t.Fatal(out)
	}
	if !strings.Contains(out, `"name":"x"`) { // the detail itself is not a list entry
		t.Fatal(out)
	}
}

func TestSafeTMDBUnchanged(t *testing.T) {
	for _, in := range []string{`{"results":[{"id":862,"title":"Toy Story"}]}`, "not json", "", `<html>`} {
		if got := string(SafeTMDB([]byte(in))); got != in {
			t.Errorf("%q rewritten to %q", in, got)
		}
	}
}

func TestProxyAppliesFilter(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"id":10545,"name":"True Blood","first_air_date":"x"},{"id":246,"name":"Avatar","first_air_date":"x"}]}`))
	}))
	defer up.Close()
	for _, filter := range []bool{true, false} {
		u := Upstream{Prefix: "/tmdb/", Base: up.URL, TTL: time.Hour}
		if filter {
			u.Filter = SafeTMDB
		}
		p := NewProxy(t.TempDir(), u)
		rec := httptest.NewRecorder()
		p.ServeHTTP(rec, httptest.NewRequest("GET", "/tmdb/3/trending/all/week", nil))
		if got := strings.Contains(rec.Body.String(), "True Blood"); got == filter {
			t.Errorf("filter=%v: %s", filter, rec.Body.String())
		}
	}
}

func TestCatalogHasNoDeniedTitles(t *testing.T) {
	for _, m := range demoMovies {
		if unsafeMovies[m.tmdb] {
			t.Errorf("library movie %s is denied", m.name)
		}
	}
	for _, s := range demoShows {
		if unsafeShows[s.tmdb] {
			t.Errorf("library show %s is denied", s.name)
		}
	}
	c := Catalog(time.Now())
	known := map[int]bool{}
	for _, ti := range c.Titles {
		known[ti.TMDB] = true
	}
	for u, h := range c.History {
		for _, id := range append(append([]int{}, h.Movies...), h.Shows...) {
			if !known[id] {
				t.Errorf("%s history has %d, not in the library", u, id)
			}
		}
	}
}
