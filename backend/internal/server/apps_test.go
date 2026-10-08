package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/Rombond/tipsarr/backend/internal/api"
	"github.com/Rombond/tipsarr/backend/internal/server"
	"github.com/danielgtaylor/huma/v2"
)

// bearerCall is call() with an Authorization header instead of a cookie.
func bearerCall(t *testing.T, base, method, path, body, token string) (*http.Response, string) {
	t.Helper()
	req, _ := http.NewRequest(method, base+path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var sb strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		sb.Write(buf[:n])
		if err != nil {
			break
		}
	}
	return resp, sb.String()
}

func appLogin(t *testing.T, base, user, pass, device string) string {
	t.Helper()
	body := `{"username":"` + user + `","password":"` + pass + `","platform":"ios","deviceName":"` + device + `","appVersion":"1.0.0"}`
	resp, out := bearerCall(t, base, "POST", "/api/v1/auth/token", body, "")
	if resp.StatusCode != 200 {
		t.Fatalf("token login = %d %s", resp.StatusCode, out)
	}
	if len(resp.Cookies()) != 0 {
		t.Fatalf("token login must not set a cookie: %v", resp.Cookies())
	}
	var tb struct {
		Token     string `json:"token"`
		ExpiresAt int64  `json:"expiresAt"`
		User      struct{ Name string }
	}
	if err := json.Unmarshal([]byte(out), &tb); err != nil || tb.Token == "" || tb.ExpiresAt == 0 || tb.User.Name != user {
		t.Fatalf("token body: %s (%v)", out, err)
	}
	return tb.Token
}

func TestBearerAndDeviceSessions(t *testing.T) {
	jf := fakeJellyfin(t)
	e := newEnv(t, true)
	loginAs(t, e.app, jf.URL, "alice", "secret") // runs setup
	base := e.app.URL

	// wrong password and bad body are refused, anonymous bearer is anonymous
	if resp, _ := bearerCall(t, base, "POST", "/api/v1/auth/token", `{"username":"bob","password":"nope","platform":"ios"}`, ""); resp.StatusCode != 401 {
		t.Fatalf("bad password = %d", resp.StatusCode)
	}
	if resp, _ := bearerCall(t, base, "POST", "/api/v1/auth/token", `{"username":"bob","password":"hunter2","platform":"plan9"}`, ""); resp.StatusCode != 422 {
		t.Fatalf("bad platform = %d", resp.StatusCode)
	}
	if resp, _ := bearerCall(t, base, "GET", "/api/v1/me", "", "garbage"); resp.StatusCode != 401 {
		t.Fatalf("garbage bearer = %d", resp.StatusCode)
	}

	phone := appLogin(t, base, "bob", "hunter2", "Bob's iPhone")
	ipad := appLogin(t, base, "bob", "hunter2", "Bob's iPad")
	alice := appLogin(t, base, "alice", "secret", "Alice's iPhone")

	// bearer works on JSON, on images and on the event stream's auth check
	if resp, body := bearerCall(t, base, "GET", "/api/v1/me", "", phone); resp.StatusCode != 200 || !strings.Contains(body, `"bob"`) {
		t.Fatalf("me = %d %s", resp.StatusCode, body)
	}
	if resp, _ := bearerCall(t, base, "GET", "/api/v1/images/jellyfin/xyz", "", ""); resp.StatusCode != 401 {
		t.Fatalf("image without login = %d", resp.StatusCode)
	}
	if resp, _ := bearerCall(t, base, "GET", "/api/v1/images/jellyfin/xyz", "", phone); resp.StatusCode == 401 {
		t.Fatalf("image with bearer must pass the login check")
	}

	// the devices list shows both, marks the current one, never leaks the token hash
	resp, body := bearerCall(t, base, "GET", "/api/v1/me/sessions", "", phone)
	if resp.StatusCode != 200 {
		t.Fatalf("sessions = %d %s", resp.StatusCode, body)
	}
	var list []struct {
		ID, Platform, DeviceName, AppVersion string
		Current                              bool
	}
	if err := json.Unmarshal([]byte(body), &list); err != nil || len(list) != 2 {
		t.Fatalf("sessions body: %s (%v)", body, err)
	}
	var padID string
	for _, s := range list {
		if len(s.ID) != 16 || s.Platform != "ios" || s.AppVersion != "1.0.0" {
			t.Fatalf("session row: %+v", s)
		}
		if s.Current != (s.DeviceName == "Bob's iPhone") {
			t.Fatalf("current flag wrong: %+v", s)
		}
		if s.DeviceName == "Bob's iPad" {
			padID = s.ID
		}
	}

	// another user cannot revoke Bob's session
	if resp, _ := bearerCall(t, base, "DELETE", "/api/v1/me/sessions/"+padID, "", alice); resp.StatusCode != 404 {
		t.Fatalf("revoke other user's session = %d", resp.StatusCode)
	}
	if resp, _ := bearerCall(t, base, "GET", "/api/v1/me", "", ipad); resp.StatusCode != 200 {
		t.Fatalf("ipad should still work: %d", resp.StatusCode)
	}
	// Bob revokes the iPad from the iPhone
	if resp, _ := bearerCall(t, base, "DELETE", "/api/v1/me/sessions/"+padID, "", phone); resp.StatusCode != 204 {
		t.Fatalf("revoke = %d", resp.StatusCode)
	}
	if resp, _ := bearerCall(t, base, "GET", "/api/v1/me", "", ipad); resp.StatusCode != 401 {
		t.Fatalf("revoked ipad = %d", resp.StatusCode)
	}

	// "sign out everywhere else" keeps the caller
	web := loginAs(t, e.app, jf.URL, "bob", "hunter2")
	if resp, _ := bearerCall(t, base, "DELETE", "/api/v1/me/sessions", "", phone); resp.StatusCode != 204 {
		t.Fatalf("revoke others = %d", resp.StatusCode)
	}
	if resp, _ := call(t, e.app, "GET", "/api/v1/me", "", web); resp.StatusCode != 401 {
		t.Fatalf("web cookie should be gone: %d", resp.StatusCode)
	}
	if resp, _ := bearerCall(t, base, "GET", "/api/v1/me", "", phone); resp.StatusCode != 200 {
		t.Fatalf("caller must survive: %d", resp.StatusCode)
	}
	if resp, _ := bearerCall(t, base, "GET", "/api/v1/me", "", alice); resp.StatusCode != 200 {
		t.Fatalf("alice untouched: %d", resp.StatusCode)
	}

	// logout with a bearer ends that session only
	if resp, _ := bearerCall(t, base, "POST", "/api/v1/auth/logout", "", phone); resp.StatusCode != 204 {
		t.Fatalf("logout = %d", resp.StatusCode)
	}
	if resp, _ := bearerCall(t, base, "GET", "/api/v1/me", "", phone); resp.StatusCode != 401 {
		t.Fatalf("after logout = %d", resp.StatusCode)
	}
}

