package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/Rombond/tipsarr/backend/internal/auth"
	"github.com/Rombond/tipsarr/backend/internal/clients/jellyfin"
	"github.com/Rombond/tipsarr/backend/internal/store"
	"github.com/danielgtaylor/huma/v2"
)

type loginInput struct {
	UserAgent string `header:"User-Agent" hidden:"true"`
	Proto     string `header:"X-Forwarded-Proto" hidden:"true"`
	Body      struct {
		Username string `json:"username" minLength:"1"`
		Password string `json:"password"`
	}
}

type loginOutput struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
	Body      *store.User
}

type logoutInput struct {
	Session string `cookie:"tipsarr_session"`
}

type logoutOutput struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
}

type meOutput struct {
	Body *store.User
}

func registerAuth(api huma.API, d Deps) {
	huma.Register(api, huma.Operation{
		OperationID: "login", Method: http.MethodPost, Path: "/auth/login",
		Summary: "Log in with a Jellyfin account", Tags: []string{"auth"},
		Errors: []int{http.StatusUnauthorized, http.StatusServiceUnavailable},
	}, func(ctx context.Context, in *loginInput) (*loginOutput, error) {
		token, user, err := d.Auth.Login(ctx, in.Body.Username, in.Body.Password, in.UserAgent)
		switch {
		case errors.Is(err, jellyfin.ErrInvalidCredentials):
			return nil, huma.Error401Unauthorized("invalid username or password")
		case errors.Is(err, auth.ErrNotConfigured):
			return nil, huma.Error503ServiceUnavailable("Tipsarr is not set up yet")
		case err != nil:
			return nil, huma.Error503ServiceUnavailable("login failed: " + err.Error())
		}
		return &loginOutput{
			SetCookie: http.Cookie{
				Name: auth.CookieName, Value: token, Path: "/",
				HttpOnly: true, SameSite: http.SameSiteLaxMode,
				Secure: in.Proto == "https",
				MaxAge: int(auth.SessionTTL.Seconds()),
			},
			Body: user,
		}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "logout", Method: http.MethodPost, Path: "/auth/logout",
		Summary: "End the current session", Tags: []string{"auth"}, DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *logoutInput) (*logoutOutput, error) {
		if in.Session != "" {
			if err := d.Auth.Logout(ctx, in.Session); err != nil {
				return nil, err
			}
		}
		return &logoutOutput{SetCookie: http.Cookie{Name: auth.CookieName, Value: "", Path: "/", HttpOnly: true, MaxAge: -1}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "me", Method: http.MethodGet, Path: "/me",
		Summary: "Current user", Tags: []string{"auth"},
		Security: []map[string][]string{{"session": {}}}, Errors: []int{http.StatusUnauthorized},
	}, func(ctx context.Context, _ *struct{}) (*meOutput, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		return &meOutput{Body: u}, nil
	})
}
