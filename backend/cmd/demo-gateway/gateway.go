package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Child is one running Tipsarr sandbox.
type Child struct {
	Addr string        // loopback host:port
	Dir  string        // its data directory
	Done chan struct{} // closed when the process exits
	stop func()        // kills it and deletes Dir
	kill func()        // only ends the process (real sandboxes)
	once sync.Once
}

// Stop kills the sandbox and deletes its data (idempotent).
func (c *Child) Stop() { c.once.Do(c.stop) }

// Spawner starts a sandbox from the template; it returns once the sandbox answers.
type Spawner interface {
	Spawn() (*Child, error)
}

type session struct {
	id       string
	child    *Child
	proxy    *httputil.ReverseProxy
	ip       string
	created  time.Time
	lastSeen atomic.Int64 // unix nanoseconds
}

// Config are the gateway's knobs (see main for the environment variables).
type Config struct {
	Warm           int
	MaxSessions    int
	IdleTimeout    time.Duration
	MaxLifetime    time.Duration
	SessionsPerIP  int
	Allowlist      *Allowlist // exempt from SessionsPerIP
	IPWindow       time.Duration
	TrustedProxies []*net.IPNet
	MaxBody        int64
	CookieSecure   bool // always mark the cookie Secure (otherwise only behind https)
}

const cookieName = "tipsarr_demo"

// Gateway hands every visitor their own sandbox and proxies to it.
type Gateway struct {
	cfg   Config
	spawn Spawner
	now   func() time.Time

	mu       sync.Mutex
	sessions map[string]*session
	warm     []*Child
	starting int
	tokens   map[string]string // app Bearer token -> session id
	perIP    map[string][]time.Time
	closed   bool

	created, expiredIdle, expiredMax, crashed, rejectedFull, rejectedIP, spawnFailed atomic.Int64
}

// NewGateway returns a gateway; call Refill (or Run) to warm the pool.
func NewGateway(cfg Config, sp Spawner) *Gateway {
	return &Gateway{cfg: cfg, spawn: sp, now: time.Now, sessions: map[string]*session{}, tokens: map[string]string{}, perIP: map[string][]time.Time{}}
}

// Refill starts sandboxes in the background until the warm pool is full (never above the session cap plus the pool).
func (g *Gateway) Refill() {
	g.mu.Lock()
	defer g.mu.Unlock()
	for !g.closed && len(g.warm)+g.starting < g.cfg.Warm && len(g.sessions)+len(g.warm)+g.starting < g.cfg.MaxSessions+g.cfg.Warm {
		g.starting++
		go func() {
			c, err := g.spawn.Spawn()
			g.mu.Lock()
			g.starting--
			if err != nil {
				g.mu.Unlock()
				g.spawnFailed.Add(1)
				slog.Error("sandbox start failed", "err", err)
				time.Sleep(5 * time.Second) // do not spin on a broken binary
				g.Refill()
				return
			}
			if g.closed {
				g.mu.Unlock()
				c.Stop()
				return
			}
			g.warm = append(g.warm, c)
			g.mu.Unlock()
		}()
	}
}

// Reap ends expired or crashed sessions, drops dead warm sandboxes, forgets old per-IP entries and refills the pool.
func (g *Gateway) Reap() {
	now := g.now()
	var dead []*Child
	g.mu.Lock()
	for id, s := range g.sessions {
		if s.child == nil { // still starting
			continue
		}
		reason := ""
		switch {
		case isDone(s.child):
			reason = "crashed"
			g.crashed.Add(1)
		case now.Sub(s.created) >= g.cfg.MaxLifetime:
			reason = "max_lifetime"
			g.expiredMax.Add(1)
		case now.Sub(time.Unix(0, s.lastSeen.Load())) >= g.cfg.IdleTimeout:
			reason = "idle"
			g.expiredIdle.Add(1)
		}
		if reason != "" {
			delete(g.sessions, id)
			dead = append(dead, s.child)
			slog.Info("session ended", "reason", reason, "age", now.Sub(s.created).Round(time.Second))
		}
	}
	for tok, id := range g.tokens {
		if g.sessions[id] == nil {
			delete(g.tokens, tok)
		}
	}
	alive := g.warm[:0]
	for _, c := range g.warm {
		if isDone(c) {
			dead = append(dead, c)
		} else {
			alive = append(alive, c)
		}
	}
	g.warm = alive
	for ip, ts := range g.perIP {
		if keep := recent(ts, now, g.cfg.IPWindow); len(keep) == 0 {
			delete(g.perIP, ip)
		} else {
			g.perIP[ip] = keep
		}
	}
	g.mu.Unlock()
	for _, c := range dead {
		c.Stop()
	}
	g.Refill()
}

