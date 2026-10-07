package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/Rombond/tipsarr/backend/internal/auth"
	"github.com/danielgtaylor/huma/v2"
)

const oidcCookie = "tipsarr_sso"

type authMethods struct {
	Password bool   `json:"password" doc:"The Jellyfin username/password form works"`
	SSO      bool   `json:"sso" doc:"Single sign-on (OpenID Connect) is set up"`
	SSOLabel string `json:"ssoLabel,omitempty" doc:"Name of the provider for the button"`
}

func registerOIDC(api huma.API, d Deps) {
	huma.Register(api, huma.Operation{
		OperationID: "authMethods", Method: http.MethodGet, Path: "/auth/methods",
		Summary:     "How people can sign in",
		Description: "The login page hides the password form when single sign-on is set up; it comes back if single sign-on refuses someone.",
		Tags:        []string{"auth"},
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body authMethods }, error) {
		c, err := d.Auth.OIDCConfig(ctx)
		if err != nil {
			return nil, err
		}
		return &struct{ Body authMethods }{authMethods{Password: true, SSO: c.Enabled()}}, nil
	})
}

// redirectURI is where the provider sends the browser back: this server's address as the browser
// sees it (behind a reverse proxy, from the forwarded headers).
func redirectURI(r *http.Request) string {
	scheme := "http"
	if r.Header.Get("X-Forwarded-Proto") == "https" || r.TLS != nil {
		scheme = "https"
	}
	host := r.Host
	if h := r.Header.Get("X-Forwarded-Host"); h != "" {
		host = h
	}
	return scheme + "://" + host + "/api/v1/auth/oidc/callback"
}

func ssoRefused(w http.ResponseWriter, r *http.Request, reason string) {
	http.SetCookie(w, &http.Cookie{Name: oidcCookie, Value: "", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/login?sso=refused&reason="+url.QueryEscape(reason), http.StatusFound)
}

func reasonOf(err error) string {
	switch {
	case errors.Is(err, auth.ErrSSONoUser):
		return "no_user"
	case errors.Is(err, auth.ErrSSODisabled):
		return "disabled"
	case errors.Is(err, auth.ErrSSOState):
		return "state"
	case errors.Is(err, auth.ErrSSOUnmatched):
		return "no_username"
	case errors.Is(err, auth.ErrSSOUnusable):
		return "unusable"
	}
	return "provider"
}

// oidcLoginHandler starts single sign-on: it sends the browser to the provider.
func oidcLoginHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authURL, state, err := d.Auth.BeginOIDC(r.Context(), redirectURI(r))
		if err != nil {
			slog.Warn("sso: cannot start", "err", err)
			ssoRefused(w, r, reasonOf(err))
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name: oidcCookie, Value: state, Path: "/", MaxAge: 600, HttpOnly: true,
			SameSite: http.SameSiteLaxMode, Secure: r.Header.Get("X-Forwarded-Proto") == "https" || r.TLS != nil,
		})
		http.Redirect(w, r, authURL, http.StatusFound)
	}
}

// oidcCallbackHandler finishes single sign-on. Anything that goes wrong sends the person back to
// the login page with the classic form visible.
func oidcCallbackHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("error") != "" { // the person (or the provider's rules) said no
			ssoRefused(w, r, "denied")
			return
		}
		c, err := r.Cookie(oidcCookie)
		if err != nil || c.Value == "" || c.Value != q.Get("state") {
			ssoRefused(w, r, "state")
			return
		}
		token, _, err := d.Auth.CompleteOIDC(r.Context(), redirectURI(r), q.Get("state"), q.Get("code"), r.UserAgent())
		if err != nil {
			slog.Info("sso: refused", "err", err)
			ssoRefused(w, r, reasonOf(err))
			return
		}
		http.SetCookie(w, &http.Cookie{Name: oidcCookie, Value: "", Path: "/", MaxAge: -1})
		http.SetCookie(w, &http.Cookie{
			Name: auth.CookieName, Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode,
			Secure: r.Header.Get("X-Forwarded-Proto") == "https" || r.TLS != nil, MaxAge: int(auth.SessionTTL.Seconds()),
		})
		http.Redirect(w, r, "/", http.StatusFound)
	}
}
