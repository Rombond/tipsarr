package store

import (
	"context"
	"sort"
	"time"

	"github.com/uptrace/bun"
)

// HistorySource says that a person's recommendations also use the watch history of another account (a shared TV
// account, say). Set by an admin.
type HistorySource struct {
	bun.BaseModel `bun:"table:history_sources"`

	UserID       string `bun:"user_id,pk"`
	SourceUserID string `bun:"source_user_id,pk"`
	CreatedAt    int64  `bun:"created_at"`
}

// HistorySources returns the accounts a person borrows history from, sorted.
func (s *Store) HistorySources(ctx context.Context, userID string) ([]string, error) {
	var ids []string
	err := s.DB.NewSelect().Model((*HistorySource)(nil)).Column("source_user_id").Where("user_id = ?", userID).Order("source_user_id").Scan(ctx, &ids)
	return ids, err
}

// SetHistorySources replaces the accounts a person borrows history from.
func (s *Store) SetHistorySources(ctx context.Context, userID string, sources []string) error {
	return s.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewDelete().Model((*HistorySource)(nil)).Where("user_id = ?", userID).Exec(ctx); err != nil {
			return err
		}
		if len(sources) == 0 {
			return nil
		}
		now := time.Now().Unix()
		rows := make([]HistorySource, len(sources))
		for i, id := range sources {
			rows[i] = HistorySource{UserID: userID, SourceUserID: id, CreatedAt: now}
		}
		_, err := tx.NewInsert().Model(&rows).Exec(ctx)
		return err
	})
}

// UsersBorrowingFrom lists the people whose recommendations use this account's history.
func (s *Store) UsersBorrowingFrom(ctx context.Context, sourceID string) ([]string, error) {
	var ids []string
	err := s.DB.NewSelect().Model((*HistorySource)(nil)).Column("user_id").Where("source_user_id = ?", sourceID).Order("user_id").Scan(ctx, &ids)
	return ids, err
}

// MergedHistory is what these accounts watched together: one row per title (the latest play, the plays added up),
// most recent first.
func (s *Store) MergedHistory(ctx context.Context, userIDs []string) ([]WatchHistory, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	var rows []WatchHistory
	if err := s.DB.NewSelect().Model(&rows).Where("user_id IN (?)", bun.In(userIDs)).Scan(ctx); err != nil {
		return nil, err
	}
	merged := map[string]WatchHistory{}
	for _, r := range rows {
		k := WatchKey(r.MediaType, r.TMDBID)
		m := merged[k]
		m.MediaType, m.TMDBID = r.MediaType, r.TMDBID
		m.LastPlayedAt = max(m.LastPlayedAt, r.LastPlayedAt)
		m.PlayCount += r.PlayCount
		merged[k] = m
	}
	out := make([]WatchHistory, 0, len(merged))
	for _, m := range merged {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].LastPlayedAt != out[j].LastPlayedAt {
			return out[i].LastPlayedAt > out[j].LastPlayedAt
		}
		return WatchKey(out[i].MediaType, out[i].TMDBID) < WatchKey(out[j].MediaType, out[j].TMDBID)
	})
	return out, nil
}

// WatchedSetOf is the "type:tmdb" keys watched by any of these accounts.
func (s *Store) WatchedSetOf(ctx context.Context, userIDs []string) (map[string]bool, error) {
	out := map[string]bool{}
	if len(userIDs) == 0 {
		return out, nil
	}
	var rows []WatchHistory
	if err := s.DB.NewSelect().Model(&rows).Column("media_type", "tmdb_id").Where("user_id IN (?)", bun.In(userIDs)).Scan(ctx); err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[WatchKey(r.MediaType, r.TMDBID)] = true
	}
	return out, nil
}

// HasHistoryOf reports whether any of these accounts has watched something.
func (s *Store) HasHistoryOf(ctx context.Context, userIDs []string) (bool, error) {
	if len(userIDs) == 0 {
		return false, nil
	}
	n, err := s.DB.NewSelect().Model((*WatchHistory)(nil)).Where("user_id IN (?)", bun.In(userIDs)).Limit(1).Count(ctx)
	return n > 0, err
}