func TestSessionSlidingRenewal(t *testing.T) {
	jf := fakeJellyfin(t)
	e := newEnv(t, true)
	loginAs(t, e.app, jf.URL, "alice", "secret")
	base := e.app.URL
	tok := appLogin(t, base, "bob", "hunter2", "iPhone")
	ctx := context.Background()

	get := func() (last, exp int64) {
		var row struct{ LastSeen, Exp int64 }
		if err := e.st.DB.NewSelect().TableExpr("sessions").
			ColumnExpr("last_seen_at AS last_seen, expires_at AS exp").Where("platform = 'ios'").Scan(ctx, &row.LastSeen, &row.Exp); err != nil {
			t.Fatal(err)
		}
		return row.LastSeen, row.Exp
	}
	last0, exp0 := get()

	// a use right away does not write
	bearerCall(t, base, "GET", "/api/v1/me", "", tok)
	if last, exp := get(); last != last0 || exp != exp0 {
		t.Fatalf("renewed too early: %d/%d -> %d/%d", last0, exp0, last, exp)
	}

	// an hour and a half later, nearly expired: the use renews it
	old, nearEnd := last0-5400, last0+3600
	if _, err := e.st.DB.ExecContext(ctx, `UPDATE sessions SET last_seen_at = ?, expires_at = ? WHERE platform = 'ios'`, old, nearEnd); err != nil {
		t.Fatal(err)
	}
	if resp, _ := bearerCall(t, base, "GET", "/api/v1/me", "", tok); resp.StatusCode != 200 {
		t.Fatalf("near-expiry use = %d", resp.StatusCode)
	}
	if last, exp := get(); last < last0 || exp < exp0 {
		t.Fatalf("not renewed: last %d exp %d (want >= %d / %d)", last, exp, last0, exp0)
	}

	// a session past its expiry stays dead
	if _, err := e.st.DB.ExecContext(ctx, `UPDATE sessions SET expires_at = ? WHERE platform = 'ios'`, last0-10); err != nil {
		t.Fatal(err)
	}
	if resp, _ := bearerCall(t, base, "GET", "/api/v1/me", "", tok); resp.StatusCode != 401 {
		t.Fatalf("expired = %d", resp.StatusCode)
	}
}

