package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/uptrace/bun"
)

const insertChunk = 200 // keeps parameter counts well under PostgreSQL/MySQL limits

func chunk[T any](rows []T, fn func([]T) error) error {
	for i := 0; i < len(rows); i += insertChunk {
		end := min(i+insertChunk, len(rows))
		if err := fn(rows[i:end]); err != nil {
			return err
		}
	}
	return nil
}

// ReplaceLibrary atomically swaps the stored library snapshot.
func (s *Store) ReplaceLibrary(ctx context.Context, items []LibraryItem, seasons []LibrarySeason) error {
	return s.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewDelete().Model((*LibraryItem)(nil)).Where("1 = 1").Exec(ctx); err != nil {
			return err
		}
		if _, err := tx.NewDelete().Model((*LibrarySeason)(nil)).Where("1 = 1").Exec(ctx); err != nil {
			return err
		}
		if err := chunk(items, func(c []LibraryItem) error {
			_, err := tx.NewInsert().Model(&c).Exec(ctx)
			return err
		}); err != nil {
			return err
		}
		return chunk(seasons, func(c []LibrarySeason) error {
			_, err := tx.NewInsert().Model(&c).Exec(ctx)
			return err
		})
	})
}

// LibraryIDs returns the TMDB ids present in the library for a media type.
func (s *Store) LibraryIDs(ctx context.Context, mediaType string, ids []int) (map[int]bool, error) {
	out := map[int]bool{}
	if len(ids) == 0 {
		return out, nil
	}
	var found []int64
	if err := s.DB.NewSelect().Model((*LibraryItem)(nil)).Column("tmdb_id").
		Where("media_type = ?", mediaType).Where("tmdb_id IN (?)", bun.In(ids)).Scan(ctx, &found); err != nil {
		return nil, err
	}
	for _, id := range found {
		out[int(id)] = true
	}
	return out, nil
}

// LibrarySeasons returns season number -> episode count in the library for a show.
func (s *Store) LibrarySeasons(ctx context.Context, tmdbID int) (map[int]int, error) {
	var rows []LibrarySeason
	if err := s.DB.NewSelect().Model(&rows).Where("tmdb_id = ?", tmdbID).Scan(ctx); err != nil {
		return nil, err
	}
	out := make(map[int]int, len(rows))
	for _, r := range rows {
		out[r.SeasonNumber] = r.EpisodeCount
	}
	return out, nil
}

// LibraryJellyfinToTMDB maps Jellyfin item id -> TMDB id for a media type.
func (s *Store) LibraryJellyfinToTMDB(ctx context.Context, mediaType string) (map[string]int64, error) {
	var rows []LibraryItem
	if err := s.DB.NewSelect().Model(&rows).Where("media_type = ?", mediaType).Scan(ctx); err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, r := range rows {
		out[r.JellyfinID] = r.TMDBID
	}
	return out, nil
}

func (s *Store) LibraryCounts(ctx context.Context) (movies, shows int64, err error) {
	m, err := s.DB.NewSelect().Model((*LibraryItem)(nil)).Where("media_type = ?", "movie").Count(ctx)
	if err != nil {
		return 0, 0, err
	}
	t, err := s.DB.NewSelect().Model((*LibraryItem)(nil)).Where("media_type = ?", "tv").Count(ctx)
	return m, t, err
}

// ReplaceUserHistory swaps one user's watch history and bumps their history version
// only when the content actually changed. It returns whether it changed.
func (s *Store) ReplaceUserHistory(ctx context.Context, userID string, rows []WatchHistory) (bool, error) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].MediaType != rows[j].MediaType {
			return rows[i].MediaType < rows[j].MediaType
		}
		return rows[i].TMDBID < rows[j].TMDBID
	})
	h := sha256.New()
	for _, r := range rows {
		fmt.Fprintf(h, "%s:%d:%d:%d;", r.MediaType, r.TMDBID, r.LastPlayedAt, r.PlayCount)
	}
	hash := hex.EncodeToString(h.Sum(nil))

	changed := false
	err := s.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		st := new(UserHistoryState)
		err := tx.NewSelect().Model(st).Where("user_id = ?", userID).Scan(ctx)
		known := err == nil
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if known && st.Hash == hash {
			return nil
		}
		changed = true
		if _, err := tx.NewDelete().Model((*WatchHistory)(nil)).Where("user_id = ?", userID).Exec(ctx); err != nil {
			return err
		}
		for i := range rows {
			rows[i].UserID = userID
		}
		if err := chunk(rows, func(c []WatchHistory) error {
			_, err := tx.NewInsert().Model(&c).Exec(ctx)
			return err
		}); err != nil {
			return err
		}
		now := time.Now().Unix()
		if known {
			st.Version++
			st.Hash, st.UpdatedAt = hash, now
			_, err = tx.NewUpdate().Model(st).WherePK().Exec(ctx)
		} else {
			_, err = tx.NewInsert().Model(&UserHistoryState{UserID: userID, Version: 1, Hash: hash, UpdatedAt: now}).Exec(ctx)
		}
		return err
	})
	return changed, err
}

// UserHistory returns a user's watch history, most recent first.
func (s *Store) UserHistory(ctx context.Context, userID string, limit int) ([]WatchHistory, error) {
	var rows []WatchHistory
	q := s.DB.NewSelect().Model(&rows).Where("user_id = ?", userID).Order("last_played_at DESC")
	if limit > 0 {
		q = q.Limit(int64(limit))
	}
	return rows, q.Scan(ctx)
}

// HistoryVersion returns 0 when the user has never been synced.
func (s *Store) HistoryVersion(ctx context.Context, userID string) (int64, error) {
	st := new(UserHistoryState)
	err := s.DB.NewSelect().Model(st).Where("user_id = ?", userID).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return st.Version, err
}

func (s *Store) SaveJobRun(ctx context.Context, r *JobRun) error {
	res, err := s.DB.NewUpdate().Model(r).WherePK().Exec(ctx)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return nil
	}
	_, err = s.DB.NewInsert().Model(r).Exec(ctx)
	return err
}

func (s *Store) JobRuns(ctx context.Context) ([]JobRun, error) {
	var rows []JobRun
	return rows, s.DB.NewSelect().Model(&rows).Order("name").Scan(ctx)
}
