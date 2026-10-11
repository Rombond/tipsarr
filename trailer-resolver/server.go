package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Server struct {
	cfg    Config
	store  *Store
	worker *Worker
	mux    *http.ServeMux
}

func NewServer(cfg Config, st *Store, w *Worker) *Server {
	s := &Server{cfg: cfg, store: st, worker: w, mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	s.mux.HandleFunc("POST /v1/resolve", s.auth(s.resolve))
	s.mux.HandleFunc("GET /v1/status/{key}", s.auth(s.status))
	s.mux.HandleFunc("GET /v1/health", s.auth(s.health))
	s.mux.HandleFunc("POST /v1/touch", s.auth(s.touch))
	s.mux.HandleFunc("POST /v1/update", s.auth(s.update))
	s.mux.HandleFunc("DELETE /v1/cache", s.auth(s.clear))
	return s
}

func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if subtle.ConstantTimeCompare([]byte(got), []byte(s.cfg.APIKey)) != 1 {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

type statusBody struct {
	Status    string `json:"status"`
	Bytes     int64  `json:"bytes,omitempty"`
	Height    int    `json:"height,omitempty"`
	Duration  int    `json:"duration,omitempty"`
	Error     string `json:"error,omitempty"`
	Permanent bool   `json:"permanent,omitempty"`
}

func body(e Entry) statusBody {
	return statusBody{Status: e.Status, Bytes: e.Bytes, Height: e.Height, Duration: e.Duration, Error: e.Error, Permanent: e.Permanent}
}

func (s *Server) resolve(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Key       string `json:"key"`
		MaxHeight int    `json:"maxHeight"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<10)).Decode(&in) != nil || !keyRe.MatchString(in.Key) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "key must be 11 characters of A-Z a-z 0-9 _ -"})
		return
	}
	if in.MaxHeight != 0 && (in.MaxHeight < 144 || in.MaxHeight > 2160) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "maxHeight must be between 144 and 2160"})
		return
	}
	if e, ok := s.store.Get(in.Key); ok {
		switch {
		case e.Status == stReady:
			writeJSON(w, http.StatusOK, body(e))
			return
		case e.Status == stFailed && e.Permanent:
			writeJSON(w, http.StatusOK, body(e)) // asking again will not help
			return
		case e.Status == stQueued || e.Status == stRunning:
			writeJSON(w, http.StatusAccepted, body(e))
			return
		}
	}
	e := s.store.Update(in.Key, func(e *Entry) {
		e.Status, e.Error, e.Permanent, e.Attempts, e.RetryAt = stQueued, "", false, 0, 0
		e.MaxHeight = in.MaxHeight
	})
	s.worker.Poke()
	writeJSON(w, http.StatusAccepted, body(e))
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if !keyRe.MatchString(key) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad key"})
		return
	}
	e, ok := s.store.Get(key)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown key"})
		return
	}
	writeJSON(w, http.StatusOK, body(e))
}

func (s *Server) touch(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Keys []string `json:"keys"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&in) != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad body"})
		return
	}
	s.store.Touch(in.Keys)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	counts, bytes := s.store.Stats()
	state, until := s.worker.BreakerState()
	yt := version(ctx, s.cfg.YtDlp, "--version")
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":              yt != "" && state == "closed",
		"ytdlpVersion":    yt,
		"denoVersion":     version(ctx, s.cfg.Deno, "--version"),
		"ffmpegVersion":   version(ctx, s.cfg.FFmpeg, "-version"),
		"cacheBytes":      bytes,
		"cacheLimitBytes": s.cfg.CacheLimit,
		"counts":          counts,
		"breaker":         map[string]any{"state": state, "until": until},
		"lastError":       s.worker.LastError(),
	})
}

// update asks yt-dlp to replace itself with the newest release.
func (s *Server) update(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	before := version(ctx, s.cfg.YtDlp, "--version")
	out, err := exec.CommandContext(ctx, s.cfg.YtDlp, "-U").CombinedOutput()
	after := version(ctx, s.cfg.YtDlp, "--version")
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": oneLine(string(out)), "before": before, "after": after})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"before": before, "after": after})
}

func (s *Server) clear(w http.ResponseWriter, r *http.Request) {
	s.store.Clear()
	w.WriteHeader(http.StatusNoContent)
}

func fileSize(path string) int64 {
	st, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return st.Size()
}
