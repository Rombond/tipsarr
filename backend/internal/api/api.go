// Package api holds the HTTP handlers. Handlers stay thin: parse, authorize, call a
// service, return. Business logic lives in the service packages.
package api

import (
	"context"
	"net/http"

	"github.com/Rombond/tipsarr/backend/internal/auth"
	"github.com/Rombond/tipsarr/backend/internal/events"
	"github.com/Rombond/tipsarr/backend/internal/jobs"
	"github.com/Rombond/tipsarr/backend/internal/library"
	"github.com/Rombond/tipsarr/backend/internal/media"
	"github.com/Rombond/tipsarr/backend/internal/notify"
	"github.com/Rombond/tipsarr/backend/internal/requests"
	"github.com/Rombond/tipsarr/backend/internal/store"
	"github.com/Rombond/tipsarr/backend/internal/suggestions"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

const Version = "0.1.0"

type Deps struct {
	Store       *store.Store
	Auth        *auth.Service
	Media       *media.Service
	Library     *library.Service
	Requests    *requests.Service
	Suggestions *suggestions.Service
	Hub         *events.Hub
	Notify      *notify.Service
	Jobs        *jobs.Manager
	DryRun      bool   // global: nothing is ever sent to Radarr/Sonarr
	ConfigDir   string // image cache lives under here
	// ImageBaseURL overrides the TMDB image host (tests).
	ImageBaseURL string
}

type userCtxKey struct{}

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
		r.Use(sessionMiddleware(d.Auth))
		api = humachi.New(r, cfg)
		registerSystem(api, d)
		registerSetup(api, d)
		registerAuth(api, d)
		registerDiscover(api, d)
		registerAdmin(api, d)
		registerSync(api, d)
		registerRequests(api, d)
		registerSuggestions(api, d)
		registerServarr(api, d)
		registerWebhooks(api, d)
		r.Get("/events", eventsHandler(d))
		r.Get("/images/tmdb/{size}/{file}", imageHandler(d))
	})
	return api
}

// sessionMiddleware loads the session user (if any) into the request context.
func sessionMiddleware(a *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if c, err := r.Cookie(auth.CookieName); err == nil {
				if u, err := a.Authenticate(r.Context(), c.Value); err == nil {
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
	return nil, huma.Error401Unauthorized("login required")
}

// requireAdmin returns the logged-in admin or a 401/403 error.
func requireAdmin(ctx context.Context) (*store.User, error) {
	u, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	if u.Role != store.RoleAdmin {
		return nil, huma.Error403Forbidden("admin only")
	}
	return u, nil
}

func chiParam(r *http.Request, name string) string { return chi.URLParam(r, name) }
