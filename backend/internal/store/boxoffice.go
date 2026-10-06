package store

import (
	"context"
	"database/sql"
	"errors"

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
