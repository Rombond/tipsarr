package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")

// UpsertUser creates the user or refreshes name / last login. Role is promoted to
// admin when asAdmin is true but never demoted automatically.
func (s *Store) UpsertUser(ctx context.Context, id, name string, asAdmin bool) (*User, error) {
	now := time.Now().Unix()
	u := new(User)
	err := s.DB.NewSelect().Model(u).Where("id = ?", id).Scan(ctx)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		u = &User{ID: id, Name: name, Role: RoleUser, CreatedAt: now, LastLoginAt: now}
		if asAdmin {
			u.Role = RoleAdmin
		}
		_, err = s.DB.NewInsert().Model(u).Exec(ctx)
		return u, err
	case err != nil:
		return nil, err
	}
	u.Name, u.LastLoginAt = name, now
	if asAdmin {
		u.Role = RoleAdmin
	}
	_, err = s.DB.NewUpdate().Model(u).Column("name", "role", "last_login_at").WherePK().Exec(ctx)
	return u, err
}

func (s *Store) GetUser(ctx context.Context, id string) (*User, error) {
	u := new(User)
	err := s.DB.NewSelect().Model(u).Where("id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func (s *Store) CreateSession(ctx context.Context, sess *Session) error {
	_, err := s.DB.NewInsert().Model(sess).Exec(ctx)
	return err
}

// SessionUser returns the user owning a live session, or ErrNotFound.
func (s *Store) SessionUser(ctx context.Context, sessionID string) (*User, error) {
	sess := new(Session)
	err := s.DB.NewSelect().Model(sess).Where("id = ?", sessionID).Where("expires_at > ?", time.Now().Unix()).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.GetUser(ctx, sess.UserID)
}

// LiveSession returns an unexpired session, or ErrNotFound.
func (s *Store) LiveSession(ctx context.Context, sessionID string) (*Session, error) {
	sess := new(Session)
	err := s.DB.NewSelect().Model(sess).Where("id = ?", sessionID).Where("expires_at > ?", time.Now().Unix()).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return sess, err
}

// TouchSession records a use of the session and moves its expiry.
func (s *Store) TouchSession(ctx context.Context, sessionID string, lastSeen, expiresAt int64) error {
	_, err := s.DB.NewUpdate().Model((*Session)(nil)).
		Set("last_seen_at = ?", lastSeen).Set("expires_at = ?", expiresAt).Where("id = ?", sessionID).Exec(ctx)
	return err
}

// ListUserSessions returns a user's live sessions, most recently used first.
func (s *Store) ListUserSessions(ctx context.Context, userID string) ([]Session, error) {
	out := []Session{}
	err := s.DB.NewSelect().Model(&out).Where("user_id = ?", userID).Where("expires_at > ?", time.Now().Unix()).
		OrderExpr("last_seen_at DESC").Scan(ctx)
	return out, err
}

// DeleteUserSession ends one of the user's sessions by its public id; ErrNotFound if it is not theirs.
func (s *Store) DeleteUserSession(ctx context.Context, userID, publicID string) error {
	res, err := s.DB.NewDelete().Model((*Session)(nil)).
		Where("user_id = ?", userID).Where("substr(id, 1, 16) = ?", publicID).Exec(ctx)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return s.pruneDevices(ctx)
}

// DeleteOtherUserSessions ends every session of the user except keepID.
func (s *Store) DeleteOtherUserSessions(ctx context.Context, userID, keepID string) error {
	_, err := s.DB.NewDelete().Model((*Session)(nil)).Where("user_id = ?", userID).Where("id <> ?", keepID).Exec(ctx)
	if err != nil {
		return err
	}
	return s.pruneDevices(ctx)
}

func (s *Store) DeleteSession(ctx context.Context, sessionID string) error {
	_, err := s.DB.NewDelete().Model((*Session)(nil)).Where("id = ?", sessionID).Exec(ctx)
	if err != nil {
		return err
	}
	return s.pruneDevices(ctx)
}

func (s *Store) PurgeExpiredSessions(ctx context.Context) error {
	_, err := s.DB.NewDelete().Model((*Session)(nil)).Where("expires_at <= ?", time.Now().Unix()).Exec(ctx)
	if err != nil {
		return err
	}
	return s.pruneDevices(ctx)
}

// GetSetting returns "" when the key is unset.
func (s *Store) GetSetting(ctx context.Context, key string) (string, error) {
	var v string
	err := s.DB.NewSelect().Model((*Setting)(nil)).Column("svalue").Where("skey = ?", key).Scan(ctx, &v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return v, err
}

func (s *Store) SetSetting(ctx context.Context, key, value string) error {
	res, err := s.DB.NewUpdate().Model((*Setting)(nil)).Set("svalue = ?", value).Where("skey = ?", key).Exec(ctx)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return nil
	}
	_, err = s.DB.NewInsert().Model(&Setting{Key: key, Value: value}).Exec(ctx)
	return err
}

// GetCache returns a non-expired cached body.
func (s *Store) GetCache(ctx context.Context, key string) ([]byte, bool, error) {
	var body string
	err := s.DB.NewSelect().Model((*TMDBCache)(nil)).Column("body").
		Where("ckey = ?", key).Where("expires_at > ?", time.Now().Unix()).Scan(ctx, &body)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return []byte(body), true, nil
}

// GetCacheStale returns a cached body even when it has expired. It is the fallback used when
// TMDB cannot be reached, so a TMDB outage shows older data instead of an error.
func (s *Store) GetCacheStale(ctx context.Context, key string) ([]byte, bool, error) {
	var body string
	err := s.DB.NewSelect().Model((*TMDBCache)(nil)).Column("body").Where("ckey = ?", key).Scan(ctx, &body)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return []byte(body), true, nil
}

func (s *Store) PutCache(ctx context.Context, key string, body []byte, ttl time.Duration) error {
	now := time.Now()
	row := &TMDBCache{Key: key, Body: string(body), FetchedAt: now.Unix(), ExpiresAt: now.Add(ttl).Unix()}
	res, err := s.DB.NewUpdate().Model(row).Column("body", "fetched_at", "expires_at").WherePK().Exec(ctx)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return nil
	}
	_, err = s.DB.NewInsert().Model(row).Exec(ctx)
	return err
}

// CacheStaleGrace is how long an expired entry is kept as an offline fallback (see GetCacheStale).
const CacheStaleGrace = 7 * 24 * time.Hour

func (s *Store) PurgeExpiredCache(ctx context.Context) error {
	_, err := s.DB.NewDelete().Model((*TMDBCache)(nil)).Where("expires_at <= ?", time.Now().Add(-CacheStaleGrace).Unix()).Exec(ctx)
	return err
}

// SetSettingIfAbsent stores the value only if the key does not exist yet and reports whether
// it did (the primary key makes the claim atomic).
func (s *Store) SetSettingIfAbsent(ctx context.Context, key, value string) (bool, error) {
	if cur, err := s.GetSetting(ctx, key); err != nil || cur != "" {
		return false, err
	}
	if _, err := s.DB.NewInsert().Model(&Setting{Key: key, Value: value}).Exec(ctx); err != nil {
		if cur, gerr := s.GetSetting(ctx, key); gerr == nil && cur != "" {
			return false, nil // lost the race
		}
		return false, err
	}
	return true, nil
}
