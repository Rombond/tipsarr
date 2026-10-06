// Package web serves the built Svelte SPA embedded in the binary.
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// dist is filled by the Docker build (frontend/build copied here). Locally it only
// holds .gitkeep, and the handler answers with a hint instead.
//
//go:embed all:dist
var distFS embed.FS

func Handler() http.Handler {
	sub, _ := fs.Sub(distFS, "dist")
	files := http.FileServerFS(sub)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := fs.Stat(sub, "index.html"); err != nil {
			http.Error(w, "frontend not built into this binary (dev: use the Vite dev server)", http.StatusNotFound)
			return
		}
		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p == "" {
			p = "index.html"
		}
		if _, err := fs.Stat(sub, p); err != nil {
			// SPA fallback: client-side router handles unknown paths.
			r.URL.Path = "/"
		}
		files.ServeHTTP(w, r)
	})
}
