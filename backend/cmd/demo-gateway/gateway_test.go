package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeSpawner starts in-process sandboxes: an HTTP server that says which sandbox it is.
type fakeSpawner struct {
	t       *testing.T
	n       atomic.Int32
	mu      sync.Mutex
	stopped []string
	release chan struct{} // the event stream waits on it
}

func (f *fakeSpawner) Spawn() (*Child, error) {
	id := fmt.Sprintf("sb%d", f.n.Add(1))
	dir, _ := os.MkdirTemp(f.t.TempDir(), id)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/events" {
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(w, "data: hello\n\n")
			w.(http.Flusher).Flush()
			select {
			case <-f.release:
			case <-r.Context().Done():
				return
			}
			fmt.Fprint(w, "data: bye\n\n")
			return
		}
		if r.URL.Path == "/api/v1/status" {
			fmt.Fprintf(w, `{"demo":true,"from":%q}`, id)
			return
		}
		if r.URL.Path == "/api/v1/auth/token" {
			b, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(b), `"password":"demo"`) {
				http.Error(w, `{"status":401}`, http.StatusUnauthorized)
				return
			}
			fmt.Fprintf(w, `{"token":"tok-%s","expiresAt":0,"user":{}}`, id)
			return
		}
		if r.Method == http.MethodPost {
			if _, err := io.ReadAll(r.Body); err != nil {
				http.Error(w, "too big", http.StatusRequestEntityTooLarge)
				return
			}
		}
		_, hasDemo := r.Header["Cookie"]
		fmt.Fprintf(w, "%s ip=%s democookie=%v", id, r.Header.Get("X-Forwarded-For"), hasDemo && strings.Contains(r.Header.Get("Cookie"), cookieName))
	}))
	done := make(chan struct{})
	c := &Child{Addr: strings.TrimPrefix(srv.URL, "http://"), Dir: dir, Done: done}
	c.stop = func() {
		srv.CloseClientConnections()
		srv.Close()
		select {
		case <-done:
		default:
			close(done)
		}
		os.RemoveAll(dir)
		f.mu.Lock()
		f.stopped = append(f.stopped, id)
		f.mu.Unlock()
	}
	return c, nil
}

type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time      { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *clock) add(d time.Duration) { c.mu.Lock(); c.t = c.t.Add(d); c.mu.Unlock() }

func newTestGateway(t *testing.T, cfg Config) (*Gateway, *fakeSpawner, *clock, *httptest.Server) {
	t.Helper()
	if cfg.IdleTimeout == 0 {
		cfg.IdleTimeout = 30 * time.Minute
	}
	if cfg.MaxLifetime == 0 {
		cfg.MaxLifetime = time.Hour
	}
	if cfg.IPWindow == 0 {
		cfg.IPWindow = time.Hour
	}
	sp := &fakeSpawner{t: t, release: make(chan struct{})}
	g := NewGateway(cfg, sp)
	ck := &clock{t: time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)}
	g.now = ck.now
	srv := httptest.NewServer(g)
	t.Cleanup(func() { close(sp.release); srv.Close(); g.Close() })
	return g, sp, ck, srv
}

type visitor struct {
	srv    *httptest.Server
	cookie *http.Cookie
	ip     string // sent as X-Forwarded-For
	lang   string
}

