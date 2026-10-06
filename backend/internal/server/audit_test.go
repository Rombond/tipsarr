package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSetupNeedsTheToken(t *testing.T) {
	e := newEnvWith(t, true, "tok-123")
	jf := fakeJellyfin(t)
	body := func(tok string) string { return `{"setupToken":"` + tok + `","jellyfinUrl":"` + jf.URL + `"}` }

	if resp, _ := call(t, e.app, "POST", "/api/v1/setup", `{"jellyfinUrl":"`+jf.URL+`"}`); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no token = %d", resp.StatusCode)
	}
	if resp, _ := call(t, e.app, "POST", "/api/v1/setup", body("wrong")); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong token = %d", resp.StatusCode)
	}
	if _, b := call(t, e.app, "GET", "/api/v1/setup/status", ""); !strings.Contains(b, `"configured":false`) {
		t.Fatalf("a rejected setup must not configure anything: %s", b)
	}
	// a Jellyfin URL that cannot be reached gives a generic message (no dial details)
	resp, b := call(t, e.app, "POST", "/api/v1/setup", `{"setupToken":"tok-123","jellyfinUrl":"http://127.0.0.1:1"}`)
	if resp.StatusCode != http.StatusUnprocessableEntity || strings.Contains(b, "dial") || strings.Contains(b, "127.0.0.1") {
		t.Fatalf("unreachable = %d %s", resp.StatusCode, b)
	}

	// concurrent claims: exactly one wins
	var wins, conflicts atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, _ := call(t, e.app, "POST", "/api/v1/setup", body("tok-123"))
			switch resp.StatusCode {
			case 200:
				wins.Add(1)
			case http.StatusConflict:
				conflicts.Add(1)
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 || conflicts.Load() != 5 {
		t.Fatalf("wins=%d conflicts=%d", wins.Load(), conflicts.Load())
	}
}

