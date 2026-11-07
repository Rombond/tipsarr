// Package server assembles the HTTP router: API under /api/v1, SPA for everything else.
package server

import (
	"net/http"
	"strings"

	"github.com/Rombond/tipsarr/backend/internal/api"
	"github.com/Rombond/tipsarr/backend/internal/demo"
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
	if d.Demo {
		return demo.Guard(r), a
	}
	return r, a
}

// securityHeaders adds conservative browser hardening. Scripts: same origin plus the hashes of the
// inline scripts of the built index.html (no 'unsafe-inline'). Styles keep 'unsafe-inline': Svelte
// transitions and components set style attributes at run time. HSTS only on HTTPS requests.
func securityHeaders(next http.Handler) http.Handler {
	script := strings.Join(append([]string{"'self'"}, web.ScriptHashes()...), " ")
	csp := "default-src 'self'; script-src " + script + "; style-src 'self' 'unsafe-inline'; " +
		"img-src 'self' data:; font-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'; object-src 'none'"
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		h.Set("Content-Security-Policy", csp)
		if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
			h.Set("Strict-Transport-Security", "max-age=31536000")
		}
		next.ServeHTTP(w, r)
	})
}
