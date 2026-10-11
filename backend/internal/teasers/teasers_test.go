package teasers

import (
	"testing"

	"github.com/Rombond/tipsarr/backend/internal/media"
)

func TestInterleave(t *testing.T) {
	it := func(typ string, id int) media.Item { return media.Item{Type: typ, TMDBID: id} }
	got := interleave(
		[]media.Item{it("movie", 1), it("movie", 2), it("movie", 3)},
		[]media.Item{it("movie", 2), it("tv", 10)},
		[]media.Item{it("movie", 20)},
	)
	// two personal, one trending, one upcoming per round; movie 2 appears once
	want := []string{"movie:1", "movie:2", "tv:10", "movie:20", "movie:3"}
	if len(got) != len(want) {
		t.Fatalf("got %d items: %+v", len(got), got)
	}
	for i, w := range want {
		if k := got[i].Type + ":" + itoa(got[i].TMDBID); k != w {
			t.Fatalf("item %d = %s, want %s", i, k, w)
		}
	}
}

func itoa(n int) string {
	return map[int]string{1: "1", 2: "2", 3: "3", 10: "10", 20: "20"}[n]
}