func TestSecurityHeadersAndLimits(t *testing.T) {
	e := newEnv(t, true)
	_, _, bob := setupUsers(t, e)

	resp, _ := call(t, e.app, "GET", "/api/v1/health", "")
	for _, h := range []string{"X-Content-Type-Options", "X-Frame-Options", "Content-Security-Policy", "Referrer-Policy"} {
		if resp.Header.Get(h) == "" {
			t.Fatalf("missing header %s", h)
		}
	}
	if !strings.Contains(resp.Header.Get("Content-Security-Policy"), "frame-ancestors 'none'") {
		t.Fatal("CSP must forbid framing")
	}
	// absurd page numbers are refused before any TMDB call
	before := tmdbHits.Load()
	if resp, _ := call(t, e.app, "GET", "/api/v1/discover/trending?page=501", "", bob); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("page 501 = %d", resp.StatusCode)
	}
	if tmdbHits.Load() != before {
		t.Fatal("rejected request still reached TMDB")
	}
	// svg is no longer served by the image proxy
	if resp, _ := call(t, e.app, "GET", "/api/v1/images/tmdb/w185/AbCdEf123456.svg", "", bob); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("svg = %d", resp.StatusCode)
	}
	// secure cookies behind an https proxy
	jf := fakeJellyfin(t)
	req, _ := http.NewRequest("POST", e.app.URL+"/api/v1/auth/login", strings.NewReader(`{"username":"bob","password":"hunter2"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-Proto", "https")
	_ = jf
	r2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	r2.Body.Close()
	if c := sessionCookie(r2); c == nil || !c.Secure || !c.HttpOnly {
		t.Fatalf("cookie = %+v", c)
	}
}

func TestStreamsPerUserAreCapped(t *testing.T) {
	e := newEnv(t, true)
	_, _, bob := setupUsers(t, e)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	open := func() int {
		req, _ := http.NewRequestWithContext(ctx, "GET", e.app.URL+"/api/v1/events", nil)
		req.AddCookie(bob)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != 200 {
			resp.Body.Close()
		}
		return resp.StatusCode
	}
	for i := 0; i < 6; i++ {
		if code := open(); code != 200 {
			t.Fatalf("stream %d = %d", i, code)
		}
	}
	if code := open(); code != http.StatusTooManyRequests {
		t.Fatalf("7th stream = %d", code)
	}
}

func TestRequestErrorsAreAdminOnly(t *testing.T) {
	e := newEnv(t, false)
	_, admin, bob := setupUsers(t, e)
	radarr := newFakeArr(t, "radarr")
	addInstance(t, e, admin, "radarr", radarr)
	radarr.failPost.Store(true)

	_, body := call(t, e.app, "POST", "/api/v1/requests", `{"type":"movie","tmdbId":5}`, bob)
	v := decodeReq(t, body)
	call(t, e.app, "POST", "/api/v1/requests/"+v.ID+"/approve", `{}`, admin)

	_, adminView := call(t, e.app, "GET", "/api/v1/requests/"+v.ID, "", admin)
	if !strings.Contains(adminView, "boom") {
		t.Fatalf("admin must see the details: %s", adminView)
	}
	_, userView := call(t, e.app, "GET", "/api/v1/requests/"+v.ID, "", bob)
	if strings.Contains(userView, "boom") || strings.Contains(userView, radarr.srv.URL) || !strings.Contains(userView, `"status":"failed"`) {
		t.Fatalf("user must not see internals: %s", userView)
	}
	_, list := call(t, e.app, "GET", "/api/v1/requests", "", bob)
	if strings.Contains(list, "boom") {
		t.Fatalf("list leaks: %s", list)
	}
}

func TestProbeDoesNotSendTheSavedKeyElsewhere(t *testing.T) {
	e := newEnv(t, true)
	_, admin, _ := setupUsers(t, e)
	radarr := newFakeArr(t, "radarr")
	attacker := newFakeArr(t, "radarr")
	addInstance(t, e, admin, "radarr", radarr)
	_, body := call(t, e.app, "GET", "/api/v1/admin/servarr", "", admin)
	id := strings.Split(strings.Split(body, `"id":"`)[1], `"`)[0]

	resp, _ := call(t, e.app, "POST", "/api/v1/admin/servarr/probe", `{"kind":"radarr","url":"`+attacker.srv.URL+`","id":"`+id+`"}`, admin)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("probe to another host with the saved key = %d", resp.StatusCode)
	}
	attacker.mu.Lock()
	n := len(attacker.calls)
	attacker.mu.Unlock()
	if n != 0 {
		t.Fatalf("the other host received %d calls", n)
	}
	// the same host still works without retyping the key
	if resp, _ := call(t, e.app, "POST", "/api/v1/admin/servarr/probe", `{"kind":"radarr","url":"`+radarr.srv.URL+`","id":"`+id+`"}`, admin); resp.StatusCode != 200 {
		t.Fatalf("same-host probe = %d", resp.StatusCode)
	}
}

func TestWebhookCannotTargetRadarr(t *testing.T) {
	e := newEnv(t, false)
	_, admin, _ := setupUsers(t, e)
	radarr := newFakeArr(t, "radarr")
	addInstance(t, e, admin, "radarr", radarr)
	before := len(radarr.writes())

	resp, body := call(t, e.app, "POST", "/api/v1/admin/webhooks", `{"name":"evil","url":"`+radarr.srv.URL+`/api/v3/movie","enabled":true}`, admin)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create = %d %s", resp.StatusCode, body)
	}
	id := strings.Split(strings.Split(body, `"id":"`)[1], `"`)[0]
	if resp, _ := call(t, e.app, "POST", "/api/v1/admin/webhooks/"+id+"/test", "", admin); resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("test delivery = %d", resp.StatusCode)
	}
	call(t, e.app, "POST", "/api/v1/requests", `{"type":"movie","tmdbId":5}`, admin) // would fire request.* events
	time.Sleep(200 * time.Millisecond)
	if after := len(radarr.writes()); after != before+1 { // exactly the one legitimate add from the request above
		t.Fatalf("the webhook wrote to Radarr: %d -> %d", before, after)
	}
	for _, c := range radarr.writes() {
		if !strings.HasPrefix(c, "POST /movie") {
			t.Fatalf("unexpected write %s", c)
		}
	}
}

func TestRoleChangeEndsSessions(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)
	_, body := call(t, e.app, "GET", "/api/v1/admin/users", "", admin)
	var bobID string
	for _, part := range strings.Split(body, `{"id":"`)[1:] {
		if strings.Contains(strings.SplitN(part, "}", 2)[0], `"name":"bob"`) {
			bobID = strings.SplitN(part, `"`, 2)[0]
		}
	}
	if resp, _ := call(t, e.app, "GET", "/api/v1/me", "", bob); resp.StatusCode != 200 {
		t.Fatalf("bob before = %d", resp.StatusCode)
	}
	if resp, _ := call(t, e.app, "PATCH", "/api/v1/admin/users/"+bobID, `{"role":"admin"}`, admin); resp.StatusCode != 200 {
		t.Fatalf("promote = %d", resp.StatusCode)
	}
	if resp, _ := call(t, e.app, "GET", "/api/v1/me", "", bob); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("old session must end after a role change, got %d", resp.StatusCode)
	}
}

