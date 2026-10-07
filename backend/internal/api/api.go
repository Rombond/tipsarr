// Package api holds the HTTP handlers. Handlers stay thin: parse, authorize, call a
// service, return. Business logic lives in the service packages.
package api

import (
	"context"
	"net"
	"net/http"

	"github.com/Rombond/tipsarr/backend/internal/auth"
	"github.com/Rombond/tipsarr/backend/internal/avatars"
	"github.com/Rombond/tipsarr/backend/internal/boxoffice"
	"github.com/Rombond/tipsarr/backend/internal/events"
	"github.com/Rombond/tipsarr/backend/internal/issues"
	"github.com/Rombond/tipsarr/backend/internal/jobs"
	"github.com/Rombond/tipsarr/backend/internal/library"
	"github.com/Rombond/tipsarr/backend/internal/marks"
	"github.com/Rombond/tipsarr/backend/internal/media"
	"github.com/Rombond/tipsarr/backend/internal/notify"
	"github.com/Rombond/tipsarr/backend/internal/ratings"
	"github.com/Rombond/tipsarr/backend/internal/requests"
	"github.com/Rombond/tipsarr/backend/internal/stats"
	"github.com/Rombond/tipsarr/backend/internal/store"
	"github.com/Rombond/tipsarr/backend/internal/suggestions"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

const Version = "0.1.0"

type Deps struct {
	Stats         *stats.Service
	Ratings       *ratings.Service
	Store         *store.Store
	Auth          *auth.Service
	Media         *media.Service
	Library       *library.Service
	Requests      *requests.Service
	Issues        *issues.Service
	Avatars       *avatars.Service
	Suggestions   *suggestions.Service
	BoxOffice     *boxoffice.Service
	Marks         *marks.Service
	LoginLimiter  *auth.Limiter // optional
	SetupToken    string        // required by POST /setup when non-empty (set by main on a fresh install)
	SecureCookies bool          // always mark the session cookie Secure (otherwise only behind X-Forwarded-Proto: https)
	Hub           *events.Hub
	Notify        *notify.Service
	Jobs          *jobs.Manager
	DryRun        bool   // global: nothing is ever sent to Radarr/Sonarr
	ConfigDir     string // image cache lives under here
	// ImageBaseURL overrides the TMDB image host (tests).
	ImageBaseURL string
}

type userCtxKey struct{}
type ipCtxKey struct{}

// NewHuma mounts the API on r under /api/v1 and returns the huma API (used to export the spec).
func NewHuma(r chi.Router, d Deps) huma.API {
	huma.DefaultArrayNullable = false // services always return non-nil slices; keeps generated TS types free of `| null`
	cfg := huma.DefaultConfig("Tipsarr", Version)
	cfg.Servers = []*huma.Server{{URL: "/api/v1"}}
	cfg.CreateHooks = nil // no "$schema" link fields in responses
	cfg.DocsPath = ""     // spec only; no bundled docs UI
	cfg.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"session": {Type: "apiKey", In: "cookie", Name: auth.CookieName},
	}

	var api huma.API
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(sessionMiddleware(d.Auth, d.Store))
		api = humachi.New(r, cfg)
		registerSystem(api, d)
		registerSetup(api, d)
		registerAuth(api, d)
		registerDiscover(api, d)
		registerAdmin(api, d)
		registerSync(api, d)
		registerRequests(api, d)
		registerSuggestions(api, d)
		registerBoxOffice(api, d)
		registerProfile(api, d)
		registerIssues(api, d)
		registerOIDC(api, d)
		registerMarks(api, d)
		registerUsers(api, d)
		registerServarr(api, d)
		registerWebhooks(api, d)
		registerLDAPImport(api, d)
		registerLibrary(api, d)
		registerStats(api, d)
		r.Get("/events", eventsHandler(d))
		r.Get("/images/tmdb/{size}/{file}", imageHandler(d))
		r.Get("/images/jellyfin/{id}", jellyfinImageHandler(d))
		r.Get("/users/{id}/avatar", avatarHandler(d))
		r.Post("/me/avatar", avatarUploadHandler(d))
		r.Delete("/me/avatar", avatarUploadHandler(d))
		r.Get("/auth/oidc/login", oidcLoginHandler(d))
		r.Get("/auth/oidc/callback", oidcCallbackHandler(d))
	})
	return api
}

// sessionMiddleware loads the session user (if any) into the request context.
func sessionMiddleware(a *auth.Service, st *store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(context.WithValue(r.Context(), ipCtxKey{}, remoteIP(r)))
			if c, err := r.Cookie(auth.CookieName); err == nil {
				if u, err := a.Authenticate(r.Context(), c.Value); err == nil {
					// A profile without a saved language follows the language the interface shows
					// (sent by the app), so titles and overviews match the screen. In-memory only.
					if h := r.Header.Get("X-Tipsarr-Language"); u.Language == "" && languageRe.MatchString(h) {
						u.Language = h
					}
					if u.Language == "" { // then the app-wide default, if the admin set one
						u.Language, _ = st.GetSetting(r.Context(), media.SettingDefaultLanguage)
					}
					r = r.WithContext(context.WithValue(r.Context(), userCtxKey{}, u))
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func userFrom(ctx context.Context) *store.User {
	u, _ := ctx.Value(userCtxKey{}).(*store.User)
	return u
}

// requireUser returns the logged-in user or a 401 error.
func requireUser(ctx context.Context) (*store.User, error) {
	if u := userFrom(ctx); u != nil {
		return u, nil
	}
	return nil, fail(401, "login_required", "login required")
}

// requireAdmin returns the logged-in admin or a 401/403 error.
func requireAdmin(ctx context.Context) (*store.User, error) {
	u, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	if u.Role != store.RoleAdmin {
		return nil, fail(403, "admin_only", "admin only")
	}
	return u, nil
}

func chiParam(r *http.Request, name string) string { return chi.URLParam(r, name) }

// remoteIP is the client address as seen by the router (chi's RealIP middleware has already
// applied X-Forwarded-For / X-Real-IP, which is right behind a reverse proxy).
func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func clientIP(ctx context.Context) string {
	ip, _ := ctx.Value(ipCtxKey{}).(string)
	return ip
}