// Run reaps every tick until stop is closed, then shuts every sandbox down.
func (g *Gateway) Run(tick time.Duration, stop <-chan struct{}) {
	g.Refill()
	t := time.NewTicker(tick)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			g.Reap()
		case <-stop:
			g.Close()
			return
		}
	}
}

// Close stops every sandbox.
func (g *Gateway) Close() {
	g.mu.Lock()
	g.closed = true
	all := append([]*Child{}, g.warm...)
	for _, s := range g.sessions {
		if s.child != nil {
			all = append(all, s.child)
		}
	}
	g.warm, g.sessions, g.tokens = nil, map[string]*session{}, map[string]string{}
	g.mu.Unlock()
	var wg sync.WaitGroup
	for _, c := range all {
		wg.Add(1)
		go func() { defer wg.Done(); c.Stop() }()
	}
	wg.Wait()
}

func isDone(c *Child) bool {
	select {
	case <-c.Done:
		return true
	default:
		return false
	}
}

func recent(ts []time.Time, now time.Time, window time.Duration) []time.Time {
	out := ts[:0]
	for _, t := range ts {
		if now.Sub(t) < window {
			out = append(out, t)
		}
	}
	return out
}

type refusal int

const (
	okAlloc refusal = iota
	refusedFull
	refusedIP
	refusedStart
)

// allocate gives the visitor a sandbox: a warm one, or a fresh one when the pool is empty.
func (g *Gateway) allocate(ip string) (*session, refusal) {
	now := g.now()
	g.mu.Lock()
	if g.closed || len(g.sessions) >= g.cfg.MaxSessions {
		g.mu.Unlock()
		g.rejectedFull.Add(1)
		return nil, refusedFull
	}
	ts := recent(g.perIP[ip], now, g.cfg.IPWindow)
	if g.cfg.SessionsPerIP > 0 && len(ts) >= g.cfg.SessionsPerIP && !g.cfg.Allowlist.Allowed(ip) {
		g.perIP[ip] = ts
		g.mu.Unlock()
		g.rejectedIP.Add(1)
		return nil, refusedIP
	}
	g.perIP[ip] = append(ts, now)
	var c *Child
	for len(g.warm) > 0 && c == nil {
		c, g.warm = g.warm[0], g.warm[1:]
		if isDone(c) {
			go c.Stop()
			c = nil
		}
	}
	// reserve the slot while a cold sandbox starts, so concurrent visitors cannot pass the cap
	s := &session{id: newID(), ip: ip, created: now}
	g.sessions[s.id] = s
	g.mu.Unlock()

	if c == nil {
		var err error
		if c, err = g.spawn.Spawn(); err != nil {
			g.mu.Lock()
			delete(g.sessions, s.id)
			g.mu.Unlock()
			g.spawnFailed.Add(1)
			slog.Error("sandbox start failed", "err", err)
			return nil, refusedStart
		}
	}
	s.lastSeen.Store(now.UnixNano())
	s.proxy = g.newProxy(c.Addr, ip)
	g.mu.Lock()
	s.child = c
	g.mu.Unlock()
	g.created.Add(1)
	go g.Refill()
	return s, okAlloc
}

// newProxy forwards to one sandbox, with the visitor's address and without the gateway cookie.
func (g *Gateway) newProxy(addr, ip string) *httputil.ReverseProxy {
	target := &url.URL{Scheme: "http", Host: addr}
	return &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			pr.Out.Host = pr.In.Host
			pr.Out.Header.Del("X-Forwarded-For")
			pr.Out.Header.Set("X-Forwarded-For", ip)
			pr.Out.Header.Set("X-Real-IP", ip)
			if g.isHTTPS(pr.In) {
				pr.Out.Header.Set("X-Forwarded-Proto", "https")
			} else {
				pr.Out.Header.Set("X-Forwarded-Proto", "http")
			}
			removeCookie(pr.Out, cookieName)
		},
		FlushInterval: -1, // server-sent events go through at once
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			slog.Warn("sandbox unreachable", "err", err)
			http.Error(w, "sandbox unreachable", http.StatusBadGateway)
		},
	}
}

// end drops a session at once (a failed app sign-in must not hold a sandbox).
func (g *Gateway) end(s *session) {
	g.mu.Lock()
	delete(g.sessions, s.id)
	g.mu.Unlock()
	if s.child != nil {
		s.child.Stop()
	}
	go g.Refill()
}

