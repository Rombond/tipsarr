package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"github.com/uptrace/bun"
)

// NewID returns a random 128-bit hex id.
func NewID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ---- servarr instances ----------------------------------------------------------------

func (s *Store) ListServarr(ctx context.Context) ([]ServarrInstance, error) {
	var rows []ServarrInstance
	return rows, s.DB.NewSelect().Model(&rows).Order("kind", "created_at").Scan(ctx)
}

func (s *Store) GetServarr(ctx context.Context, id string) (*ServarrInstance, error) {
	in := new(ServarrInstance)
	err := s.DB.NewSelect().Model(in).Where("id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return in, err
}

// DefaultServarr returns the default instance of a kind, or ErrNotFound.
func (s *Store) DefaultServarr(ctx context.Context, kind string) (*ServarrInstance, error) {
	in := new(ServarrInstance)
	err := s.DB.NewSelect().Model(in).Where("kind = ?", kind).Order("is_default DESC", "created_at").Limit(1).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return in, err
}

// SaveServarr inserts or updates an instance; when it is the default of its kind the others are cleared.
func (s *Store) SaveServarr(ctx context.Context, in *ServarrInstance) error {
	return s.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		existing := new(ServarrInstance)
		err := tx.NewSelect().Model(existing).Where("id = ?", in.ID).Scan(ctx)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			n, err := tx.NewSelect().Model((*ServarrInstance)(nil)).Where("kind = ?", in.Kind).Count(ctx)
			if err != nil {
				return err
			}
			if n == 0 {
				in.IsDefault = 1 // first instance of a kind is the default
			}
			in.CreatedAt = time.Now().Unix()
			if _, err := tx.NewInsert().Model(in).Exec(ctx); err != nil {
				return err
			}
		case err != nil:
			return err
		default:
			in.CreatedAt = existing.CreatedAt
			if _, err := tx.NewUpdate().Model(in).WherePK().Exec(ctx); err != nil {
				return err
			}
		}
		if in.IsDefault == 1 {
			_, err = tx.NewUpdate().Model((*ServarrInstance)(nil)).Set("is_default = 0").
				Where("kind = ?", in.Kind).Where("id <> ?", in.ID).Exec(ctx)
			return err
		}
		return nil
	})
}

func (s *Store) DeleteServarr(ctx context.Context, id string) error {
	_, err := s.DB.NewDelete().Model((*ServarrInstance)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

// ---- requests -----------------------------------------------------------------------------

// RequestFilter narrows ListRequests. Empty fields are ignored.
type RequestFilter struct {
	RequestedBy string
	Statuses    []string
	MediaType   string
	TMDBID      int64
	// Unwatched keeps what is available but its requester never watched (admin clean-up view).
	Unwatched  bool
	Take, Skip int
}

// unwatchedByRequester: the requester has no play of the title, neither in the synced history nor
// in the plays copied from Playback Reporting. Imported requests have no requester and never match.
const unwatchedByRequester = `?TableAlias.status = 'available' AND ?TableAlias.requested_by <> '' ` +
	`AND NOT EXISTS (SELECT 1 FROM watch_history h WHERE h.user_id = ?TableAlias.requested_by AND h.media_type = ?TableAlias.media_type AND h.tmdb_id = ?TableAlias.tmdb_id) ` +
	`AND NOT EXISTS (SELECT 1 FROM watch_events e WHERE e.user_id = ?TableAlias.requested_by AND e.media_type = ?TableAlias.media_type AND e.tmdb_id = ?TableAlias.tmdb_id AND e.seconds >= 60)`

// CountUnwatchedRequests counts what the "not watched" view lists.
func (s *Store) CountUnwatchedRequests(ctx context.Context) (int, error) {
	n, err := s.DB.NewSelect().Model((*Request)(nil)).Where(unwatchedByRequester).Count(ctx)
	return int(n), err
}

func (s *Store) CreateRequest(ctx context.Context, r *Request, seasons []int) error {
	now := time.Now().Unix()
	r.CreatedAt, r.UpdatedAt = now, now
	return s.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(r).Exec(ctx); err != nil {
			return err
		}
		if len(seasons) == 0 {
			return nil
		}
		rows := make([]RequestSeason, len(seasons))
		for i, n := range seasons {
			rows[i] = RequestSeason{RequestID: r.ID, SeasonNumber: n}
		}
		_, err := tx.NewInsert().Model(&rows).Exec(ctx)
		return err
	})
}

