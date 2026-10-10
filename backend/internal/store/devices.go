package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/uptrace/bun"
)

// Push categories a device can opt into (bitmask).
const (
	PushRequests = 1 << iota // approved, declined, failed, available (my requests)
	PushAdmin                // new requests waiting for approval (admins)
	PushIssues               // issue reports and comments
	PushAll      = PushRequests | PushAdmin | PushIssues
)

// Device is an app install that can receive push notifications, tied to one session.
type Device struct {
	bun.BaseModel `bun:"table:devices"`

	ID         string `bun:"id,pk"`
	UserID     string `bun:"user_id"`
	SessionID  string `bun:"session_id"`
	Platform   string `bun:"platform"` // ios or android
	PushToken  string `bun:"push_token"`
	Sandbox    bool   `bun:"sandbox"`
	AppVersion string `bun:"app_version"`
	Language   string `bun:"language"`
	Categories int    `bun:"categories"`
	CreatedAt  int64  `bun:"created_at"`
	UpdatedAt  int64  `bun:"updated_at"`
}

// SaveDevice registers or refreshes the device of a session. The same token under another session
// (the app signed in again) is dropped, so a device never receives a notification twice.
func (s *Store) SaveDevice(ctx context.Context, d *Device) error {
	now := time.Now().Unix()
	d.UpdatedAt = now
	return s.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewDelete().Model((*Device)(nil)).
			Where("push_token = ?", d.PushToken).Where("session_id <> ?", d.SessionID).Exec(ctx); err != nil {
			return err
		}
		res, err := tx.NewUpdate().Model((*Device)(nil)).
			Set("user_id = ?", d.UserID).Set("platform = ?", d.Platform).Set("push_token = ?", d.PushToken).
			Set("sandbox = ?", d.Sandbox).Set("app_version = ?", d.AppVersion).Set("language = ?", d.Language).
			Set("categories = ?", d.Categories).Set("updated_at = ?", now).
			Where("session_id = ?", d.SessionID).Exec(ctx)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			return nil
		}
		if d.ID == "" {
			d.ID = NewID()
		}
		d.CreatedAt = now
		_, err = tx.NewInsert().Model(d).Exec(ctx)
		return err
	})
}

// DeviceBySession returns the device registered by a session, or ErrNotFound.
func (s *Store) DeviceBySession(ctx context.Context, sessionID string) (*Device, error) {
	d := new(Device)
	err := s.DB.NewSelect().Model(d).Where("session_id = ?", sessionID).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return d, err
}

func (s *Store) DeleteDeviceBySession(ctx context.Context, sessionID string) error {
	_, err := s.DB.NewDelete().Model((*Device)(nil)).Where("session_id = ?", sessionID).Exec(ctx)
	return err
}

// DeleteDevice removes a device by id (the relay said its token is no longer valid).
func (s *Store) DeleteDevice(ctx context.Context, id string) error {
	_, err := s.DB.NewDelete().Model((*Device)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

// DevicesFor returns the devices of the given users plus, when admins is set, every admin's devices.
func (s *Store) DevicesFor(ctx context.Context, userIDs []string, admins bool) ([]Device, error) {
	out := []Device{}
	if len(userIDs) == 0 && !admins {
		return out, nil
	}
	q := s.DB.NewSelect().Model(&out)
	switch {
	case admins && len(userIDs) > 0:
		q = q.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where("user_id IN (?)", bun.In(userIDs)).
				WhereOr("user_id IN (SELECT id FROM users WHERE role = ?)", RoleAdmin)
		})
	case admins:
		q = q.Where("user_id IN (SELECT id FROM users WHERE role = ?)", RoleAdmin)
	default:
		q = q.Where("user_id IN (?)", bun.In(userIDs))
	}
	return out, q.OrderExpr("created_at").Scan(ctx)
}

// pruneDevices drops devices whose session no longer exists (signed out, revoked, expired, purged).
func (s *Store) pruneDevices(ctx context.Context) error {
	_, err := s.DB.NewDelete().Model((*Device)(nil)).
		Where("session_id NOT IN (SELECT id FROM sessions)").Exec(ctx)
	return err
}
