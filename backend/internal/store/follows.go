package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/uptrace/bun"
)

// Follow is a request followed from a phone: the server pushes its stage and progress to a Live Activity (iOS)
// or an ongoing notification (Android) until it reaches a final state.
type Follow struct {
	bun.BaseModel `bun:"table:request_follows"`

	ID          string `bun:"id,pk"`
	RequestID   string `bun:"request_id"`
	UserID      string `bun:"user_id"`
	SessionID   string `bun:"session_id"`
	Platform    string `bun:"platform"`   // ios or android
	PushToken   string `bun:"push_token"` // Live Activity token (iOS) or FCM token (Android)
	Sandbox     bool   `bun:"sandbox"`
	Server      string `bun:"server"`
	ActivityID  string `bun:"activity_id"` // iOS: ActivityKit id; a new one restarts the follow
	LastStage   string `bun:"last_stage"`  // last state pushed (throttling)
	LastPercent int    `bun:"last_percent"`
	LastSentAt  int64  `bun:"last_sent_at"`
	CreatedAt   int64  `bun:"created_at"`
	ExpiresAt   int64  `bun:"expires_at"`
}

// SaveFollow creates or refreshes the follow of (session, request). On a refresh the token, server and sandbox
// are updated; when restart is set (a new Live Activity, or Android), created_at, expires_at and the throttle
// state are taken from f too. created reports whether a new row was inserted.
func (s *Store) SaveFollow(ctx context.Context, f *Follow, restart func(old *Follow) bool) (created bool, err error) {
	err = s.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		old := new(Follow)
		err := tx.NewSelect().Model(old).Where("session_id = ?", f.SessionID).Where("request_id = ?", f.RequestID).Scan(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			if f.ID == "" {
				f.ID = NewID()
			}
			created = true
			_, err = tx.NewInsert().Model(f).Exec(ctx)
			return err
		}
		if err != nil {
			return err
		}
		f.ID = old.ID
		if !restart(old) {
			f.CreatedAt, f.ExpiresAt = old.CreatedAt, old.ExpiresAt
			f.LastStage, f.LastPercent, f.LastSentAt = old.LastStage, old.LastPercent, old.LastSentAt
		}
		_, err = tx.NewUpdate().Model(f).WherePK().Exec(ctx)
		return err
	})
	return created, err
}

func (s *Store) DeleteFollow(ctx context.Context, sessionID, requestID string) error {
	_, err := s.DB.NewDelete().Model((*Follow)(nil)).Where("session_id = ?", sessionID).Where("request_id = ?", requestID).Exec(ctx)
	return err
}

func (s *Store) DeleteFollowByID(ctx context.Context, id string) error {
	_, err := s.DB.NewDelete().Model((*Follow)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

func (s *Store) DeleteFollowsForRequest(ctx context.Context, requestID string) error {
	_, err := s.DB.NewDelete().Model((*Follow)(nil)).Where("request_id = ?", requestID).Exec(ctx)
	return err
}

func (s *Store) FollowsForRequest(ctx context.Context, requestID string) ([]Follow, error) {
	out := []Follow{}
	return out, s.DB.NewSelect().Model(&out).Where("request_id = ?", requestID).OrderExpr("created_at").Scan(ctx)
}

func (s *Store) FollowsBySession(ctx context.Context, sessionID string) ([]Follow, error) {
	out := []Follow{}
	return out, s.DB.NewSelect().Model(&out).Where("session_id = ?", sessionID).OrderExpr("created_at").Scan(ctx)
}

func (s *Store) CountFollowsBySession(ctx context.Context, sessionID string) (int, error) {
	n, err := s.DB.NewSelect().Model((*Follow)(nil)).Where("session_id = ?", sessionID).Count(ctx)
	return int(n), err
}

// MarkFollowSent records the state last pushed to a follow.
func (s *Store) MarkFollowSent(ctx context.Context, id, stage string, percent int, at int64) error {
	_, err := s.DB.NewUpdate().Model((*Follow)(nil)).
		Set("last_stage = ?", stage).Set("last_percent = ?", percent).Set("last_sent_at = ?", at).
		Where("id = ?", id).Exec(ctx)
	return err
}

// ExpiredFollows returns the follows whose expires_at has passed.
func (s *Store) ExpiredFollows(ctx context.Context, now int64) ([]Follow, error) {
	out := []Follow{}
	return out, s.DB.NewSelect().Model(&out).Where("expires_at <= ?", now).Scan(ctx)
}

// FollowedRequestIDs returns the distinct ids of followed requests.
func (s *Store) FollowedRequestIDs(ctx context.Context) ([]string, error) {
	var out []string
	err := s.DB.NewSelect().Model((*Follow)(nil)).ColumnExpr("DISTINCT request_id").Scan(ctx, &out)
	return out, err
}
