package store

import "context"

// UserRequestStats counts one user's requests by status and media type.
type UserRequestStats struct {
	Total, Movies, Shows                           int
	Pending, Approved, Available, Declined, Failed int
}

func (s *Store) UserRequestStats(ctx context.Context, userID string) (UserRequestStats, error) {
	var rows []struct {
		Status    string `bun:"status"`
		MediaType string `bun:"media_type"`
		N         int    `bun:"n"`
	}
	if err := s.DB.NewSelect().Model((*Request)(nil)).ColumnExpr("status").ColumnExpr("media_type").ColumnExpr("COUNT(*) AS n").
		Where("requested_by = ?", userID).Group("status", "media_type").Scan(ctx, &rows); err != nil {
		return UserRequestStats{}, err
	}
	var st UserRequestStats
	for _, r := range rows {
		st.Total += r.N
		if r.MediaType == "tv" {
			st.Shows += r.N
		} else {
			st.Movies += r.N
		}
		switch r.Status {
		case StatusPending:
			st.Pending += r.N
		case StatusApproved:
			st.Approved += r.N
		case StatusAvailable:
			st.Available += r.N
		case StatusDeclined:
			st.Declined += r.N
		case StatusFailed:
			st.Failed += r.N
		}
	}
	return st, nil
}

// WatchedCount is how many distinct titles a user has watched (per the synced Jellyfin history).
func (s *Store) WatchedCount(ctx context.Context, userID string) (int, error) {
	n, err := s.DB.NewSelect().Model((*WatchHistory)(nil)).Where("user_id = ?", userID).Count(ctx)
	return int(n), err
}
