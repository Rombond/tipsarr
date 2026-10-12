package teasers

import (
	"strconv"
	"testing"

	"github.com/Rombond/tipsarr/backend/internal/media"
)

func card(typ string, id int) Teaser {
	return Teaser{Item: media.Item{Type: typ, TMDBID: id}, TrailerKey: "k" + strconv.Itoa(id)}
}

func cards(typ string, from, n int) []Teaser {
	var out []Teaser
	for i := 0; i < n; i++ {
		out = append(out, card(typ, from+i))
	}
	return out
}

func ids(list []Teaser) []int {
	var out []int
	for _, t := range list {
		out = append(out, t.TMDBID)
	}
	return out
}

// pick always 0: no shuffle (the group stays two pool titles then the recommendation) and always the first title
func firstPick(int) int { return 0 }

func TestComposeGroupsOfThree(t *testing.T) {
	s := &Service{pick: firstPick}
	pool := cards("movie", 100, 10)
	recs := cards("movie", 200, 4)
	got := ids(s.compose(pool, recs, 6))
	// the shuffle with pick = 0 turns [pool, pool, rec] into [pool, rec, pool], so each group has one recommendation
	want := []int{100, 200, 101, 102, 201, 103}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestComposeFallsBackWhenAListRunsDry(t *testing.T) {
	s := &Service{pick: firstPick}
	// no recommendation at all: every slot comes from the pool
	if got := ids(s.compose(cards("movie", 1, 5), nil, 5)); len(got) != 5 {
		t.Fatalf("no recommendations: %v", got)
	}
	// no pool: recommendations fill the groups
	if got := ids(s.compose(nil, cards("movie", 1, 4), 4)); len(got) != 4 {
		t.Fatalf("no pool: %v", got)
	}
	// less than asked: everything, once
	if got := s.compose(cards("movie", 1, 2), cards("movie", 2, 2), 10); len(got) != 3 {
		t.Fatalf("a title in both lists must appear once: %v", ids(got))
	}
	if got := s.compose(nil, nil, 5); got == nil || len(got) != 0 {
		t.Fatalf("empty input must give an empty, non-nil list: %#v", got)
	}
}

func TestComposeDrawsFromTheTopQuarter(t *testing.T) {
	// the highest allowed pick is used: with 8 pool titles the top quarter is the first 2
	s := &Service{pick: func(n int) int { return n - 1 }}
	pool := cards("movie", 1, 8)
	for range 20 {
		got := s.compose(append([]Teaser(nil), pool...), nil, 1)
		if len(got) != 1 || got[0].TMDBID > 2 {
			t.Fatalf("drawn %v, must come from the top quarter (ids 1 and 2)", ids(got))
		}
	}
}

func TestUniqueAndAlternate(t *testing.T) {
	it := func(typ string, id int) media.Item { return media.Item{Type: typ, TMDBID: id} }
	got := unique(alternate([]media.Item{it("movie", 1), it("movie", 2), it("movie", 3)}, []media.Item{it("movie", 2), it("movie", 20)}), []media.Item{it("movie", 1), it("tv", 1)})
	want := []string{"movie:1", "movie:2", "movie:20", "movie:3", "tv:1"}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for i, w := range want {
		if k := got[i].Type + ":" + strconv.Itoa(got[i].TMDBID); k != w {
			t.Fatalf("item %d = %s, want %s", i, k, w)
		}
	}
}
