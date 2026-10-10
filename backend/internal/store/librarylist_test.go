package store

import (
	"context"
	"path/filepath"
	"testing"
)

func TestListLibraryGenreMode(t *testing.T) {
	ctx := context.Background()
	s, err := Open(ctx, "sqlite:"+filepath.Join(t.TempDir(), "x.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	items := []LibraryItem{
		{MediaType: "movie", TMDBID: 1, JellyfinID: "a", Title: "Both", Genres: "|Action|Comedy|"},
		{MediaType: "movie", TMDBID: 2, JellyfinID: "b", Title: "Action only", Genres: "|Action|"},
		{MediaType: "movie", TMDBID: 3, JellyfinID: "c", Title: "Drama", Genres: "|Drama|"},
	}
	if err := s.ReplaceLibrary(ctx, items, nil); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		any  bool
		want int
	}{{false, 1}, {true, 2}} {
		_, total, err := s.ListLibrary(ctx, LibraryFilter{Genres: []string{"Action", "Comedy"}, AnyGenre: c.any, Limit: 10})
		if err != nil {
			t.Fatal(err)
		}
		if total != c.want {
			t.Fatalf("any=%v: got %d titles, want %d", c.any, total, c.want)
		}
	}
}
