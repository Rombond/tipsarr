package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/Rombond/tipsarr/backend/internal/store"
	"github.com/danielgtaylor/huma/v2"
)

// deviceSession is one place the user is signed in (a browser or an app).
type deviceSession struct {
	ID         string `json:"id" doc:"Opaque id, usable in DELETE /me/sessions/{id}"`
	Platform   string `json:"platform" enum:"web,ios,android"`
	DeviceName string `json:"deviceName"`
	AppVersion string `json:"appVersion"`
	UserAgent  string `json:"userAgent"`
	CreatedAt  int64  `json:"createdAt"`
	LastSeenAt int64  `json:"lastSeenAt"`
	ExpiresAt  int64  `json:"expiresAt"`
	Current    bool   `json:"current" doc:"The session making this request"`
}

func registerSessions(api huma.API, d Deps) {
	sec := []map[string][]string{{"session": {}}, {"bearer": {}}}

	huma.Register(api, huma.Operation{
		OperationID: "listSessions", Method: http.MethodGet, Path: "/me/sessions",
		Summary: "Where I am signed in", Tags: []string{"auth"}, Security: sec, Errors: []int{http.StatusUnauthorized},
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body []deviceSession }, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		rows, err := d.Store.ListUserSessions(ctx, u.ID)
		if err != nil {
			return nil, err
		}
		cur := sessionFrom(ctx)
		out := make([]deviceSession, 0, len(rows))
		for i := range rows {
			s := &rows[i]
			out = append(out, deviceSession{
				ID: s.PublicID(), Platform: s.Platform, DeviceName: s.DeviceName, AppVersion: s.AppVersion,
				UserAgent: s.UserAgent, CreatedAt: s.CreatedAt, LastSeenAt: s.LastSeenAt, ExpiresAt: s.ExpiresAt,
				Current: cur != nil && cur.ID == s.ID,
			})
		}
		return &struct{ Body []deviceSession }{out}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "revokeSession", Method: http.MethodDelete, Path: "/me/sessions/{id}",
		Summary: "Sign one of my devices out", Tags: []string{"auth"}, Security: sec, DefaultStatus: http.StatusNoContent,
		Errors: []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, in *struct {
		ID string `path:"id" minLength:"16" maxLength:"16"`
	}) (*struct{}, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		if err := d.Store.DeleteUserSession(ctx, u.ID, in.ID); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, fail(404, "not_found", "no such session")
			}
			return nil, err
		}
		return nil, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "revokeOtherSessions", Method: http.MethodDelete, Path: "/me/sessions",
		Summary: "Sign out everywhere except here", Tags: []string{"auth"}, Security: sec, DefaultStatus: http.StatusNoContent,
		Errors: []int{http.StatusUnauthorized},
	}, func(ctx context.Context, _ *struct{}) (*struct{}, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		cur := sessionFrom(ctx)
		if cur == nil {
			return nil, fail(401, "login_required", "login required")
		}
		return nil, d.Store.DeleteOtherUserSessions(ctx, u.ID, cur.ID)
	})
}
