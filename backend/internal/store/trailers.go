package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/uptrace/bun"
)

// Trailer file states.
const (
	TrailerQueued  = "queued"
	TrailerRunning = "running"
	TrailerReady   = "ready"
	TrailerFailed  = "failed"
)

// TrailerFile is the state of one trailer file the resolver sidecar fetches for a YouTube key.
type TrailerFile struct {
	bun.BaseModel `bun:"table:trailer_files"`

	YoutubeKey  string `bun:"youtube_key,pk"`
	Status      string `bun:"status"`
	Bytes       int64  `bun:"bytes"`
	Height      int    `bun:"height"`
	Duration    int    `bun:"duration"`
	Error       string `bun:"error"`
	Permanent   int    `bun:"permanent"` // 1: never retry
	Attempts    int    `bun:"attempts"`
	RequestedAt int64  `bun:"requested_at"`
	ReadyAt     int64  `bun:"ready_at"`
	LastUsedAt  int64  `bun:"last_used_at"`
	NextTryAt   int64  `bun:"next_try_at"` // a temporary failure is retried after this unix time
}

// TrailerFileByKey returns nil when the key has no row.
func (s *Store) TrailerFileByKey(ctx context.Context, key string) (*TrailerFile, error) {
	f := new(TrailerFile)
	err := s.DB.NewSelect().Model(f).Where("youtube_key = ?", key).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return f, err
}

// AddTrailerFile inserts a queued row; it reports false (and changes nothing) when the key already has one.
func (s *Store) AddTrailerFile(ctx context.Context, key string, now int64) (bool, error) {
	f := &TrailerFile{YoutubeKey: key, Status: TrailerQueued, RequestedAt: now}
	if _, err := s.DB.NewInsert().Model(f).Exec(ctx); err != nil {
		// a concurrent request may have inserted the same key first
		if ex, rerr := s.TrailerFileByKey(ctx, key); rerr == nil && ex != nil {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// SaveTrailerFile writes every field of an existing row.
func (s *Store) SaveTrailerFile(ctx context.Context, f *TrailerFile) error {
	_, err := s.DB.NewUpdate().Model(f).WherePK().Exec(ctx)
	return err
}

// TrailerFilesByStatus lists rows in any of the given states, oldest request first.
func (s *Store) TrailerFilesByStatus(ctx context.Context, limit int, statuses ...string) ([]TrailerFile, error) {
	var rows []TrailerFile
	q := s.DB.NewSelect().Model(&rows).Where("status IN (?)", bun.In(statuses)).Order("requested_at", "youtube_key")
	if limit > 0 {
		q = q.Limit(int64(limit))
	}
	return rows, q.Scan(ctx)
}

// TrailerFilesToRetry lists temporary failures whose back-off is over and that were asked for since the given time.
func (s *Store) TrailerFilesToRetry(ctx context.Context, now, askedSince int64, limit int) ([]TrailerFile, error) {
	var rows []TrailerFile
	return rows, s.DB.NewSelect().Model(&rows).
		Where("status = ?", TrailerFailed).Where("permanent = 0").Where("next_try_at <= ?", now).Where("requested_at >= ?", askedSince).
		Order("next_try_at", "youtube_key").Limit(int64(limit)).Scan(ctx)
}

// TouchTrailerFile records that a person played the file.
func (s *Store) TouchTrailerFile(ctx context.Context, key string, now int64) error {
	_, err := s.DB.NewUpdate().Model((*TrailerFile)(nil)).Set("last_used_at = ?", now).Where("youtube_key = ?", key).Exec(ctx)
	return err
}

// RecentTrailerKeys lists the files played since the given time.
func (s *Store) RecentTrailerKeys(ctx context.Context, since int64, limit int) ([]string, error) {
	var keys []string
	return keys, s.DB.NewSelect().Model((*TrailerFile)(nil)).Column("youtube_key").
		Where("last_used_at >= ?", since).Order("last_used_at DESC").Limit(int64(limit)).Scan(ctx, &keys)
}

// TrailerCounts counts rows per state.
func (s *Store) TrailerCounts(ctx context.Context) (map[string]int, error) {
	var rows []struct {
		Status string `bun:"status"`
		N      int    `bun:"n"`
	}
	err := s.DB.NewSelect().Model((*TrailerFile)(nil)).Column("status").ColumnExpr("COUNT(*) AS n").Group("status").Scan(ctx, &rows)
	out := map[string]int{}
	for _, r := range rows {
		out[r.Status] = r.N
	}
	return out, err
}

// DeleteTrailerFiles forgets every row (the cache was cleared).
func (s *Store) DeleteTrailerFiles(ctx context.Context) error {
	_, err := s.DB.NewDelete().Model((*TrailerFile)(nil)).Where("1 = 1").Exec(ctx)
	return err
}
