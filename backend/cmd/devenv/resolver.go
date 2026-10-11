package main

import (
	_ "embed"
	"encoding/json"
	"hash/fnv"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// A fake trailer-resolver: speaks the real resolver's API but "downloads" a small sample video (testdata/sample.mp4,
// generated once with AVFoundation) after a delay, so the Teasers feed and the apps can be tried without YouTube.
//
//	DEVENV_TRAILERS_DIR       where the files go (the folder Tipsarr reads as TIPSARR_TRAILERS_DIR), default ./trailers
//	DEVENV_RESOLVER_KEY       bearer key, default dev-resolver-key
//	DEVENV_RESOLVER_DELAY     seconds one "download" takes, default 4 (one at a time, like the real worker)
//	DEVENV_RESOLVER_FAIL      0..1, share of keys that fail for good ("Video unavailable"), default 0
//
//go:embed testdata/sample.mp4
var sampleVideo []byte

var resolverKeyRe = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

type fakeStatus struct {
	Status    string `json:"status"`
	Bytes     int64  `json:"bytes"`
	Height    int    `json:"height"`
	Duration  int    `json:"duration"`
	Error     string `json:"error,omitempty"`
	Permanent bool   `json:"permanent"`
}

type fakeResolver struct {
	dir   string
	key   string
	delay time.Duration
	fail  float64

	mu     sync.Mutex
	status map[string]*fakeStatus
	queue  chan string
}

func envFloat(name string, def float64) float64 {
	if v, err := strconv.ParseFloat(os.Getenv(name), 64); err == nil {
		return v
	}
	return def
}

func newFakeResolver() http.Handler {
	r := &fakeResolver{
		dir:    os.Getenv("DEVENV_TRAILERS_DIR"),
		key:    os.Getenv("DEVENV_RESOLVER_KEY"),
		delay:  time.Duration(envFloat("DEVENV_RESOLVER_DELAY", 4) * float64(time.Second)),
		fail:   envFloat("DEVENV_RESOLVER_FAIL", 0),
		status: map[string]*fakeStatus{},
		queue:  make(chan string, 1000),
	}
	if r.dir == "" {
		r.dir = "trailers"
	}
	if r.key == "" {
		r.key = "dev-resolver-key"
	}
	if err := os.MkdirAll(r.dir, 0o755); err != nil {
		log.Fatalf("resolver: %v", err)
	}
	go r.worker()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/resolve", r.resolve)
	mux.HandleFunc("GET /v1/status/{key}", r.getStatus)
	mux.HandleFunc("GET /v1/health", r.health)
	mux.HandleFunc("POST /v1/touch", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	mux.HandleFunc("POST /v1/update", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"before":"2026.08.19","after":"2026.08.19","updated":false}`))
	})
	mux.HandleFunc("DELETE /v1/cache", r.clear)
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Authorization") != "Bearer "+r.key {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		mux.ServeHTTP(w, req)
	})
}

func reply(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (r *fakeResolver) resolve(w http.ResponseWriter, req *http.Request) {
	var in struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(req.Body).Decode(&in); err != nil || !resolverKeyRe.MatchString(in.Key) {
		reply(w, http.StatusUnprocessableEntity, map[string]string{"error": "bad key"})
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if st, ok := r.status[in.Key]; ok && st.Status != "failed" {
		code := http.StatusAccepted
		if st.Status == "ready" {
			code = http.StatusOK
		}
		reply(w, code, st)
		return
	}
	st := &fakeStatus{Status: "queued"}
	r.status[in.Key] = st
	r.queue <- in.Key
	reply(w, http.StatusAccepted, st)
}

func (r *fakeResolver) worker() {
	for key := range r.queue {
		r.mu.Lock()
		if st := r.status[key]; st != nil {
			st.Status = "running"
		}
		r.mu.Unlock()
		time.Sleep(r.delay)
		h := fnv.New32a()
		_, _ = h.Write([]byte(key))
		fails := float64(h.Sum32()%1000)/1000 < r.fail
		r.mu.Lock()
		st := r.status[key]
		if st == nil { // the cache was cleared meanwhile
			r.mu.Unlock()
			continue
		}
		if fails {
			*st = fakeStatus{Status: "failed", Error: "Video unavailable (fake)", Permanent: true}
		} else if err := os.WriteFile(filepath.Join(r.dir, key+".mp4"), sampleVideo, 0o644); err != nil {
			*st = fakeStatus{Status: "failed", Error: err.Error()}
		} else {
			*st = fakeStatus{Status: "ready", Bytes: int64(len(sampleVideo)), Height: 360, Duration: 4}
		}
		r.mu.Unlock()
	}
}

func (r *fakeResolver) getStatus(w http.ResponseWriter, req *http.Request) {
	r.mu.Lock()
	defer r.mu.Unlock()
	st, ok := r.status[strings.TrimSpace(req.PathValue("key"))]
	if !ok {
		reply(w, http.StatusNotFound, map[string]string{"error": "unknown key"})
		return
	}
	reply(w, http.StatusOK, st)
}

func (r *fakeResolver) health(w http.ResponseWriter, _ *http.Request) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var counts struct {
		Queued  int `json:"queued"`
		Running int `json:"running"`
		Ready   int `json:"ready"`
		Failed  int `json:"failed"`
	}
	var bytes int64
	for _, st := range r.status {
		switch st.Status {
		case "queued":
			counts.Queued++
		case "running":
			counts.Running++
		case "ready":
			counts.Ready++
			bytes += st.Bytes
		case "failed":
			counts.Failed++
		}
	}
	reply(w, http.StatusOK, map[string]any{
		"ok": true, "ytdlpVersion": "fake", "denoVersion": "fake", "ffmpegVersion": "fake",
		"cacheBytes": bytes, "cacheLimitBytes": 5 << 30, "counts": counts,
		"breaker": map[string]any{"state": "closed", "until": 0},
	})
}

func (r *fakeResolver) clear(w http.ResponseWriter, _ *http.Request) {
	r.mu.Lock()
	r.status = map[string]*fakeStatus{}
	r.mu.Unlock()
	matches, _ := filepath.Glob(filepath.Join(r.dir, "*.mp4"))
	for _, m := range matches {
		_ = os.Remove(m)
	}
	w.WriteHeader(http.StatusNoContent)
}