func removeCookie(r *http.Request, name string) {
	cs := r.Cookies()
	r.Header.Del("Cookie")
	for _, c := range cs {
		if c.Name != name {
			r.AddCookie(c)
		}
	}
}

func newID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

// bearer is the app token of the request ("" when none).
func bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) > 7 && strings.EqualFold(h[:7], "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

// lookup finds the session of a request: by the gateway cookie (web), or by the app's Bearer token.
func (g *Gateway) lookup(r *http.Request) *session {
	id := ""
	if c, err := r.Cookie(cookieName); err == nil && len(c.Value) == 64 {
		id = c.Value
	}
	tok := bearer(r)
	g.mu.Lock()
	defer g.mu.Unlock()
	if id == "" && tok != "" {
		id = g.tokens[tok]
	}
	if id == "" {
		return nil
	}
	s := g.sessions[id]
	if s == nil || s.child == nil {
		return nil
	}
	now := g.now()
	if now.Sub(s.created) >= g.cfg.MaxLifetime || now.Sub(time.Unix(0, s.lastSeen.Load())) >= g.cfg.IdleTimeout || isDone(s.child) {
		return nil // the reaper cleans it up
	}
	return s
}

// clientIP is the visitor's address: the peer, or, when the peer is a trusted proxy, the last address in
// X-Forwarded-For that is not a trusted proxy.
func (g *Gateway) clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if !g.trusted(host) {
		return host
	}
	var hops []string
	for _, v := range r.Header.Values("X-Forwarded-For") {
		for _, h := range strings.Split(v, ",") {
			if h = strings.TrimSpace(h); h != "" {
				hops = append(hops, h)
			}
		}
	}
	for i := len(hops) - 1; i >= 0; i-- {
		if net.ParseIP(hops[i]) == nil {
			break
		}
		if !g.trusted(hops[i]) {
			return hops[i]
		}
	}
	return host
}

func (g *Gateway) trusted(ip string) bool {
	p := net.ParseIP(ip)
	if p == nil {
		return false
	}
	for _, n := range g.cfg.TrustedProxies {
		if n.Contains(p) {
			return true
		}
	}
	return false
}

func (g *Gateway) isHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return g.trusted(host) && strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

// wantsPage: only a browser loading a page gets a new sandbox, never a script calling the API.
func wantsPage(r *http.Request) bool {
	return r.Method == http.MethodGet && strings.Contains(r.Header.Get("Accept"), "text/html") && !strings.HasPrefix(r.URL.Path, "/api/")
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s := g.lookup(r)
	if r.URL.Path == "/__demo/session" {
		g.sessionInfo(w, s)
		return
	}
	signIn := r.Method == http.MethodPost && (r.URL.Path == "/api/v1/auth/token" || r.URL.Path == "/api/v1/auth/login")
	fresh := false
	if s == nil {
		switch {
		case publicCall(r) && bearer(r) == "":
			g.servePublic(w, r)
			return
		case signIn:
			// an app signing in: it gets its sandbox now, kept only if the sign-in succeeds
			var why refusal
			s, why = g.allocate(g.clientIP(r))
			switch why {
			case refusedFull, refusedStart:
				w.Header().Set("Retry-After", "300")
				problem(w, http.StatusServiceUnavailable, "the demo is full, try again in a few minutes", "demo_full")
				return
			case refusedIP:
				w.Header().Set("Retry-After", "300")
				problem(w, http.StatusTooManyRequests, "too many demo sessions from this address, wait a little", "demo_ip_limit")
				return
			}
			fresh = true
		case !wantsPage(r):
			problem(w, http.StatusUnauthorized, "the demo session has ended, sign in again", "demo_expired")
			return
		default:
			var why refusal
			s, why = g.allocate(g.clientIP(r))
			switch why {
			case refusedFull, refusedStart:
				page(w, r, http.StatusServiceUnavailable, "full")
				return
			case refusedIP:
				page(w, r, http.StatusTooManyRequests, "ip")
				return
			}
			http.SetCookie(w, g.cookie(r, s))
		}
	}
	if r.URL.Path != "/api/v1/events" { // an open event stream is not activity
		s.lastSeen.Store(g.now().UnixNano())
	}
	if g.cfg.MaxBody > 0 && r.Body != nil {
		r.Body = http.MaxBytesReader(w, r.Body, g.cfg.MaxBody)
	}
	if !signIn {
		s.proxy.ServeHTTP(w, r)
		return
	}
	ok := false
	p := *s.proxy
	p.ModifyResponse = func(resp *http.Response) error {
		if resp.StatusCode/100 != 2 {
			return nil
		}
		ok = true
		if r.URL.Path == "/api/v1/auth/token" {
			b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
			resp.Body.Close()
			resp.Body = io.NopCloser(bytes.NewReader(b))
			var out struct{ Token string }
			if err == nil && json.Unmarshal(b, &out) == nil && out.Token != "" {
				g.mu.Lock()
				g.tokens[out.Token] = s.id
				g.mu.Unlock()
			}
		} else if fresh {
			resp.Header.Add("Set-Cookie", g.cookie(r, s).String())
		}
		return nil
	}
	p.ServeHTTP(w, r)
	if fresh && !ok {
		g.end(s)
	}
}