func TestSetupTokenIsOptional(t *testing.T) {
	// default: no token, the setup call works as before and the UI is told not to ask for one
	e := newEnv(t, true)
	if _, b := call(t, e.app, "GET", "/api/v1/setup/status", ""); !strings.Contains(b, `"tokenRequired":false`) {
		t.Fatalf("status = %s", b)
	}
	jf := fakeJellyfin(t)
	if resp, _ := call(t, e.app, "POST", "/api/v1/setup", `{"jellyfinUrl":"`+jf.URL+`"}`); resp.StatusCode != 200 {
		t.Fatalf("setup without token = %d", resp.StatusCode)
	}
	// opt-in: the status says a token is needed
	e2 := newEnvWith(t, true, "tok")
	if _, b := call(t, e2.app, "GET", "/api/v1/setup/status", ""); !strings.Contains(b, `"tokenRequired":true`) {
		t.Fatalf("status = %s", b)
	}
}

// Errors carry a stable code next to the English text so the UI can translate them.
func TestErrorsCarryCodes(t *testing.T) {
	e := newEnv(t, true)
	jf := fakeJellyfin(t)
	call(t, e.app, "POST", "/api/v1/setup", `{"jellyfinUrl":"`+jf.URL+`"}`)

	code := func(body string) string {
		var v struct {
			Errors []struct {
				Location string `json:"location"`
				Value    string `json:"value"`
			} `json:"errors"`
		}
		_ = json.Unmarshal([]byte(body), &v)
		for _, d := range v.Errors {
			if d.Location == "code" {
				return d.Value
			}
		}
		return ""
	}
	resp, body := call(t, e.app, "POST", "/api/v1/auth/login", `{"username":"alice","password":"wrong"}`)
	if resp.StatusCode != 401 || code(body) != "invalid_credentials" {
		t.Fatalf("login = %d %s", resp.StatusCode, body)
	}
	if resp, body := call(t, e.app, "GET", "/api/v1/me", ""); resp.StatusCode != 401 || code(body) != "login_required" {
		t.Fatalf("me = %d %s", resp.StatusCode, body)
	}
	c := loginAs(t, e.app, jf.URL, "bob", "hunter2")
	if resp, body := call(t, e.app, "GET", "/api/v1/admin/users", "", c); resp.StatusCode != 403 || code(body) != "admin_only" {
		t.Fatalf("admin = %d %s", resp.StatusCode, body)
	}
	call(t, e.app, "PUT", "/api/v1/admin/settings", `{"tmdbApiKey":"k123"}`, loginAs(t, e.app, jf.URL, "alice", "secret"))
	call(t, e.app, "POST", "/api/v1/requests", `{"type":"movie","tmdbId":5}`, c)
	if resp, body := call(t, e.app, "POST", "/api/v1/requests", `{"type":"movie","tmdbId":5}`, c); resp.StatusCode != 409 || code(body) != "duplicate_request" {
		t.Fatalf("duplicate = %d %s", resp.StatusCode, body)
	}
}
