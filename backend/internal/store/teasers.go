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
