package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/auth"
	"github.com/Rombond/tipsarr/backend/internal/clients/jellyfin"
	"github.com/Rombond/tipsarr/backend/internal/demo"
	"github.com/Rombond/tipsarr/backend/internal/store"
	"github.com/danielgtaylor/huma/v2"
)

type loginInput struct {
	UserAgent string `header:"User-Agent" hidden:"true"`
	Proto     string `header:"X-Forwarded-Proto" hidden:"true"`
	Body      struct {
		Username string `json:"username" minLength:"1" maxLength:"256"`
		Password string `json:"password" maxLength:"512"`
	}
}

type loginOutput struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
	Body      *store.User
}

// tokenInput is the app flavour of login: the token comes back in the body instead of a cookie.
type tokenInput struct {
	UserAgent string `header:"User-Agent" hidden:"true"`
	Body      struct {
		Username   string `json:"username" minLength:"1" maxLength:"256"`
		Password   string `json:"password" maxLength:"512"`
		Platform   string `json:"platform" enum:"ios,android" doc:"Which app is signing in"`
		DeviceName string `json:"deviceName,omitempty" maxLength:"64" doc:"Shown in the signed-in devices list"`
		AppVersion string `json:"appVersion,omitempty" maxLength:"32"`
	}
}

type tokenBody struct {
	Token     string     `json:"token" doc:"Send as Authorization: Bearer <token>"`
	ExpiresAt int64      `json:"expiresAt" doc:"Unix seconds; every use renews it, see sessions"`
	User      store.User `json:"user"`
}

type tokenOutput struct {
	Body tokenBody
}

type demoLoginInput struct {
	UserAgent string `header:"User-Agent" hidden:"true"`
	Proto     string `header:"X-Forwarded-Proto" hidden:"true"`
	Body      struct {
		Role string `json:"role" enum:"admin,user" doc:"admin: every right; user: a plain account"`
	}
}

type logoutOutput struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
}

type meOutput struct {
	Body *store.User
}

// signIn runs the shared part of both logins: rate limit, Jellyfin check, error mapping.
func signIn(ctx context.Context, d Deps, username, password, userAgent string, dev auth.Device) (string, *store.User, error) {
	ukey, ikey := auth.UserKey(username), auth.IPKey(clientIP(ctx))
	if d.LoginLimiter != nil {
		if wait := d.LoginLimiter.Blocked(ctx, ukey, ikey); wait > 0 {
			slog.Info("sign-in refused: locked after repeated failures", "user", username, "ip", clientIP(ctx), "for", wait.Round(time.Second).String())
			return "", nil, fail(429, "too_many_attempts", "too many failed sign-in attempts, try again in a few minutes")
		}
	}
	token, user, err := d.Auth.Login(ctx, username, password, userAgent, dev)
	if d.LoginLimiter != nil {
		if errors.Is(err, jellyfin.ErrInvalidCredentials) {
			d.LoginLimiter.Fail(ctx, ukey, ikey)
		} else if err == nil {
			d.LoginLimiter.Reset(ctx, ukey)
		}
	}
	switch {
	case errors.Is(err, jellyfin.ErrInvalidCredentials):
		return "", nil, fail(401, "invalid_credentials", "invalid username or password")
	case errors.Is(err, auth.ErrNotConfigured):
		return "", nil, fail(503, "not_set_up", "Tipsarr is not set up yet")
	case err != nil:
		slog.Warn("login failed", "err", err)
		return "", nil, fail(503, "signin_unavailable", "sign-in is temporarily unavailable")
	}
	return token, user, nil
}

func registerAuth(api huma.API, d Deps) {
	loginErrors := []int{http.StatusUnauthorized, http.StatusTooManyRequests, http.StatusServiceUnavailable}
	huma.Register(api, huma.Operation{
		OperationID: "login", Method: http.MethodPost, Path: "/auth/login",
		Summary: "Log in with a Jellyfin account", Tags: []string{"auth"}, Errors: loginErrors,
	}, func(ctx context.Context, in *loginInput) (*loginOutput, error) {
		token, user, err := signIn(ctx, d, in.Body.Username, in.Body.Password, in.UserAgent, auth.Device{Platform: "web"})
		if err != nil {
			return nil, err
		}
		return &loginOutput{
			SetCookie: http.Cookie{
				Name: auth.CookieName, Value: token, Path: "/",
				HttpOnly: true, SameSite: http.SameSiteLaxMode,
				Secure: cookieSecure(d.SecureCookies, in.Proto, false),
				MaxAge: int(d.Auth.SessionTTL().Seconds()),
			},
			Body: user,
		}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "demoLogin", Method: http.MethodPost, Path: "/auth/demo",
		Summary: "One-click sign-in of the public demo", Tags: []string{"auth"}, Errors: append(loginErrors, http.StatusNotFound),
		Description: "Only on a demo sandbox (status.demo): signs in as the seeded admin or as a plain user, without a password.",
	}, func(ctx context.Context, in *demoLoginInput) (*loginOutput, error) {
		if !d.Demo {
			return nil, fail(404, "not_found", "not a demo server")
		}
		name := demo.PlainUser
		if in.Body.Role == "admin" {
			name = demo.AdminUser
		}
		token, user, err := signIn(ctx, d, name, demo.Password, in.UserAgent, auth.Device{Platform: "web"})
		if err != nil {
			return nil, err
		}
		return &loginOutput{
			SetCookie: http.Cookie{Name: auth.CookieName, Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode,
				Secure: cookieSecure(d.SecureCookies, in.Proto, false), MaxAge: int(d.Auth.SessionTTL().Seconds())},
			Body: user,
		}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "loginToken", Method: http.MethodPost, Path: "/auth/token",
		Summary: "Log in from an app and get a bearer token", Tags: []string{"auth"}, Errors: loginErrors,
	}, func(ctx context.Context, in *tokenInput) (*tokenOutput, error) {
		dev := auth.Device{Platform: in.Body.Platform, Name: in.Body.DeviceName, AppVersion: in.Body.AppVersion}
		token, user, err := signIn(ctx, d, in.Body.Username, in.Body.Password, in.UserAgent, dev)
		if err != nil {
			return nil, err
		}
		return &tokenOutput{Body: tokenBody{Token: token, ExpiresAt: time.Now().Add(d.Auth.SessionTTL()).Unix(), User: *user}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "logout", Method: http.MethodPost, Path: "/auth/logout",
		Summary: "End the current session", Tags: []string{"auth"}, DefaultStatus: http.StatusNoContent,
		Security: []map[string][]string{{"session": {}}, {"bearer": {}}},
	}, func(ctx context.Context, _ *struct{}) (*logoutOutput, error) {
		if sess := sessionFrom(ctx); sess != nil {
			if err := d.Store.DeleteSession(ctx, sess.ID); err != nil {
				return nil, err
			}
		}
		return &logoutOutput{SetCookie: http.Cookie{Name: auth.CookieName, Value: "", Path: "/", HttpOnly: true, MaxAge: -1}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "me", Method: http.MethodGet, Path: "/me",
		Summary: "Current user", Tags: []string{"auth"},
		Security: []map[string][]string{{"session": {}}, {"bearer": {}}}, Errors: []int{http.StatusUnauthorized},
	}, func(ctx context.Context, _ *struct{}) (*meOutput, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		// the stored profile, without the per-request language hint
		if stored, err := d.Store.GetUser(ctx, u.ID); err == nil {
			u = stored
		}
		return &meOutput{Body: u}, nil
	})
}