func (s *Store) GetRequest(ctx context.Context, id string) (*Request, error) {
	r := new(Request)
	err := s.DB.NewSelect().Model(r).Where("id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return r, err
}

func (s *Store) UpdateRequest(ctx context.Context, r *Request) error {
	r.UpdatedAt = time.Now().Unix()
	_, err := s.DB.NewUpdate().Model(r).WherePK().Exec(ctx)
	return err
}

func (s *Store) DeleteRequest(ctx context.Context, id string) error {
	return s.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewDelete().Model((*RequestSeason)(nil)).Where("request_id = ?", id).Exec(ctx); err != nil {
			return err
		}
		_, err := tx.NewDelete().Model((*Request)(nil)).Where("id = ?", id).Exec(ctx)
		return err
	})
}

func (s *Store) ListRequests(ctx context.Context, f RequestFilter) ([]Request, int, error) {
	var rows []Request
	q := s.DB.NewSelect().Model(&rows).Order("created_at DESC")
	if f.RequestedBy != "" {
		q = q.Where("requested_by = ?", f.RequestedBy)
	}
	if len(f.Statuses) > 0 {
		q = q.Where("status IN (?)", bun.In(f.Statuses))
	}
	if f.Unwatched {
		q = q.Where(unwatchedByRequester)
	}
	if f.MediaType != "" {
		q = q.Where("media_type = ?", f.MediaType)
	}
	if f.TMDBID != 0 {
		q = q.Where("tmdb_id = ?", f.TMDBID)
	}
	if f.Take > 0 {
		q = q.Limit(int64(f.Take)).Offset(int64(f.Skip))
	}
	total, err := q.ScanAndCount(ctx)
	return rows, int(total), err
}

func (s *Store) RequestSeasons(ctx context.Context, ids []string) (map[string][]int, error) {
	out := map[string][]int{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []RequestSeason
	if err := s.DB.NewSelect().Model(&rows).Where("request_id IN (?)", bun.In(ids)).Order("season_number").Scan(ctx); err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.RequestID] = append(out[r.RequestID], r.SeasonNumber)
	}
	return out, nil
}

// RequestCounts returns the number of requests per status (optionally for one user).
func (s *Store) RequestCounts(ctx context.Context, userID string) (map[string]int, error) {
	var rows []struct {
		Status string `bun:"status"`
		N      int    `bun:"n"`
	}
	q := s.DB.NewSelect().Model((*Request)(nil)).ColumnExpr("status").ColumnExpr("COUNT(*) AS n").Group("status")
	if userID != "" {
		q = q.Where("requested_by = ?", userID)
	}
	if err := q.Scan(ctx, &rows); err != nil {
		return nil, err
	}
	out := map[string]int{}
	for _, r := range rows {
		out[r.Status] = r.N
	}
	return out, nil
}

// InFlightRequests are approved requests that were really sent to Radarr/Sonarr.
func (s *Store) InFlightRequests(ctx context.Context) ([]Request, error) {
	var rows []Request
	return rows, s.DB.NewSelect().Model(&rows).Where("status = ?", StatusApproved).Where("sent_at > 0").Scan(ctx)
}

// UserNames maps user ids to display names.
func (s *Store) UserNames(ctx context.Context, ids []string) (map[string]string, error) {
	out := map[string]string{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []User
	if err := s.DB.NewSelect().Model(&rows).Column("id", "name").Where("id IN (?)", bun.In(ids)).Scan(ctx); err != nil {
		return nil, err
	}
	for _, u := range rows {
		out[u.ID] = u.Name
	}
	return out, nil
}

// ---- webhooks -------------------------------------------------------------------------------

func (s *Store) ListWebhooks(ctx context.Context) ([]Webhook, error) {
	var rows []Webhook
	return rows, s.DB.NewSelect().Model(&rows).Order("created_at").Scan(ctx)
}

func (s *Store) GetWebhook(ctx context.Context, id string) (*Webhook, error) {
	w := new(Webhook)
	err := s.DB.NewSelect().Model(w).Where("id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return w, err
}

func (s *Store) SaveWebhook(ctx context.Context, w *Webhook) error {
	res, err := s.DB.NewUpdate().Model(w).Column("name", "url", "secret", "events", "enabled").WherePK().Exec(ctx)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return nil
	}
	w.CreatedAt = time.Now().Unix()
	_, err = s.DB.NewInsert().Model(w).Exec(ctx)
	return err
}

func (s *Store) DeleteWebhook(ctx context.Context, id string) error {
	_, err := s.DB.NewDelete().Model((*Webhook)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

// ActiveRequestStatuses maps TMDB id -> "pending" | "approved" for titles with an active request.
func (s *Store) ActiveRequestStatuses(ctx context.Context, mediaType string, ids []int) (map[int]string, error) {
	out := map[int]string{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []Request
	if err := s.DB.NewSelect().Model(&rows).Column("tmdb_id", "status").
		Where("media_type = ?", mediaType).Where("tmdb_id IN (?)", bun.In(ids)).
		Where("status IN (?)", bun.In([]string{StatusPending, StatusApproved})).Scan(ctx); err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[int(r.TMDBID)] = r.Status
	}
	return out, nil
}
