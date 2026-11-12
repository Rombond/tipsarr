package demo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Upstream is one service the caching proxy forwards to.
type Upstream struct {
	Prefix  string              // e.g. "/tmdb/"
	Base    string              // e.g. "https://api.themoviedb.org"
	TTL     time.Duration       // how long an answer is reused
	Limiter *Limiter            // shared outbound rate limit (nil = none)
	TMDBKey string              // when set, the key is injected (any api_key from the caller is dropped)
	Filter  func([]byte) []byte // when set, rewrites successful JSON answers as they are served (the cache keeps them raw)
}

// Proxy is a caching, rate-limited reverse proxy for GET calls to TMDB (API and images) and the box office. Every
// demo sandbox talks to TMDB through it, so all of them share one cache and one rate limit, and the TMDB key never
// enters a sandbox. The cache is a directory of files written by this one process (no locking between processes).
type Proxy struct {
	Dir       string
	Upstreams []Upstream
	Client    *http.Client

	mu       sync.Mutex
	inflight map[string]*call

	Hits, Misses, Errors atomic.Int64
}

type call struct {
	done chan struct{}
	ent  *entry
	err  error
}

type entry struct {
	Status      int    `json:"status"`
	ContentType string `json:"contentType"`
	Fetched     int64  `json:"fetched"`
	body        []byte
}

const maxProxyBody = 15 << 20

// NewProxy returns a proxy caching under dir.
func NewProxy(dir string, ups ...Upstream) *Proxy {
	return &Proxy{Dir: dir, Upstreams: ups, Client: &http.Client{Timeout: 30 * time.Second}, inflight: map[string]*call{}}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	for _, up := range p.Upstreams {
		if !strings.HasPrefix(r.URL.Path, up.Prefix) {
			continue
		}
		q := r.URL.Query()
		q.Del("api_key")
		path := "/" + strings.TrimPrefix(r.URL.Path, up.Prefix)
		if strings.Contains(path, "..") {
			http.NotFound(w, r)
			return
		}
		key := up.Prefix + path + "?" + q.Encode()
		ent, err := p.get(r.Context(), up, key, path, q)
		if err != nil {
			p.Errors.Add(1)
			status := http.StatusBadGateway
			if errors.Is(err, ErrRateLimited) {
				status = http.StatusTooManyRequests
				w.Header().Set("Retry-After", "2")
			}
			http.Error(w, err.Error(), status)
			return
		}
		if ent.ContentType != "" {
			w.Header().Set("Content-Type", ent.ContentType)
		}
		body := ent.body
		if up.Filter != nil && ent.Status == http.StatusOK && isTMDBJSON(ent.ContentType) {
			body = up.Filter(body)
		}
		w.WriteHeader(ent.Status)
		_, _ = w.Write(body)
		return
	}
	http.NotFound(w, r)
}

func (p *Proxy) file(key string) string {
	h := sha256.Sum256([]byte(key))
	s := hex.EncodeToString(h[:])
	return filepath.Join(p.Dir, s[:2], s)
}

func (p *Proxy) load(key string, ttl time.Duration) *entry {
	f := p.file(key)
	meta, err := os.ReadFile(f + ".json")
	if err != nil {
		return nil
	}
	var e entry
	if json.Unmarshal(meta, &e) != nil || time.Since(time.Unix(e.Fetched, 0)) > ttl {
		return nil
	}
	if e.body, err = os.ReadFile(f); err != nil {
		return nil
	}
	return &e
}

func (p *Proxy) store(key string, e *entry) {
	f := p.file(key)
	if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
		return
	}
	meta, _ := json.Marshal(e)
	for _, part := range []struct {
		name string
		data []byte
	}{{f, e.body}, {f + ".json", meta}} { // body first: a meta file always has its body
		tmp, err := os.CreateTemp(filepath.Dir(f), ".tmp-*")
		if err != nil {
			return
		}
		_, werr := tmp.Write(part.data)
		tmp.Close()
		if werr != nil || os.Rename(tmp.Name(), part.name) != nil {
			os.Remove(tmp.Name())
			return
		}
	}
}

func (p *Proxy) get(ctx context.Context, up Upstream, key, path string, q url.Values) (*entry, error) {
	ttl := up.TTL
	if e := p.load(key, ttl); e != nil {
		p.Hits.Add(1)
		return e, nil
	}
	p.mu.Lock()
	if c, ok := p.inflight[key]; ok { // someone is already fetching it: wait for that answer
		p.mu.Unlock()
		select {
		case <-c.done:
			p.Hits.Add(1)
			return c.ent, c.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	c := &call{done: make(chan struct{})}
	p.inflight[key] = c
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		delete(p.inflight, key)
		p.mu.Unlock()
		close(c.done)
	}()
	p.Misses.Add(1)
	// fetch with a context of its own: a caller who gives up must not fail the others waiting for this answer
	fctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	c.ent, c.err = p.fetch(fctx, up, path, q)
	if c.err == nil && (c.ent.Status == http.StatusOK || c.ent.Status == http.StatusNotFound) {
		p.store(key, c.ent)
	}
	return c.ent, c.err
}

func (p *Proxy) fetch(ctx context.Context, up Upstream, path string, q url.Values) (*entry, error) {
	if up.Limiter != nil {
		if err := up.Limiter.Wait(ctx, 20*time.Second); err != nil {
			return nil, err
		}
	}
	bearer := strings.HasPrefix(up.TMDBKey, "eyJ")
	if up.TMDBKey != "" && !bearer {
		q.Set("api_key", up.TMDBKey)
	}
	u := strings.TrimRight(up.Base, "/") + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	if bearer {
		req.Header.Set("Authorization", "Bearer "+up.TMDBKey)
	}
	req.Header.Set("User-Agent", "Tipsarr-demo")
	resp, err := p.Client.Do(req)
	if err != nil {
		var ue *url.Error
		if errors.As(err, &ue) { // the URL carries the key
			err = ue.Err
		}
		slog.Warn("demo proxy: upstream unreachable", "upstream", up.Prefix, "err", err)
		return nil, errors.New("upstream unreachable")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxProxyBody))
	if err != nil {
		return nil, err
	}
	return &entry{Status: resp.StatusCode, ContentType: resp.Header.Get("Content-Type"), Fetched: time.Now().Unix(), body: body}, nil
}

// ErrRateLimited is returned when the outbound budget stays exhausted for too long.
var ErrRateLimited = errors.New("demo: upstream rate limit, retry later")

// Limiter is a token bucket shared by every sandbox.
type Limiter struct {
	mu     sync.Mutex
	rate   float64 // tokens per second
	burst  float64
	tokens float64
	last   time.Time
}

// NewLimiter allows rate calls per second on average, burst at once.
func NewLimiter(rate float64, burst int) *Limiter {
	return &Limiter{rate: rate, burst: float64(burst), tokens: float64(burst), last: time.Now()}
}

// Wait takes one token, waiting at most max for it.
func (l *Limiter) Wait(ctx context.Context, max time.Duration) error {
	deadline := time.Now().Add(max)
	for {
		l.mu.Lock()
		now := time.Now()
		l.tokens = min(l.burst, l.tokens+now.Sub(l.last).Seconds()*l.rate)
		l.last = now
		if l.tokens >= 1 {
			l.tokens--
			l.mu.Unlock()
			return nil
		}
		wait := time.Duration((1 - l.tokens) / l.rate * float64(time.Second))
		l.mu.Unlock()
		if now.Add(wait).After(deadline) {
			return ErrRateLimited
		}
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
