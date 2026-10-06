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
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer, securityHeaders)
	a := api.NewHuma(r, d)
	r.NotFound(web.Handler().ServeHTTP)
	return r, a
}

// securityHeaders adds conservative browser hardening. The CSP allows only same-origin
// resources (plus the inline bootstrap script and styles SvelteKit's static build uses).
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		h.Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; "+
			"img-src 'self' data:; font-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		next.ServeHTTP(w, r)
	})
}
