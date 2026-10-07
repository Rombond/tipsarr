package server_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

// fakeIdP is a minimal OpenID Connect provider (Authelia stand-in).
type fakeIdP struct {
	srv      *httptest.Server
	mu       sync.Mutex
	nonce    string // set by the test from the authorization redirect
	username string
	groups   []string
	rejected bool // the token endpoint answers 400
	// profile claims only through the userinfo endpoint, like Authelia by default
	userinfoOnly bool
}

func newFakeIdP(t *testing.T) *fakeIdP {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwk := jose.JSONWebKey{Key: &key.PublicKey, KeyID: "k1", Algorithm: "RS256", Use: "sig"}
	signer, _ := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, (&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "k1"))
	f := &fakeIdP{}
	mux := http.NewServeMux()
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer": f.srv.URL, "authorization_endpoint": f.srv.URL + "/authorize", "token_endpoint": f.srv.URL + "/token",
			"jwks_uri": f.srv.URL + "/jwks", "userinfo_endpoint": f.srv.URL + "/userinfo", "id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		_ = r.ParseForm()
		if f.rejected || r.Form.Get("code_verifier") == "" || r.Form.Get("grant_type") != "authorization_code" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		claims := map[string]any{"preferred_username": f.username, "groups": f.groups, "nonce": f.nonce}
		if f.userinfoOnly {
			claims = map[string]any{"nonce": f.nonce}
		}
		std := jwt.Claims{Issuer: f.srv.URL, Subject: "sub-" + f.username, Audience: jwt.Audience{"tipsarr"}, IssuedAt: jwt.NewNumericDate(time.Now()), Expiry: jwt.NewNumericDate(time.Now().Add(time.Hour))}
		raw, err := jwt.Signed(signer).Claims(std).Claims(claims).Serialize()
		if err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "at", "token_type": "Bearer", "id_token": raw})
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if r.Header.Get("Authorization") != "Bearer at" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"sub": "sub-" + f.username, "preferred_username": f.username, "groups": f.groups})
	})
	return f
}