func TestWebCookieRenewedWhenUsed(t *testing.T) {
	jf := fakeJellyfin(t)
	e := newEnv(t, true)
	web := loginAs(t, e.app, jf.URL, "alice", "secret")
	if _, err := e.st.DB.ExecContext(context.Background(), `UPDATE sessions SET last_seen_at = last_seen_at - 7200`); err != nil {
		t.Fatal(err)
	}
	resp, _ := call(t, e.app, "GET", "/api/v1/me", "", web)
	c := sessionCookie(resp)
	if resp.StatusCode != 200 || c == nil || c.Value != web.Value || c.MaxAge <= 0 {
		t.Fatalf("renewed cookie missing: %d %+v", resp.StatusCode, c)
	}
}

// TestAppContract signs in the way an app does and walks the endpoints the v1 screens use, so a
// change that breaks bearer auth or one of those responses is caught here before an app is.
func TestAppContract(t *testing.T) {
	jf := fakeJellyfin(t)
	e := newEnv(t, true)
	admin := loginAs(t, e.app, jf.URL, "alice", "secret")
	if resp, _ := call(t, e.app, "PUT", "/api/v1/admin/settings", `{"tmdbApiKey":"k123"}`, admin); resp.StatusCode != 200 {
		t.Fatalf("settings = %d", resp.StatusCode)
	}
	base := e.app.URL
	user := appLogin(t, base, "bob", "hunter2", "iPhone")
	adm := appLogin(t, base, "alice", "secret", "iPad")

	// status is public and tells an app what it is talking to
	resp, body := bearerCall(t, base, "GET", "/api/v1/status", "", "")
	var st struct {
		APIVersion    int
		MinAppVersion *string
		Features      *struct{ Push *bool }
	}
	if err := json.Unmarshal([]byte(body), &st); resp.StatusCode != 200 || err != nil || st.APIVersion != 1 || st.MinAppVersion == nil || st.Features == nil || st.Features.Push == nil {
		t.Fatalf("status = %d %s", resp.StatusCode, body)
	}

	for _, path := range []string{
		"/me", "/me/sessions", "/discover/trending", "/media/movie/1", "/media/tv/2", "/search?q=movie",
		"/requests", "/requests/counts", "/issues", "/issues/counts",
		"/library", "/library/facets", "/suggestions", "/watchlist", "/blocklist", "/stats", "/auth/methods",
	} {
		resp, body := bearerCall(t, base, "GET", "/api/v1"+path, "", user)
		if resp.StatusCode != 200 || !json.Valid([]byte(body)) {
			t.Errorf("GET %s as user = %d %.200s", path, resp.StatusCode, body)
		}
	}
	for _, path := range []string{"/admin/sync", "/admin/users", "/admin/settings"} {
		if resp, _ := bearerCall(t, base, "GET", "/api/v1"+path, "", user); resp.StatusCode != 403 {
			t.Errorf("GET %s as user = %d, want 403", path, resp.StatusCode)
		}
		if resp, body := bearerCall(t, base, "GET", "/api/v1"+path, "", adm); resp.StatusCode != 200 || !json.Valid([]byte(body)) {
			t.Errorf("GET %s as admin = %d %.200s", path, resp.StatusCode, body)
		}
	}

	// no Radarr/Sonarr yet: a clear, coded refusal (the app shows its own text)
	if resp, body := bearerCall(t, base, "GET", "/api/v1/requests/options?type=movie", "", user); resp.StatusCode != 409 || !strings.Contains(body, "no_instance") {
		t.Errorf("options without instance = %d %.200s", resp.StatusCode, body)
	}

	// every error an app must localize carries a machine-readable code
	resp, body = bearerCall(t, base, "GET", "/api/v1/media/movie/404", "", user)
	if resp.StatusCode != 404 || !strings.Contains(body, `"code"`) {
		t.Errorf("error shape = %d %s", resp.StatusCode, body)
	}
	resp, body = bearerCall(t, base, "GET", "/api/v1/me", "", "")
	if resp.StatusCode != 401 || !strings.Contains(body, "login_required") {
		t.Errorf("anon shape = %d %s", resp.StatusCode, body)
	}
}

// TestSpecEveryProtectedOperationAcceptsBearer keeps the generated mobile spec honest: an operation
// that needs a login must list the bearer scheme next to the cookie one.
func TestSpecEveryProtectedOperationAcceptsBearer(t *testing.T) {
	_, a := server.New(api.Deps{})
	n := 0
	for path, item := range a.OpenAPI().Paths {
		for method, op := range map[string]*huma.Operation{"GET": item.Get, "POST": item.Post, "PUT": item.Put, "PATCH": item.Patch, "DELETE": item.Delete} {
			if op == nil || len(op.Security) == 0 {
				continue
			}
			n++
			ok := false
			for _, req := range op.Security {
				if _, has := req["bearer"]; has {
					ok = true
				}
			}
			if !ok {
				t.Errorf("%s %s: security lists no bearer scheme", method, path)
			}
		}
	}
	if n < 50 {
		t.Fatalf("only %d protected operations found, the spec walk is broken", n)
	}
}
