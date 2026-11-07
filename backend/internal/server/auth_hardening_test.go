package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/Rombond/tipsarr/backend/internal/perms"
)

// A delegated ManageSettings cannot touch sign-in, directory or API key settings (only the admin role can).
func TestIdentitySettingsNeedAdminRole(t *testing.T) {
	e := newEnv(t, true)
	jf := fakeJellyfin(t)
	admin := loginAs(t, e.app, jf.URL, "alice", "secret")
	bob := loginAs(t, e.app, jf.URL, "bob", "hunter2")
	grantPermissions(t, e, admin, bob, perms.Default|perms.ManageSettings)
	for _, body := range []string{
		`{"oidcAdminGroup":"mine"}`, `{"oidcIssuer":""}`, `{"oidcClientSecret":"x"}`, `{"oidcUsernameClaim":"name"}`,
		`{"ldapUrl":""}`, `{"jellyfinApiKey":"x"}`, `{"apiKeyEnabled":true}`, `{"regenerateApiKey":true}`, `{"appPublicUrl":"https://evil.example"}`,
	} {
		if resp, _ := call(t, e.app, "PUT", "/api/v1/admin/settings", body, bob); resp.StatusCode != http.StatusForbidden {
			t.Errorf("%s by a delegate = %d", body, resp.StatusCode)
		}
	}
	// the rest of the settings stay delegable
	if resp, b := call(t, e.app, "PUT", "/api/v1/admin/settings", `{"boxofficeRegions":"FR"}`, bob); resp.StatusCode != 200 {
		t.Fatalf("plain setting by a delegate = %d %s", resp.StatusCode, b)
	}
	if resp, _ := call(t, e.app, "PUT", "/api/v1/admin/settings", `{"oidcAdminGroup":"admins"}`, admin); resp.StatusCode != 200 {
		t.Fatalf("admin = %d", resp.StatusCode)
	}
}

// An admin role that came from Jellyfin goes away when Jellyfin takes it back: at the next sign-in, and at
// the hourly session re-check. A role given by hand in Tipsarr stays.
func TestJellyfinAdminDemotion(t *testing.T) {
	defer jfAliceDemoted.Store(false)
	e := newEnv(t, true)
	jf := fakeJellyfin(t)
	alice := loginAs(t, e.app, jf.URL, "alice", "secret")
	call(t, e.app, "PUT", "/api/v1/admin/settings", `{"jellyfinApiKey":"jfkey"}`, alice)
	bob := loginAs(t, e.app, jf.URL, "bob", "hunter2")
	if _, b := call(t, e.app, "GET", "/api/v1/me", "", alice); !strings.Contains(b, `"role":"admin"`) {
		t.Fatalf("alice = %s", b)
	}

	// re-check (every request in tests): the live session loses the role
	jfAliceDemoted.Store(true)
	if _, b := call(t, e.app, "GET", "/api/v1/me", "", alice); !strings.Contains(b, `"role":"user"`) {
		t.Fatalf("after re-check = %s", b)
	}
	// sign-in agrees
	jfAliceDemoted.Store(false)
	alice = loginAs(t, e.app, jf.URL, "alice", "secret")
	if _, b := call(t, e.app, "GET", "/api/v1/me", "", alice); !strings.Contains(b, `"role":"admin"`) {
		t.Fatalf("promoted again = %s", b)
	}
	jfAliceDemoted.Store(true)
	resp, b := call(t, e.app, "POST", "/api/v1/auth/login", `{"username":"alice","password":"secret"}`)
	if resp.StatusCode != 200 || !strings.Contains(b, `"role":"user"`) {
		t.Fatalf("sign-in after Jellyfin demotion = %d %s", resp.StatusCode, b)
	}

	// bob made admin by hand in Tipsarr (by alice while she was admin) keeps it although Jellyfin says no
	jfAliceDemoted.Store(false)
	alice = loginAs(t, e.app, jf.URL, "alice", "secret")
	if resp, b := call(t, e.app, "PATCH", "/api/v1/admin/users/"+userID(t, e, bob), `{"role":"admin"}`, alice); resp.StatusCode != 200 {
		t.Fatalf("promote bob = %d %s", resp.StatusCode, b)
	}
	resp, b = call(t, e.app, "POST", "/api/v1/auth/login", `{"username":"bob","password":"hunter2"}`)
	if resp.StatusCode != 200 || !strings.Contains(b, `"role":"admin"`) {
		t.Fatalf("manual admin after sign-in = %s", b)
	}
	bob = sessionCookie(resp)
	if _, b := call(t, e.app, "GET", "/api/v1/me", "", bob); !strings.Contains(b, `"role":"admin"`) {
		t.Fatalf("manual admin after re-check = %s", b)
	}
}

// TIPSARR_COOKIE_SECURE forces the Secure flag even without X-Forwarded-Proto.
func TestCookieSecureForced(t *testing.T) {
	e := newEnvOpts(t, true, "", false)
	jf := fakeJellyfin(t)
	call(t, e.app, "POST", "/api/v1/setup", `{"jellyfinUrl":"`+jf.URL+`"}`)
	resp, _ := call(t, e.app, "POST", "/api/v1/auth/login", `{"username":"bob","password":"hunter2"}`)
	if c := sessionCookie(resp); c == nil || c.Secure {
		t.Fatalf("auto, plain http: secure must be off: %+v", c)
	}
	e2 := newEnvSecure(t)
	call(t, e2.app, "POST", "/api/v1/setup", `{"jellyfinUrl":"`+jf.URL+`"}`)
	resp, _ = call(t, e2.app, "POST", "/api/v1/auth/login", `{"username":"bob","password":"hunter2"}`)
	if !strings.Contains(resp.Header.Get("Set-Cookie"), "Secure") {
		t.Fatalf("forced: %s", resp.Header.Get("Set-Cookie"))
	}
}

// envSecureCookies: the next newEnv forces the Secure cookie flag (TIPSARR_COOKIE_SECURE=true).
var envSecureCookies bool

func newEnvSecure(t *testing.T) *env {
	envSecureCookies = true
	defer func() { envSecureCookies = false }()
	return newEnvOpts(t, true, "", false)
}
