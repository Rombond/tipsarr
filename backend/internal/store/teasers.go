package store

import (
	"context"
	"time"

	"github.com/uptrace/bun"
)

// TeaserSeen is one title a person swiped past in the Teasers feed.
type TeaserSeen struct {
	bun.BaseModel `bun:"table:teaser_seen"`

	UserID    string `bun:"user_id,pk"`
	MediaType string `bun:"media_type,pk"`
	TMDBID    int64  `bun:"tmdb_id,pk"`
	SeenAt    int64  `bun:"seen_at"`
}

// MarkTeaserSeen records (or refreshes) that a person has seen a title in the feed.
func (s *Store) MarkTeaserSeen(ctx context.Context, userID, mediaType string, tmdbID int64) error {
	m := &TeaserSeen{UserID: userID, MediaType: mediaType, TMDBID: tmdbID, SeenAt: time.Now().Unix()}
	res, err := s.DB.NewUpdate().Model(m).Column("seen_at").WherePK().Exec(ctx)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return nil
	}
	_, err = s.DB.NewInsert().Model(m).Exec(ctx)
	return err
}

// TeaserSeenSet returns the "type:tmdb" keys a person saw since the given unix time.
func (s *Store) TeaserSeenSet(ctx context.Context, userID string, since int64) (map[string]bool, error) {
	var rows []TeaserSeen
	if err := s.DB.NewSelect().Model(&rows).Where("user_id = ?", userID).Where("seen_at >= ?", since).Scan(ctx); err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(rows))
	for _, r := range rows {
		out[WatchKey(r.MediaType, r.TMDBID)] = true
	}
	return out, nil
}

// TeaserSeenPage returns a page of the titles a person swiped past, newest first, and how many there are in all.
func (s *Store) TeaserSeenPage(ctx context.Context, userID string, limit, offset int) ([]TeaserSeen, int, error) {
	var rows []TeaserSeen
	total, err := s.DB.NewSelect().Model(&rows).Where("user_id = ?", userID).
		Order("seen_at DESC", "tmdb_id").Limit(int64(limit)).Offset(int64(offset)).ScanAndCount(ctx)
	return rows, int(total), err
}

// DeleteTeaserSeen makes one title eligible for the feed again; it reports whether there was a row.
func (s *Store) DeleteTeaserSeen(ctx context.Context, userID, mediaType string, tmdbID int64) (bool, error) {
	res, err := s.DB.NewDelete().Model((*TeaserSeen)(nil)).Where("user_id = ?", userID).Where("media_type = ?", mediaType).Where("tmdb_id = ?", tmdbID).Exec(ctx)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// ClearTeaserSeen forgets everything a person swiped past and returns how many titles that was.
func (s *Store) ClearTeaserSeen(ctx context.Context, userID string) (int, error) {
	res, err := s.DB.NewDelete().Model((*TeaserSeen)(nil)).Where("user_id = ?", userID).Exec(ctx)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// TeaserPoolRow is one title of the always-ready pool of a language and region.
type TeaserPoolRow struct {
	bun.BaseModel `bun:"table:teaser_pool"`

	Scope      string `bun:"scope,pk"`
	MediaType  string `bun:"media_type,pk"`
	TMDBID     int64  `bun:"tmdb_id,pk"`
	YoutubeKey string `bun:"youtube_key"`
	SortOrder  int    `bun:"sort_order"`
	Item       string `bun:"item"`
	UpdatedAt  int64  `bun:"updated_at"`
}

// ReplaceTeaserPool swaps the whole pool of one scope for these rows.
func (s *Store) ReplaceTeaserPool(ctx context.Context, scope string, rows []TeaserPoolRow) error {
	return s.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewDelete().Model((*TeaserPoolRow)(nil)).Where("scope = ?", scope).Exec(ctx); err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		_, err := tx.NewInsert().Model(&rows).Exec(ctx)
		return err
	})
}

// TeaserPool returns the pool of a scope in feed order.
func (s *Store) TeaserPool(ctx context.Context, scope string) ([]TeaserPoolRow, error) {
	var rows []TeaserPoolRow
	return rows, s.DB.NewSelect().Model(&rows).Where("scope = ?", scope).Order("sort_order", "tmdb_id").Scan(ctx)
}