func (v *visitor) get(t *testing.T, path string, page bool) (int, string) {
	t.Helper()
	req, _ := http.NewRequest("GET", v.srv.URL+path, nil)
	if page {
		req.Header.Set("Accept", "text/html,application/xhtml+xml")
	}
	if v.ip != "" {
		req.Header.Set("X-Forwarded-For", v.ip)
	}
	if v.lang != "" {
		req.Header.Set("Accept-Language", v.lang)
	}
	if v.cookie != nil {
		req.AddCookie(v.cookie)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	for _, c := range resp.Cookies() {
		if c.Name == cookieName {
			v.cookie = c
		}
	}
	return resp.StatusCode, string(b)
}

func waitUntil(t *testing.T, what string, cond func() bool) {
	t.Helper()
	for i := 0; i < 200; i++ {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func (g *Gateway) counts() (sessions, warm int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.sessions), len(g.warm)
}

func TestSessionAllocation(t *testing.T) {
	g, _, _, srv := newTestGateway(t, Config{Warm: 2, MaxSessions: 5})
	g.Refill()
	waitUntil(t, "warm pool", func() bool { _, w := g.counts(); return w == 2 })

	a := &visitor{srv: srv}
	code, body := a.get(t, "/", true)
	if code != 200 || a.cookie == nil || !a.cookie.HttpOnly || len(a.cookie.Value) != 64 {
		t.Fatalf("first visit = %d %q cookie %+v", code, body, a.cookie)
	}
	first := strings.Fields(body)[0]
	if strings.Contains(body, "democookie=true") {
		t.Fatalf("the gateway cookie reached the sandbox: %s", body)
	}
	// the same visitor stays on the same sandbox, API calls included
	if _, body := a.get(t, "/api/v1/me", false); !strings.HasPrefix(body, first+" ") {
		t.Fatalf("second call went to %q, want %s", body, first)
	}
	// another visitor gets another sandbox
	b := &visitor{srv: srv}
	if _, body := b.get(t, "/", true); strings.HasPrefix(body, first+" ") {
		t.Fatalf("two visitors share %s", first)
	}
	// the pool refills to 2 warm sandboxes
	waitUntil(t, "refill", func() bool { s, w := g.counts(); return s == 2 && w == 2 })

	// a script without a session gets no sandbox
	c := &visitor{srv: srv}
	if code, body := c.get(t, "/api/v1/requests", false); code != 401 || !strings.Contains(body, "demo_expired") {
		t.Fatalf("api without session = %d %s", code, body)
	}
	if s, _ := g.counts(); s != 2 {
		t.Fatalf("sessions = %d", s)
	}
	// session info for the banner
	if code, body := a.get(t, "/__demo/session", false); code != 200 || !strings.Contains(body, `"endsAt"`) {
		t.Fatalf("session info = %d %s", code, body)
	}
}

func TestIdleAndMaxLifetime(t *testing.T) {
	g, sp, ck, srv := newTestGateway(t, Config{Warm: 1, MaxSessions: 5, IdleTimeout: 30 * time.Minute, MaxLifetime: time.Hour})
	a, b := &visitor{srv: srv}, &visitor{srv: srv}
	a.get(t, "/", true)
	b.get(t, "/", true)
	g.mu.Lock()
	var dirA string
	for _, s := range g.sessions {
		if s.id == a.cookie.Value {
			dirA = s.child.Dir
		}
	}
	g.mu.Unlock()

	// b stays active, a goes idle
	for i := 0; i < 4; i++ {
		ck.add(10 * time.Minute)
		b.get(t, "/x", false)
	}
	g.Reap()
	if s, _ := g.counts(); s != 1 {
		t.Fatalf("after idle: %d sessions", s)
	}
	if _, err := os.Stat(dirA); !os.IsNotExist(err) {
		t.Fatalf("the idle sandbox's data is still there: %v", err)
	}
	if g.expiredIdle.Load() != 1 {
		t.Fatalf("idle count = %d", g.expiredIdle.Load())
	}
	// an expired visitor gets a fresh sandbox on the next page load
	old := a.cookie.Value
	if code, _ := a.get(t, "/", true); code != 200 || a.cookie.Value == old {
		t.Fatalf("revisit = %d, same cookie %v", code, a.cookie.Value == old)
	}

	// b reaches the hard limit even though it is active
	ck.add(25 * time.Minute)
	b.get(t, "/x", false)
	g.Reap()
	if g.expiredMax.Load() != 1 {
		t.Fatalf("max lifetime count = %d", g.expiredMax.Load())
	}
	sp.mu.Lock()
	stopped := len(sp.stopped)
	sp.mu.Unlock()
	if stopped != 2 {
		t.Fatalf("stopped = %d", stopped)
	}
	waitUntil(t, "pool refill", func() bool { _, w := g.counts(); return w == 1 })
}

func TestMaxSessionsAndPerIPLimit(t *testing.T) {
	_, trusted, _ := net.ParseCIDR("127.0.0.0/8")
	g, _, _, srv := newTestGateway(t, Config{MaxSessions: 3, SessionsPerIP: 2, TrustedProxies: []*net.IPNet{trusted}})

	// the client address comes from X-Forwarded-For because the peer (127.0.0.1) is a trusted proxy
	v1 := &visitor{srv: srv, ip: "203.0.113.7"}
	if _, body := v1.get(t, "/", true); !strings.Contains(body, "ip=203.0.113.7") {
		t.Fatalf("forwarded ip = %s", body)
	}
	v2 := &visitor{srv: srv, ip: "198.51.100.1, 203.0.113.7"} // a spoofed first hop is ignored
	v2.get(t, "/", true)
	v3 := &visitor{srv: srv, ip: "203.0.113.7", lang: "fr-FR,fr;q=0.9"}
	code, body := v3.get(t, "/", true)
	if code != http.StatusTooManyRequests || !strings.Contains(body, "Trop de démos") {
		t.Fatalf("third session from one ip = %d %.80s", code, body)
	}
	v4 := &visitor{srv: srv, ip: "192.0.2.10"}
	if code, _ := v4.get(t, "/", true); code != 200 {
		t.Fatalf("other ip = %d", code)
	}
	v5 := &visitor{srv: srv, ip: "192.0.2.11", lang: "en"}
	code, body = v5.get(t, "/", true)
	if code != http.StatusServiceUnavailable || !strings.Contains(body, "demo is full") {
		t.Fatalf("over the cap = %d %.80s", code, body)
	}
	if g.rejectedFull.Load() != 1 || g.rejectedIP.Load() != 1 {
		t.Fatalf("rejections full=%d ip=%d", g.rejectedFull.Load(), g.rejectedIP.Load())
	}
}

func TestUntrustedPeerCannotSpoofIP(t *testing.T) {
	g := NewGateway(Config{}, nil)
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "203.0.113.9:4000"
	r.Header.Set("X-Forwarded-For", "1.2.3.4")
	if ip := g.clientIP(r); ip != "203.0.113.9" {
		t.Fatalf("ip = %s", ip)
	}
}

func TestEventStreamIsProxiedLive(t *testing.T) {
	g, sp, ck, srv := newTestGateway(t, Config{MaxSessions: 2})
	v := &visitor{srv: srv}
	v.get(t, "/", true)

	req, _ := http.NewRequest("GET", srv.URL+"/api/v1/events", nil)
	req.AddCookie(v.cookie)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	got := make(chan string, 1)
	go func() {
		line, _ := bufio.NewReader(resp.Body).ReadString('\n')
		got <- line
	}()
	select {
	case line := <-got: // arrives while the sandbox still holds the stream open
		if strings.TrimSpace(line) != "data: hello" {
			t.Fatalf("first event = %q", line)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the event was buffered by the proxy")
	}
	// an open stream does not keep the session alive
	ck.add(31 * time.Minute)
	g.Reap()
	if s, _ := g.counts(); s != 0 {
		t.Fatalf("sessions = %d", s)
	}
	_ = sp
}

func TestRequestBodyLimit(t *testing.T) {
	_, _, _, srv := newTestGateway(t, Config{MaxSessions: 2, MaxBody: 1024})
	v := &visitor{srv: srv}
	v.get(t, "/", true)
	req, _ := http.NewRequest("POST", srv.URL+"/api/v1/me/avatar", strings.NewReader(strings.Repeat("x", 4096)))
	req.AddCookie(v.cookie)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode == 200 {
		t.Fatalf("a body over the limit went through")
	}
}

// app is a mobile client: no cookie, a Bearer token after sign-in.
type app struct {
	srv   *httptest.Server
	ip    string
	token string
}

func (a *app) do(t *testing.T, method, path, body string) (int, string) {
	t.Helper()
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	req, _ := http.NewRequest(method, a.srv.URL+path, rd)
	if a.ip != "" {
		req.Header.Set("X-Forwarded-For", a.ip)
	}
	if a.token != "" {
		req.Header.Set("Authorization", "Bearer "+a.token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func (a *app) signIn(t *testing.T, password string) (int, string) {
	t.Helper()
	code, body := a.do(t, "POST", "/api/v1/auth/token", `{"username":"admin","password":"`+password+`","platform":"ios"}`)
	if code == 200 {
		a.token = strings.Split(strings.Split(body, `"token":"`)[1], `"`)[0]
	}
	return code, body
}

func TestAppStatusWithoutSession(t *testing.T) {
	g, _, _, srv := newTestGateway(t, Config{Warm: 1, MaxSessions: 2})
	a := &app{srv: srv}
	// nothing running yet: a clear refusal, no sandbox
	if code, body := a.do(t, "GET", "/api/v1/status", ""); code != 503 || !strings.Contains(body, "demo_starting") {
		t.Fatalf("status with empty pool = %d %s", code, body)
	}
	g.Refill()
	waitUntil(t, "warm pool", func() bool { _, w := g.counts(); return w == 1 })
	for i := 0; i < 3; i++ {
		if code, body := a.do(t, "GET", "/api/v1/status", ""); code != 200 || !strings.Contains(body, `"demo":true`) {
			t.Fatalf("status = %d %s", code, body)
		}
	}
	if s, w := g.counts(); s != 0 || w != 1 {
		t.Fatalf("status allocated: sessions=%d warm=%d", s, w)
	}
}

func TestAppSignInAndBearerRouting(t *testing.T) {
	g, _, ck, srv := newTestGateway(t, Config{Warm: 1, MaxSessions: 3})
	g.Refill()
	waitUntil(t, "warm pool", func() bool { _, w := g.counts(); return w == 1 })
	a := &app{srv: srv}
	// a wrong password does not keep a sandbox
	if code, _ := a.signIn(t, "nope"); code != 401 {
		t.Fatalf("bad sign-in = %d", code)
	}
	if s, _ := g.counts(); s != 0 {
		t.Fatalf("failed sign-in kept %d sessions", s)
	}
	code, body := a.signIn(t, "demo")
	if code != 200 || a.token == "" {
		t.Fatalf("sign-in = %d %s", code, body)
	}
	sb := strings.TrimPrefix(a.token, "tok-")
	if s, _ := g.counts(); s != 1 {
		t.Fatalf("sessions = %d", s)
	}
	// later calls with the token reach the same sandbox
	if code, body := a.do(t, "GET", "/api/v1/me", ""); code != 200 || !strings.HasPrefix(body, sb+" ") {
		t.Fatalf("bearer call = %d %s, want %s", code, body, sb)
	}
	if code, body := a.do(t, "PUT", "/api/v1/me/devices/current", `{"pushToken":"x"}`); code != 200 || !strings.HasPrefix(body, sb+" ") {
		t.Fatalf("device registration = %d %s", code, body)
	}
	// an unknown token gets no sandbox
	other := &app{srv: srv, token: "forged"}
	if code, body := other.do(t, "GET", "/api/v1/me", ""); code != 401 || !strings.Contains(body, "demo_expired") {
		t.Fatalf("unknown token = %d %s", code, body)
	}
	// the event stream goes through with the token
	req, _ := http.NewRequest("GET", srv.URL+"/api/v1/events", nil)
	req.Header.Set("Authorization", "Bearer "+a.token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	line, _ := bufio.NewReader(resp.Body).ReadString('\n')
	resp.Body.Close()
	if strings.TrimSpace(line) != "data: hello" {
		t.Fatalf("event stream = %q", line)
	}
	// once the session ends, the token gets demo_expired so the app shows its sign-in screen
	ck.add(31 * time.Minute)
	g.Reap()
	if code, body := a.do(t, "GET", "/api/v1/me", ""); code != 401 || !strings.Contains(body, "demo_expired") {
		t.Fatalf("expired token = %d %s", code, body)
	}
	g.mu.Lock()
	left := len(g.tokens)
	g.mu.Unlock()
	if left != 0 {
		t.Fatalf("tokens kept after the session ended: %d", left)
	}
	// signing in again gets a new sandbox
	a.token = ""
	if code, _ := a.signIn(t, "demo"); code != 200 || strings.TrimPrefix(a.token, "tok-") == sb {
		t.Fatalf("second sign-in = %d token %s", code, a.token)
	}
}

func TestAppSignInPerIPLimit(t *testing.T) {
	_, trusted, _ := net.ParseCIDR("127.0.0.0/8")
	g, _, _, srv := newTestGateway(t, Config{MaxSessions: 5, SessionsPerIP: 2, TrustedProxies: []*net.IPNet{trusted}})
	for i := 0; i < 2; i++ {
		a := &app{srv: srv, ip: "203.0.113.7"}
		if code, _ := a.signIn(t, "demo"); code != 200 {
			t.Fatalf("sign-in %d = %d", i, code)
		}
	}
	a := &app{srv: srv, ip: "203.0.113.7"}
	if code, body := a.signIn(t, "demo"); code != http.StatusTooManyRequests || !strings.Contains(body, "demo_ip_limit") {
		t.Fatalf("third sign-in = %d %s", code, body)
	}
	if g.rejectedIP.Load() != 1 {
		t.Fatalf("ip rejections = %d", g.rejectedIP.Load())
	}
}
