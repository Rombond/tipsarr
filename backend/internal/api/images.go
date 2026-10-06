package api

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/clients/tmdb"
)

var (
	imageFileRe = regexp.MustCompile(`^[A-Za-z0-9_-]{5,64}\.(jpg|png)$`)
	imageSizes  = map[string]bool{
		"w92": true, "w154": true, "w185": true, "w300": true, "w342": true,
		"w500": true, "w780": true, "w1280": true, "h632": true, "original": true,
	}
	imageClient = &http.Client{Timeout: 20 * time.Second}

	// remembered misses stop users from turning unknown image names into endless TMDB calls
	missMu sync.Mutex
	misses = map[string]time.Time{}
)

const (
	maxImageBytes = 10 << 20
	missTTL       = 10 * time.Minute
	maxMisses     = 2000
)

func recentlyMissed(key string) bool {
	missMu.Lock()
	defer missMu.Unlock()
	t, ok := misses[key]
	if ok && time.Since(t) > missTTL {
		delete(misses, key)
		return false
	}
	return ok
}

func rememberMiss(key string) {
	missMu.Lock()
	defer missMu.Unlock()
	if len(misses) >= maxMisses {
		misses = map[string]time.Time{}
	}
	misses[key] = time.Now()
}

// imageHandler serves TMDB images through a disk cache so the browser only ever talks to
// Tipsarr (same origin) and TMDB is hit once per image. Requires a logged-in user.
func imageHandler(d Deps) http.HandlerFunc {
	base := d.ImageBaseURL
	if base == "" {
		base = tmdb.DefaultImageURL
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if userFrom(r.Context()) == nil {
			http.Error(w, "login required", http.StatusUnauthorized)
			return
		}
		size, file := chiParam(r, "size"), chiParam(r, "file")
		if !imageSizes[size] || !imageFileRe.MatchString(file) {
			http.NotFound(w, r)
			return
		}
		cached := filepath.Join(d.ConfigDir, "cache", "images", size, file)
		if _, err := os.Stat(cached); err == nil {
			serveImage(w, r, cached)
			return
		}
		if recentlyMissed(size + "/" + file) {
			http.NotFound(w, r)
			return
		}
		resp, err := imageClient.Get(base + "/" + size + "/" + file)
		if err != nil {
			http.Error(w, "image fetch failed", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			rememberMiss(size + "/" + file)
			http.NotFound(w, r)
			return
		}
		if ct := resp.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(ct, "image/") && !strings.HasPrefix(ct, "application/octet-stream") {
			http.NotFound(w, r) // never cache or serve anything that is not an image
			return
		}
		if err := os.MkdirAll(filepath.Dir(cached), 0o755); err == nil {
			tmp, err := os.CreateTemp(filepath.Dir(cached), ".dl-*")
			if err == nil {
				_, cerr := io.Copy(tmp, io.LimitReader(resp.Body, maxImageBytes))
				tmp.Close()
				if cerr == nil && os.Rename(tmp.Name(), cached) == nil {
					serveImage(w, r, cached)
					return
				}
				os.Remove(tmp.Name())
			}
		}
		http.Error(w, "image cache write failed", http.StatusInternalServerError)
	}
}

func serveImage(w http.ResponseWriter, r *http.Request, path string) {
	w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	http.ServeFile(w, r, path)
}
