package store

import (
	"context"
	"sort"
	"strings"
)

// LibraryFilter is what the Library page asks for.
type LibraryFilter struct {
	UserID     string   // whose watched state ("mine") applies
	MediaType  string   // "", movie or tv
	Query      string   // part of the title
	Genres     []string // every one of them must match, or one of them when AnyGenre
	AnyGenre   bool
	YearFrom   int
	YearTo     int
	MinRating  int    // x 10
	MaxRuntime int    // minutes, 0 = any
	Watched    string // "", "yes" or "no" (by UserID)
	Nobody     bool   // watched by nobody at all (admin clean-up view)
	Sort       string // added, title, year, rating, runtime, popular
	Desc       bool
	Offset     int
	Limit      int
}

type LibraryRow struct {
	MediaType  string `bun:"media_type"`
	TMDBID     int64  `bun:"tmdb_id"`
	JellyfinID string `bun:"jellyfin_id"`
	Title      string `bun:"title"`
	Year       int    `bun:"year"`
	RuntimeMin int    `bun:"runtime_min"`
	Rating10   int    `bun:"rating10"`
	AddedAt    int64  `bun:"added_at"`
	Genres     string `bun:"genres"`
	ImageTag   string `bun:"image_tag"`
	Plays      int    `bun:"plays"`
	Viewers    int    `bun:"viewers"`
	MyPlays    int    `bun:"my_plays"`
}

var librarySorts = map[string]string{
	"added": "l.added_at", "title": "LOWER(l.title)", "year": "l.year", "rating": "l.rating10",
	"runtime": "l.runtime_min", "popular": "COALESCE(w.plays, 0)",
}

const watchedJoin = "LEFT JOIN (SELECT media_type, tmdb_id, SUM(play_count) AS plays, COUNT(*) AS viewers FROM watch_history GROUP BY media_type, tmdb_id) AS w ON w.media_type = l.media_type AND w.tmdb_id = l.tmdb_id"

func escapeLike(s string) string {
	return strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(s)
}

// ListLibrary returns one page of the library and the total number of matches.
func (s *Store) ListLibrary(ctx context.Context, f LibraryFilter) ([]LibraryRow, int, error) {
	q := s.DB.NewSelect().TableExpr("library_items AS l").
		ColumnExpr("l.media_type, l.tmdb_id, l.jellyfin_id, l.title, l.year, l.runtime_min, l.rating10, l.added_at, l.genres, l.image_tag").
		ColumnExpr("COALESCE(w.plays, 0) AS plays, COALESCE(w.viewers, 0) AS viewers, COALESCE(me.play_count, 0) AS my_plays").
		Join(watchedJoin).
		Join("LEFT JOIN watch_history AS me ON me.user_id = ? AND me.media_type = l.media_type AND me.tmdb_id = l.tmdb_id", f.UserID)
	if f.MediaType == "movie" || f.MediaType == "tv" {
		q = q.Where("l.media_type = ?", f.MediaType)
	}
	if t := strings.ToLower(strings.TrimSpace(f.Query)); t != "" {
		q = q.Where(`LOWER(l.title) LIKE ? ESCAPE '!'`, "%"+escapeLike(t)+"%")
	}
	var anyOf []string
	var anyArgs []any
	for _, g := range f.Genres {
		if g = strings.TrimSpace(g); g != "" {
			if f.AnyGenre {
				anyOf = append(anyOf, `l.genres LIKE ? ESCAPE '!'`)
				anyArgs = append(anyArgs, "%|"+escapeLike(g)+"|%")
			} else {
				q = q.Where(`l.genres LIKE ? ESCAPE '!'`, "%|"+escapeLike(g)+"|%")
			}
		}
	}
	if len(anyOf) > 0 {
		q = q.Where("("+strings.Join(anyOf, " OR ")+")", anyArgs...)
	}
	if f.YearFrom > 0 {
		q = q.Where("l.year >= ?", f.YearFrom)
	}
	if f.YearTo > 0 {
		q = q.Where("l.year > 0 AND l.year <= ?", f.YearTo)
	}
	if f.MinRating > 0 {
		q = q.Where("l.rating10 >= ?", f.MinRating)
	}
	if f.MaxRuntime > 0 {
		q = q.Where("l.runtime_min > 0 AND l.runtime_min <= ?", f.MaxRuntime)
	}
	switch f.Watched {
	case "yes":
		q = q.Where("COALESCE(me.play_count, 0) > 0")
	case "no":
		q = q.Where("COALESCE(me.play_count, 0) = 0")
	}
	if f.Nobody {
		q = q.Where("COALESCE(w.plays, 0) = 0").
			Where("NOT EXISTS (SELECT 1 FROM watch_events e WHERE e.media_type = l.media_type AND e.tmdb_id = l.tmdb_id AND e.seconds >= 60)")
	}
	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	col, ok := librarySorts[f.Sort]
	if !ok {
		col = librarySorts["added"]
	}
	dir := "ASC"
	if f.Desc {
		dir = "DESC"
	}
	var rows []LibraryRow
	err = q.OrderExpr(col+" "+dir).OrderExpr("LOWER(l.title) ASC").OrderExpr("l.tmdb_id ASC").Offset(int64(f.Offset)).Limit(int64(f.Limit)).Scan(ctx, &rows)
	return rows, int(total), err
}

type LibraryFacets struct {
	Movies, Shows int
	Genres        []GenreCount
	YearMin       int
	YearMax       int
	MaxRuntime    int
}

type GenreCount struct {
	Name  string
	Count int
}

// LibraryFacets summarises the library for the filter bar (genres with counts, year range...).
func (s *Store) LibraryFacets(ctx context.Context) (LibraryFacets, error) {
	var rows []struct {
		MediaType  string `bun:"media_type"`
		Genres     string `bun:"genres"`
		Year       int    `bun:"year"`
		RuntimeMin int    `bun:"runtime_min"`
	}
	if err := s.DB.NewSelect().Table("library_items").Column("media_type", "genres", "year", "runtime_min").Scan(ctx, &rows); err != nil {
		return LibraryFacets{}, err
	}
	var out LibraryFacets
	count := map[string]int{}
	for _, r := range rows {
		if r.MediaType == "movie" {
			out.Movies++
		} else {
			out.Shows++
		}
		for _, g := range strings.Split(strings.Trim(r.Genres, "|"), "|") {
			if g != "" {
				count[g]++
			}
		}
		if r.Year > 0 && (out.YearMin == 0 || r.Year < out.YearMin) {
			out.YearMin = r.Year
		}
		out.YearMax = max(out.YearMax, r.Year)
		out.MaxRuntime = max(out.MaxRuntime, r.RuntimeMin)
	}
	for g, n := range count {
		out.Genres = append(out.Genres, GenreCount{g, n})
	}
	sort.Slice(out.Genres, func(i, j int) bool {
		if out.Genres[i].Count != out.Genres[j].Count {
			return out.Genres[i].Count > out.Genres[j].Count
		}
		return out.Genres[i].Name < out.Genres[j].Name
	})
	return out, nil
}
