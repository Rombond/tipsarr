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
