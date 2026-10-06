package store

import (
	"context"
	"time"
)

// AddMark adds a title to a user's watchlist/blocklist (idempotent).
func (s *Store) AddMark(ctx context.Context, m *UserMark) error {
	m.CreatedAt = time.Now().Unix()
	res, err := s.DB.NewUpdate().Model(m).Column("title", "poster_path", "release_date", "vote_tenths").WherePK().Exec(ctx)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return nil
	}
	_, err = s.DB.NewInsert().Model(m).Exec(ctx)
	return err
}

func (s *Store) RemoveMark(ctx context.Context, userID, kind, mediaType string, tmdbID int64) error {
	_, err := s.DB.NewDelete().Model((*UserMark)(nil)).
		Where("user_id = ?", userID).Where("kind = ?", kind).Where("media_type = ?", mediaType).Where("tmdb_id = ?", tmdbID).Exec(ctx)
	return err
}

// Marks lists a user's marks of one kind, newest first.
func (s *Store) Marks(ctx context.Context, userID, kind string) ([]UserMark, error) {
	var rows []UserMark
	return rows, s.DB.NewSelect().Model(&rows).Where("user_id = ?", userID).Where("kind = ?", kind).Order("created_at DESC", "tmdb_id").Scan(ctx)
}

// MarkSet returns "type:tmdb" keys of a user's marks of one kind.
func (s *Store) MarkSet(ctx context.Context, userID, kind string) (map[string]bool, error) {
	rows, err := s.Marks(ctx, userID, kind)
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(rows))
	for _, r := range rows {
		out[WatchKey(r.MediaType, r.TMDBID)] = true
	}
	return out, nil
}

// HasMark reports which kinds a user has set on a title.
func (s *Store) HasMark(ctx context.Context, userID, mediaType string, tmdbID int64) (watchlisted, blocklisted bool, err error) {
	var kinds []string
	err = s.DB.NewSelect().Model((*UserMark)(nil)).Column("kind").
		Where("user_id = ?", userID).Where("media_type = ?", mediaType).Where("tmdb_id = ?", tmdbID).Scan(ctx, &kinds)
	for _, k := range kinds {
		watchlisted = watchlisted || k == MarkWatchlist
		blocklisted = blocklisted || k == MarkBlocklist
	}
	return
}

// ---- users admin ----------------------------------------------------------------------------

func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	var rows []User
	return rows, s.DB.NewSelect().Model(&rows).Order("name").Scan(ctx)
}

func (s *Store) CountAdmins(ctx context.Context) (int, error) {
	n, err := s.DB.NewSelect().Model((*User)(nil)).Where("role = ?", RoleAdmin).Count(ctx)
	return int(n), err
}

// UpdatePrefs changes only region and language (never the role, which a stale copy could revert).
func (s *Store) UpdatePrefs(ctx context.Context, u *User) error {
	_, err := s.DB.NewUpdate().Model(u).Column("region", "language").WherePK().Exec(ctx)
	return err
}

// DeleteUserSessions signs a user out everywhere (used when their role changes).
func (s *Store) DeleteUserSessions(ctx context.Context, userID string) error {
	_, err := s.DB.NewDelete().Model((*Session)(nil)).Where("user_id = ?", userID).Exec(ctx)
	return err
}

// UpdateUser changes the editable fields of a user.
func (s *Store) UpdateUser(ctx context.Context, u *User) error {
	_, err := s.DB.NewUpdate().Model(u).Column("role", "region", "language").WherePK().Exec(ctx)
	return err
}

// EnsureUser creates a user known from Jellyfin without logging them in (no-op if present;
// the name is refreshed). It never changes the role of an existing user.
func (s *Store) EnsureUser(ctx context.Context, id, name string, admin bool) (created bool, err error) {
	if _, err := s.GetUser(ctx, id); err == nil {
		_, err = s.DB.NewUpdate().Model((*User)(nil)).Set("name = ?", name).Where("id = ?", id).Exec(ctx)
		return false, err
	} else if err != ErrNotFound {
		return false, err
	}
	role := RoleUser
	if admin {
		role = RoleAdmin
	}
	_, err = s.DB.NewInsert().Model(&User{ID: id, Name: name, Role: role, CreatedAt: time.Now().Unix()}).Exec(ctx)
	return err == nil, err
}
