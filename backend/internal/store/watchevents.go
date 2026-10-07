package store

import (
	"context"
)

// MaxWatchEventRow is the highest plugin rowid already copied (0 when none).
func (s *Store) MaxWatchEventRow(ctx context.Context) (int64, error) {
	var n int64
	err := s.DB.NewSelect().Model((*WatchEvent)(nil)).ColumnExpr("COALESCE(MAX(source_rowid), 0)").Scan(ctx, &n)
	return n, err
}

func (s *Store) InsertWatchEvents(ctx context.Context, rows []WatchEvent) error {
	return chunk(rows, func(c []WatchEvent) error {
		_, err := s.DB.NewInsert().Model(&c).Ignore().Exec(ctx)
		return err
	})
}

func (s *Store) DeleteAllWatchEvents(ctx context.Context) error {
	_, err := s.DB.NewDelete().Model((*WatchEvent)(nil)).Where("1 = 1").Exec(ctx)
	return err
}

func (s *Store) CountWatchEvents(ctx context.Context) (int, error) {
	n, err := s.DB.NewSelect().Model((*WatchEvent)(nil)).Count(ctx)
	return int(n), err
}

// WatchEvents returns the plays of one user ("" = everyone) since a unix time.
func (s *Store) WatchEvents(ctx context.Context, userID string, since int64) ([]WatchEvent, error) {
	var rows []WatchEvent
	q := s.DB.NewSelect().Model(&rows).Where("played_at >= ?", since)
	if userID != "" {
		q = q.Where("user_id = ?", userID)
	}
	return rows, q.Scan(ctx)
}

// AllLibrary returns every library row (the stats need genres, year, runtime and the poster tag).
func (s *Store) AllLibrary(ctx context.Context) ([]LibraryItem, error) {
	var rows []LibraryItem
	return rows, s.DB.NewSelect().Model(&rows).Scan(ctx)
}

// WatchHistoryOf returns the synced watch history of one user ("" = everyone).
func (s *Store) WatchHistoryOf(ctx context.Context, userID string) ([]WatchHistory, error) {
	var rows []WatchHistory
	q := s.DB.NewSelect().Model(&rows)
	if userID != "" {
		q = q.Where("user_id = ?", userID)
	}
	return rows, q.Scan(ctx)
}

// RequestsByStatus counts requests per status made since a unix time, by one user ("" = everyone).
func (s *Store) RequestsByStatus(ctx context.Context, userID string, since int64) (map[string]int, error) {
	var rows []struct {
		Status string `bun:"status"`
		N      int    `bun:"n"`
	}
	q := s.DB.NewSelect().Model((*Request)(nil)).ColumnExpr("status").ColumnExpr("COUNT(*) AS n").Where("created_at >= ?", since)
	if userID != "" {
		q = q.Where("requested_by = ?", userID)
	}
	if err := q.Group("status").Scan(ctx, &rows); err != nil {
		return nil, err
	}
	out := map[string]int{}
	for _, r := range rows {
		out[r.Status] = r.N
	}
	return out, nil
}

// UnknownPlayedTitles lists (type, title) of plays that are not linked to a TMDB title and have not
// been looked up since `retryBefore` (unix); at most `limit`, most played first.
func (s *Store) UnknownPlayedTitles(ctx context.Context, retryBefore int64, limit int) ([][2]string, error) {
	var rows []struct {
		MediaType string `bun:"media_type"`
		Title     string `bun:"title"`
	}
	err := s.DB.NewSelect().TableExpr("watch_events AS e").ColumnExpr("e.media_type, e.title").
		Join("LEFT JOIN watch_titles AS w ON w.media_type = e.media_type AND w.title = e.title").
		Where("e.tmdb_id = 0").Where("e.seconds >= 60").Where("LENGTH(e.title) <= 191").
		Where("w.title IS NULL OR (w.tmdb_id = 0 AND w.checked_at < ?)", retryBefore).
		GroupExpr("e.media_type, e.title").OrderExpr("SUM(e.seconds) DESC").Limit(int64(limit)).Scan(ctx, &rows)
	out := make([][2]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, [2]string{r.MediaType, r.Title})
	}
	return out, err
}

// SaveWatchTitle stores the outcome of a lookup (a zero TMDBID records "no match").
func (s *Store) SaveWatchTitle(ctx context.Context, w *WatchTitle) error {
	res, err := s.DB.NewUpdate().Model(w).Column("tmdb_id", "poster", "year", "rating10", "genres", "checked_at").WherePK().Exec(ctx)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return nil
	}
	_, err = s.DB.NewInsert().Model(w).Exec(ctx)
	return err
}

// WatchTitles returns every stored lookup keyed by type + "\x00" + title.
func (s *Store) WatchTitles(ctx context.Context) (map[string]WatchTitle, error) {
	var rows []WatchTitle
	if err := s.DB.NewSelect().Model(&rows).Where("tmdb_id > 0").Scan(ctx); err != nil {
		return nil, err
	}
	out := make(map[string]WatchTitle, len(rows))
	for _, r := range rows {
		out[r.MediaType+"\x00"+r.Title] = r
	}
	return out, nil
}
