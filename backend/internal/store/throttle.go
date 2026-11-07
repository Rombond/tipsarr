package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/uptrace/bun"
)

// LoginThrottle is the failed sign-in counter of one key (a username or a client address).
type LoginThrottle struct {
	bun.BaseModel `bun:"table:login_throttle"`

	Key         string `bun:"throttle_key,pk"`
	Failures    int    `bun:"failures"`
	LockedUntil int64  `bun:"locked_until"` // unix seconds; 0 = not locked
	UpdatedAt   int64  `bun:"updated_at"`   // last failure
}

// GetLoginThrottle returns the counter of a key, or a zero one.
func (s *Store) GetLoginThrottle(ctx context.Context, key string) (*LoginThrottle, error) {
	t := &LoginThrottle{}
	err := s.DB.NewSelect().Model(t).Where("throttle_key = ?", key).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return &LoginThrottle{Key: key}, nil
	}
	return t, err
}

// SaveLoginThrottle writes a counter (insert or update).
func (s *Store) SaveLoginThrottle(ctx context.Context, t *LoginThrottle) error {
	res, err := s.DB.NewUpdate().Model(t).Column("failures", "locked_until", "updated_at").WherePK().Exec(ctx)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return nil
	}
	_, err = s.DB.NewInsert().Model(t).Exec(ctx)
	return err
}

// DeleteLoginThrottle clears a counter.
func (s *Store) DeleteLoginThrottle(ctx context.Context, key string) error {
	_, err := s.DB.NewDelete().Model((*LoginThrottle)(nil)).Where("throttle_key = ?", key).Exec(ctx)
	return err
}
