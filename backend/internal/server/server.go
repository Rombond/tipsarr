// Package server assembles the HTTP router: API under /api/v1, SPA for everything else.
package server

import (
	"net/http"

	"github.com/Rombond/tipsarr/backend/internal/api"
	"github.com/Rombond/tipsarr/backend/internal/web"
	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// New returns the root handler plus the huma API (for spec export).
func New(d api.Deps) (http.Handler, huma.API) {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer)
	a := api.NewHuma(r, d)
	r.NotFound(web.Handler().ServeHTTP)
	return r, a
}
