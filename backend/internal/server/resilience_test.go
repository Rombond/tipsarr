package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// A session ends once its Jellyfin account is disabled or deleted (checked against Jellyfin).
func TestSessionEndsWhenJellyfinAccountDisabled(t *testing.T) {
	jf := fakeJellyfin(t)
	app := newApp(t)
	admin := loginAs(t, app, jf.URL, "alice", "secret")
	if resp, body := call(t, app, "PUT", "/api/v1/admin/settings", `{"jellyfinApiKey":"jfkey"}`, admin); resp.StatusCode != 200 {
		t.Fatalf("settings = %d %s", resp.StatusCode, body)
	}
	bob := loginAs(t, app, jf.URL, "bob", "hunter2")
	if resp, body := call(t, app, "GET", "/api/v1/me", "", bob); resp.StatusCode != 200 {
		t.Fatalf("me before = %d %s", resp.StatusCode, body)
	}

	jfDisabled.Store("bbbbbbbbbbbbccccddddeeeeeeeeeeee")
	t.Cleanup(func() { jfDisabled.Store("") })
	if resp, _ := call(t, app, "GET", "/api/v1/me", "", bob); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("disabled account kept its session: %d", resp.StatusCode)
	}
	// enabling the account again does not bring the old session back
	jfDisabled.Store("")
	if resp, _ := call(t, app, "GET", "/api/v1/me", "", bob); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("session came back: %d", resp.StatusCode)
	}
	// alice (not disabled) is untouched
	if resp, _ := call(t, app, "GET", "/api/v1/me", "", admin); resp.StatusCode != 200 {
		t.Fatalf("alice = %d", resp.StatusCode)
	}
}

// When Jellyfin cannot be reached the session keeps working (fail open).
func TestSessionSurvivesJellyfinOutage(t *testing.T) {
	jf := fakeJellyfin(t)
	app := newApp(t)
	admin := loginAs(t, app, jf.URL, "alice", "secret")
	call(t, app, "PUT", "/api/v1/admin/settings", `{"jellyfinApiKey":"wrongkey"}`, admin) // fake answers 401 to it
	if resp, _ := call(t, app, "GET", "/api/v1/me", "", admin); resp.StatusCode != 200 {
		t.Fatalf("me with a failing Jellyfin check = %d", resp.StatusCode)
	}
}

// Linking Jellyfin prefills the LDAP settings from its LDAP plugin; the button can redo it.
func TestLDAPImportFromJellyfin(t *testing.T) {
	jf := fakeJellyfin(t)
	app := newApp(t)
	admin := loginAs(t, app, jf.URL, "alice", "secret")
	bob := loginAs(t, app, jf.URL, "bob", "hunter2")

	if resp, _ := call(t, app, "POST", "/api/v1/admin/ldap/import-jellyfin", "", admin); resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("import without API key = %d", resp.StatusCode)
	}
	resp, body := call(t, app, "PUT", "/api/v1/admin/settings", `{"jellyfinApiKey":"jfkey"}`, admin)
	if resp.StatusCode != 200 || !strings.Contains(body, `"ldapUrl":"ldap://127.0.0.1:1"`) || !strings.Contains(body, `"ldapBindPasswordConfigured":true`) || strings.Contains(body, `"pw"`) {
		t.Fatalf("no prefill after linking: %d %s", resp.StatusCode, body)
	}
	// the button overwrites, reports the connection test, and is admin only
	call(t, app, "PUT", "/api/v1/admin/settings", `{"ldapBaseDn":"ou=other"}`, admin)
	resp, body = call(t, app, "POST", "/api/v1/admin/ldap/import-jellyfin", "", admin)
	if resp.StatusCode != 200 || !strings.Contains(body, `"ldapBaseDn":"ou=people,dc=x,dc=com"`) || !strings.Contains(body, `"connected":false`) || strings.Contains(body, `"pw"`) {
		t.Fatalf("import = %d %s", resp.StatusCode, body)
	}
	if resp, _ := call(t, app, "POST", "/api/v1/admin/ldap/import-jellyfin", "", bob); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("user import = %d", resp.StatusCode)
	}
}
