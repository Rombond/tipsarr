package store

import (
	"context"
	"strconv"

	"github.com/uptrace/bun"
)

// ReplaceSuggestions swaps all suggestion rows (and their items) of one user atomically.
func (s *Store) ReplaceSuggestions(ctx context.Context, userID string, rows []SuggestionRow, items []SuggestionItem) error {
	return s.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var old []string
		if err := tx.NewSelect().Model((*SuggestionRow)(nil)).Column("id").Where("user_id = ?", userID).Scan(ctx, &old); err != nil {
			return err
		}
		if len(old) > 0 {
			if _, err := tx.NewDelete().Model((*SuggestionItem)(nil)).Where("row_id IN (?)", bun.In(old)).Exec(ctx); err != nil {
				return err
			}
			if _, err := tx.NewDelete().Model((*SuggestionRow)(nil)).Where("user_id = ?", userID).Exec(ctx); err != nil {
				return err
			}
		}
		if err := chunk(rows, func(c []SuggestionRow) error {
			_, err := tx.NewInsert().Model(&c).Exec(ctx)
			return err
		}); err != nil {
			return err
		}
		return chunk(items, func(c []SuggestionItem) error {
			_, err := tx.NewInsert().Model(&c).Exec(ctx)
			return err
		})
	})
}

// SuggestionRows returns a user's rows (ordered) and their items grouped by row id (ordered).
func (s *Store) SuggestionRows(ctx context.Context, userID string) ([]SuggestionRow, map[string][]SuggestionItem, error) {
	var rows []SuggestionRow
	if err := s.DB.NewSelect().Model(&rows).Where("user_id = ?", userID).Order("position").Scan(ctx); err != nil {
		return nil, nil, err
	}
	items := map[string][]SuggestionItem{}
	if len(rows) == 0 {
		return rows, items, nil
	}
	ids := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	var all []SuggestionItem
	if err := s.DB.NewSelect().Model(&all).Where("row_id IN (?)", bun.In(ids)).Order("row_id", "pos").Scan(ctx); err != nil {
		return nil, nil, err
	}
	for _, it := range all {
		items[it.RowID] = append(items[it.RowID], it)
	}
	return rows, items, nil
}

// WatchedSet returns "type:tmdb" keys for everything a user has watched.
func (s *Store) WatchedSet(ctx context.Context, userID string) (map[string]bool, error) {
	var rows []WatchHistory
	if err := s.DB.NewSelect().Model(&rows).Column("media_type", "tmdb_id").Where("user_id = ?", userID).Scan(ctx); err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(rows))
	for _, r := range rows {
		out[WatchKey(r.MediaType, r.TMDBID)] = true
	}
	return out, nil
}

func WatchKey(mediaType string, tmdbID int64) string {
	return mediaType + ":" + strconv.FormatInt(tmdbID, 10)
}

// GlobalHistorySeeds returns the titles watched by the most users on the server (ties: most
// recent first). It seeds suggestions for users who have no history of their own.
func (s *Store) GlobalHistorySeeds(ctx context.Context, limit int) ([]WatchHistory, error) {
	var rows []struct {
		MediaType string `bun:"media_type"`
		TMDBID    int64  `bun:"tmdb_id"`
		Last      int64  `bun:"last"`
	}
	err := s.DB.NewSelect().Model((*WatchHistory)(nil)).
		ColumnExpr("media_type").ColumnExpr("tmdb_id").ColumnExpr("MAX(last_played_at) AS last").
		Group("media_type", "tmdb_id").
		OrderExpr("COUNT(DISTINCT user_id) DESC, MAX(last_played_at) DESC").
		Limit(int64(limit)).Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}
	out := make([]WatchHistory, len(rows))
	for i, r := range rows {
		out[i] = WatchHistory{MediaType: r.MediaType, TMDBID: r.TMDBID, LastPlayedAt: r.Last}
	}
	return out, nil
}

// LibraryTitle returns the Jellyfin title of a library item ("" if unknown).
func (s *Store) LibraryTitle(ctx context.Context, mediaType string, tmdbID int64) string {
	var t string
	_ = s.DB.NewSelect().Model((*LibraryItem)(nil)).Column("title").
		Where("media_type = ?", mediaType).Where("tmdb_id = ?", tmdbID).Scan(ctx, &t)
	return t
}
