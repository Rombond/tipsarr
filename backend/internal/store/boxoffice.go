package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/uptrace/bun"
)

// SaveBoxOffice replaces one stored chart (week header + entries).
func (s *Store) SaveBoxOffice(ctx context.Context, week BoxOfficeWeek, entries []BoxOfficeEntry) error {
	return s.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewDelete().Model((*BoxOfficeEntry)(nil)).Where("region = ?", week.Region).Where("week_key = ?", week.WeekKey).Exec(ctx); err != nil {
			return err
		}
		if _, err := tx.NewDelete().Model((*BoxOfficeWeek)(nil)).Where("region = ?", week.Region).Where("week_key = ?", week.WeekKey).Exec(ctx); err != nil {
			return err
		}
		if _, err := tx.NewInsert().Model(&week).Exec(ctx); err != nil {
			return err
		}
		return chunk(entries, func(c []BoxOfficeEntry) error {
			_, err := tx.NewInsert().Model(&c).Exec(ctx)
			return err
		})
	})
}

// BoxOfficeWeeks lists stored weeks of a region, newest first.
func (s *Store) BoxOfficeWeeks(ctx context.Context, region string) ([]BoxOfficeWeek, error) {
	var rows []BoxOfficeWeek
	return rows, s.DB.NewSelect().Model(&rows).Where("region = ?", region).Order("week_key DESC").Scan(ctx)
}

// BoxOffice returns one stored chart; ErrNotFound if the week is not stored.
func (s *Store) BoxOffice(ctx context.Context, region, weekKey string) (*BoxOfficeWeek, []BoxOfficeEntry, error) {
	w := new(BoxOfficeWeek)
	err := s.DB.NewSelect().Model(w).Where("region = ?", region).Where("week_key = ?", weekKey).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	var entries []BoxOfficeEntry
	err = s.DB.NewSelect().Model(&entries).Where("region = ?", region).Where("week_key = ?", weekKey).Order("pos").Scan(ctx)
	return w, entries, err
}

// AliasKey normalises a chart title for alias lookups.
func AliasKey(title string) string { return strings.ToLower(strings.TrimSpace(title)) }

// BoxOfficeAlias returns the pinned TMDB id for a title, or 0.
func (s *Store) BoxOfficeAlias(ctx context.Context, title string) (int64, error) {
	a := new(BoxOfficeAlias)
	err := s.DB.NewSelect().Model(a).Where("title_key = ?", AliasKey(title)).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return a.TMDBID, err
}

func (s *Store) SetBoxOfficeAlias(ctx context.Context, title string, tmdbID int64) error {
	a := &BoxOfficeAlias{TitleKey: AliasKey(title), TMDBID: tmdbID, CreatedAt: time.Now().Unix()}
	res, err := s.DB.NewUpdate().Model(a).Column("tmdb_id").WherePK().Exec(ctx)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return nil
	}
	_, err = s.DB.NewInsert().Model(a).Exec(ctx)
	return err
}

func (s *Store) DeleteBoxOfficeAlias(ctx context.Context, title string) error {
	_, err := s.DB.NewDelete().Model((*BoxOfficeAlias)(nil)).Where("title_key = ?", AliasKey(title)).Exec(ctx)
	return err
}

// SetBoxOfficeMatch updates the TMDB match (and snapshot) of every stored entry with this title.
func (s *Store) SetBoxOfficeMatch(ctx context.Context, title string, e BoxOfficeEntry) error {
	_, err := s.DB.NewUpdate().Model((*BoxOfficeEntry)(nil)).
		Set("tmdb_id = ?", e.TMDBID).Set("poster_path = ?", e.PosterPath).Set("release_date = ?", e.ReleaseDate).
		Set("vote_tenths = ?", e.VoteTenths).Set("overview = ?", e.Overview).
		Where("LOWER(title) = ?", AliasKey(title)).Exec(ctx)
	return err
}
