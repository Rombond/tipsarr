package store

import (
	"context"
	"strconv"

	"github.com/Rombond/tipsarr/backend/internal/perms"
)

// SettingDefaultPermissions is the mask new people start with (an integer); empty = perms.Default.
const SettingDefaultPermissions = "permissions.default"

// Request quotas apply to people without a quota of their own: how many requests (movies) or seasons (shows)
// in how many days. A limit of 0 or no days means no limit.
const (
	SettingQuotaMovieLimit = "quota.movie.limit"
	SettingQuotaMovieDays  = "quota.movie.days"
	SettingQuotaTVLimit    = "quota.tv.limit"
	SettingQuotaTVDays     = "quota.tv.days"
)

// DefaultPermissions is the mask given to a new person.
func (s *Store) DefaultPermissions(ctx context.Context) perms.Mask {
	v, _ := s.GetSetting(ctx, SettingDefaultPermissions)
	if n, err := strconv.ParseInt(v, 10, 64); err == nil && v != "" {
		return perms.Stored(perms.Mask(n))
	}
	return perms.Default
}

// SetDefaultPermissions saves the mask new people start with.
func (s *Store) SetDefaultPermissions(ctx context.Context, m perms.Mask) error {
	return s.SetSetting(ctx, SettingDefaultPermissions, strconv.FormatInt(int64(perms.Stored(m)), 10))
}

// SettingInt reads an integer setting, def when it is missing or not a number.
func (s *Store) SettingInt(ctx context.Context, key string, def int) int {
	v, _ := s.GetSetting(ctx, key)
	if n, err := strconv.Atoi(v); err == nil && v != "" {
		return n
	}
	return def
}

// QuotaUsage counts what a person requested since the given time: requests for movies, seasons for shows.
// Declined and failed requests give their share back. oldest is the creation time of the oldest counted request.
func (s *Store) QuotaUsage(ctx context.Context, userID, mediaType string, since int64) (used int, oldest int64, err error) {
	var row struct {
		N      int
		Oldest int64
	}
	q := `SELECT COUNT(*) AS n, COALESCE(MIN(r.created_at), 0) AS oldest FROM requests r WHERE r.requested_by = ? AND r.media_type = ? AND r.created_at >= ? AND r.status NOT IN ('declined', 'failed')`
	if mediaType == "tv" {
		q = `SELECT COUNT(*) AS n, COALESCE(MIN(r.created_at), 0) AS oldest FROM request_seasons rs JOIN requests r ON r.id = rs.request_id WHERE r.requested_by = ? AND r.media_type = ? AND r.created_at >= ? AND r.status NOT IN ('declined', 'failed')`
	}
	if err := s.DB.NewRaw(q, userID, mediaType, since).Scan(ctx, &row); err != nil {
		return 0, 0, err
	}
	return row.N, row.Oldest, nil
}