func (g *Gateway) cookie(r *http.Request, s *session) *http.Cookie {
	return &http.Cookie{Name: cookieName, Value: s.id, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode,
		Secure: g.cfg.CookieSecure || g.isHTTPS(r), MaxAge: int(g.cfg.MaxLifetime.Seconds())}
}

// problem writes an API error in the sandbox's shape (the code in errors[0].value).
func problem(w http.ResponseWriter, status int, detail, code string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"title": http.StatusText(status), "status": status, "detail": detail,
		"errors": []map[string]any{{"message": detail, "location": "code", "value": code}}})
}

// publicCall: the read-only calls an app makes before signing in (server check). They need no sandbox of their own.
func publicCall(r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	switch r.URL.Path {
	case "/api/v1/status", "/api/v1/health", "/api/v1/auth/methods", "/api/v1/setup/status":
		return true
	}
	return false
}

// servePublic answers a public call from any running sandbox (a warm one first), without allocating anything.
func (g *Gateway) servePublic(w http.ResponseWriter, r *http.Request) {
	g.mu.Lock()
	addr := ""
	for _, c := range g.warm {
		if !isDone(c) {
			addr = c.Addr
			break
		}
	}
	if addr == "" {
		for _, s := range g.sessions {
			if s.child != nil && !isDone(s.child) {
				addr = s.child.Addr
				break
			}
		}
	}
	g.mu.Unlock()
	if addr == "" {
		w.Header().Set("Retry-After", "30")
		problem(w, http.StatusServiceUnavailable, "the demo is starting, try again in a moment", "demo_starting")
		return
	}
	r.Header.Del("Cookie") // a sandbox session cookie means nothing to another sandbox
	g.newProxy(addr, g.clientIP(r)).ServeHTTP(w, r)
}

func (g *Gateway) sessionInfo(w http.ResponseWriter, s *session) {
	w.Header().Set("Cache-Control", "no-store")
	if s == nil {
		http.Error(w, "no session", http.StatusNotFound)
		return
	}
	end := s.created.Add(g.cfg.MaxLifetime)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"endsAt": end.Unix(), "idleTimeout": int(g.cfg.IdleTimeout.Seconds())})
}

// Metrics writes the Prometheus text format.
func (g *Gateway) Metrics(w http.ResponseWriter, extra map[string]int64) {
	g.mu.Lock()
	active, warm, starting := len(g.sessions), len(g.warm), g.starting
	g.mu.Unlock()
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	gauge := func(name, help string, v int64) {
		fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s gauge\n%s %d\n", name, help, name, name, v)
	}
	counter := func(name, help, label string, vals map[string]int64) {
		fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s counter\n", name, help, name)
		for k, v := range vals {
			if label == "" {
				fmt.Fprintf(w, "%s %d\n", name, v)
			} else {
				fmt.Fprintf(w, "%s{%s=%q} %d\n", name, label, k, v)
			}
		}
	}
	gauge("demo_sessions_active", "Sessions with a sandbox.", int64(active))
	gauge("demo_pool_warm", "Started sandboxes waiting for a visitor.", int64(warm))
	gauge("demo_pool_starting", "Sandboxes being started.", int64(starting))
	gauge("demo_sessions_max", "Session cap.", int64(g.cfg.MaxSessions))
	counter("demo_sessions_created_total", "Sessions handed out.", "", map[string]int64{"": g.created.Load()})
	counter("demo_sessions_ended_total", "Sessions ended, by reason.", "reason", map[string]int64{"idle": g.expiredIdle.Load(), "max_lifetime": g.expiredMax.Load(), "crashed": g.crashed.Load()})
	counter("demo_sessions_rejected_total", "Visitors turned away, by reason.", "reason", map[string]int64{"full": g.rejectedFull.Load(), "ip_limit": g.rejectedIP.Load()})
	counter("demo_sandbox_start_failures_total", "Sandboxes that failed to start.", "", map[string]int64{"": g.spawnFailed.Load()})
	for k, v := range extra {
		counter(k, k, "", map[string]int64{"": v})
	}
}