// noFollow talks to the app without following redirects.
func noFollow(t *testing.T, app *httptest.Server, path string, cookies ...*http.Cookie) *http.Response {
	t.Helper()
	req, _ := http.NewRequest("GET", app.URL+path, nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	resp, err := (&http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp
}

// ssoLogin runs the whole redirect dance and returns the callback response.
func ssoLogin(t *testing.T, e *env, idp *fakeIdP, username string, groups []string) *http.Response {
	t.Helper()
	resp := noFollow(t, e.app, "/api/v1/auth/oidc/login")
	loc, err := url.Parse(resp.Header.Get("Location"))
	if resp.StatusCode != http.StatusFound || err != nil || !strings.HasPrefix(loc.String(), idp.srv.URL+"/authorize") {
		t.Fatalf("login = %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	q := loc.Query()
	if q.Get("code_challenge") == "" || q.Get("code_challenge_method") != "S256" || q.Get("state") == "" || q.Get("nonce") == "" {
		t.Fatalf("authorization request lacks PKCE/state/nonce: %v", q)
	}
	idp.mu.Lock()
	idp.nonce, idp.username, idp.groups = q.Get("nonce"), username, groups
	idp.mu.Unlock()
	var cookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "tipsarr_sso" {
			cookie = c
		}
	}
	if cookie == nil {
		t.Fatal("no sso cookie")
	}
	return noFollow(t, e.app, "/api/v1/auth/oidc/callback?code=abc&state="+url.QueryEscape(q.Get("state")), cookie)
}

func TestSingleSignOn(t *testing.T) {
	e := newEnv(t, true)
	jf := fakeJellyfin(t)
	admin := loginAs(t, e.app, jf.URL, "alice", "secret")
	idp := newFakeIdP(t)

	// off by default: password only
	if _, b := call(t, e.app, "GET", "/api/v1/auth/methods", ""); !strings.Contains(b, `"sso":false`) {
		t.Fatalf("methods = %s", b)
	}
	if resp := noFollow(t, e.app, "/api/v1/auth/oidc/login"); !strings.Contains(resp.Header.Get("Location"), "sso=refused") {
		t.Fatalf("login while off = %s", resp.Header.Get("Location"))
	}

	// admin settings: the provider is checked when saving
	if resp, _ := call(t, e.app, "PUT", "/api/v1/admin/settings", `{"oidcIssuer":"http://127.0.0.1:1"}`, admin); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("unreachable provider = %d", resp.StatusCode)
	}
	call(t, e.app, "PUT", "/api/v1/admin/settings", `{"jellyfinApiKey":"jfkey"}`, admin)
	resp, body := call(t, e.app, "PUT", "/api/v1/admin/settings",
		`{"oidcIssuer":"`+idp.srv.URL+`","oidcClientId":"tipsarr","oidcClientSecret":"s3cret","oidcAdminGroup":"tipsarr-admins"}`, admin)
	if resp.StatusCode != 200 || !strings.Contains(body, `"oidcClientSecretConfigured":true`) || strings.Contains(body, "s3cret") {
		t.Fatalf("save sso = %d %s", resp.StatusCode, body)
	}
	if _, b := call(t, e.app, "GET", "/api/v1/auth/methods", ""); !strings.Contains(b, `"sso":true`) {
		t.Fatalf("methods = %s", b)
	}

	// a Jellyfin user signs in, with no password
	cb := ssoLogin(t, e, idp, "Bob", nil) // the match ignores case
	var session *http.Cookie
	for _, c := range cb.Cookies() {
		if c.Name == "tipsarr_session" {
			session = c
		}
	}
	if cb.StatusCode != http.StatusFound || cb.Header.Get("Location") != "/" || session == nil {
		t.Fatalf("callback = %d %s", cb.StatusCode, cb.Header.Get("Location"))
	}
	if _, b := call(t, e.app, "GET", "/api/v1/me", "", session); !strings.Contains(b, `"name":"bob"`) || !strings.Contains(b, `"role":"user"`) {
		t.Fatalf("me = %s", b)
	}
	// the group claim grants admin
	cb = ssoLogin(t, e, idp, "carol", []string{"users", "tipsarr-admins"})
	for _, c := range cb.Cookies() {
		if c.Name == "tipsarr_session" {
			session = c
		}
	}
	if _, b := call(t, e.app, "GET", "/api/v1/me", "", session); !strings.Contains(b, `"name":"carol"`) || !strings.Contains(b, `"role":"admin"`) {
		t.Fatalf("carol = %s", b)
	}

	// providers that keep profile claims out of the ID token (Authelia): they come from userinfo
	idp.mu.Lock()
	idp.userinfoOnly = true
	idp.mu.Unlock()
	cb = ssoLogin(t, e, idp, "bob", []string{"users"})
	if cb.StatusCode != http.StatusFound || cb.Header.Get("Location") != "/" {
		t.Fatalf("userinfo-only provider = %d %s", cb.StatusCode, cb.Header.Get("Location"))
	}
	idp.mu.Lock()
	idp.userinfoOnly = false
	idp.mu.Unlock()

	// refused: no Jellyfin account with that name -> back to the login page (classic form)
	cb = ssoLogin(t, e, idp, "mallory", nil)
	if loc := cb.Header.Get("Location"); cb.StatusCode != http.StatusFound || !strings.Contains(loc, "/login?sso=refused&reason=no_user") {
		t.Fatalf("unknown user = %d %s", cb.StatusCode, loc)
	}
	for _, c := range cb.Cookies() {
		if c.Name == "tipsarr_session" && c.Value != "" {
			t.Fatal("a refused sign-in must not start a session")
		}
	}
	// refused: the provider rejects the code
	idp.mu.Lock()
	idp.rejected = true
	idp.mu.Unlock()
	if cb = ssoLogin(t, e, idp, "bob", nil); !strings.Contains(cb.Header.Get("Location"), "reason=provider") {
		t.Fatalf("provider failure = %s", cb.Header.Get("Location"))
	}
	// refused: a callback that this browser did not start (no cookie / wrong state)
	if r := noFollow(t, e.app, "/api/v1/auth/oidc/callback?code=x&state=forged"); !strings.Contains(r.Header.Get("Location"), "reason=state") {
		t.Fatalf("forged callback = %s", r.Header.Get("Location"))
	}
	if r := noFollow(t, e.app, "/api/v1/auth/oidc/callback?error=access_denied"); !strings.Contains(r.Header.Get("Location"), "reason=denied") {
		t.Fatalf("provider denial = %s", r.Header.Get("Location"))
	}

	// the password form keeps working underneath
	if resp, _ := call(t, e.app, "POST", "/api/v1/auth/login", `{"username":"bob","password":"hunter2"}`); resp.StatusCode != 200 {
		t.Fatalf("password login = %d", resp.StatusCode)
	}
}
