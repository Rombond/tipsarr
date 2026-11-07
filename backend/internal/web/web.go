// Package web serves the built Svelte SPA embedded in the binary.
package web

import (
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"io/fs"
	"net/http"
	"path"
	"regexp"
	"strings"
	"sync"
)

// dist is filled by the Docker build (frontend/build copied here). Locally it only
// holds .gitkeep, and the handler answers with a hint instead.
//
//go:embed all:dist
var distFS embed.FS

// Built reports whether a frontend build is embedded in this binary.
func Built() bool {
	_, err := distFS.ReadFile("dist/index.html")
	return err == nil
}

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

var inlineScript = regexp.MustCompile(`(?s)<script>(.*?)</script>`)

// ScriptHashes returns the CSP sources ('sha256-…') of the inline scripts of the embedded
// index.html (theme bootstrap, SvelteKit start-up), so the CSP needs no 'unsafe-inline' for
// scripts. Empty when no frontend is built in.
var ScriptHashes = sync.OnceValue(func() []string {
	raw, err := distFS.ReadFile("dist/index.html")
	if err != nil {
		return nil
	}
	return scriptHashes(raw)
})

func scriptHashes(html []byte) []string {
	var out []string
	for _, m := range inlineScript.FindAllSubmatch(html, -1) {
		sum := sha256.Sum256(m[1])
		out = append(out, "'sha256-"+base64.StdEncoding.EncodeToString(sum[:])+"'")
	}
	return out
}
