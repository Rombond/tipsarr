package server_test

import (
	"context"

	"github.com/Rombond/tipsarr/backend/internal/auth"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// In demo mode every call that exists only to reach another service answers 403 demo_disabled, and the locked
// settings cannot be changed.
func TestDemoRefusals(t *testing.T) {
	e := newEnvOpts(t, true, "", true)
	jf := fakeJellyfin(t)
	admin := loginAs(t, e.app, jf.URL, "alice", "secret")

	var hits atomic.Int32
	outside := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { hits.Add(1) }))
	defer outside.Close()

	if _, body := call(t, e.app, "GET", "/api/v1/status", ""); !strings.Contains(body, `"demo":true`) {
		t.Fatalf("status = %s", body)
	}
	_, body := call(t, e.app, "POST", "/api/v1/admin/webhooks", `{"name":"x","url":"`+outside.URL+`","enabled":true}`, admin)
	id := strings.Split(strings.Split(body, `"id":"`)[1], `"`)[0]
	refused := func(method, path, in string) {
		t.Helper()
		resp, body := call(t, e.app, method, path, in, admin)
		if resp.StatusCode != http.StatusForbidden || !strings.Contains(body, `"demo_disabled"`) {
			t.Fatalf("%s %s = %d %s", method, path, resp.StatusCode, body)
		}
	}
	refused("POST", "/api/v1/admin/webhooks/"+id+"/test", "")
	refused("POST", "/api/v1/admin/push/register", `{"relayUrl":"`+outside.URL+`"}`)
	refused("POST", "/api/v1/admin/ldap/import-jellyfin", "")
	refused("POST", "/api/v1/admin/teasers/files/test", "")
	refused("GET", "/api/v1/auth/oidc/login", "")
	refused("PUT", "/api/v1/admin/settings", `{"oidcIssuer":"`+outside.URL+`"}`)
	refused("PUT", "/api/v1/admin/settings", `{"ldapUrl":"ldap://`+strings.TrimPrefix(outside.URL, "http://")+`"}`)
	refused("PUT", "/api/v1/admin/settings", `{"pushEnabled":true}`)
	if hits.Load() != 0 {
		t.Fatalf("the outside service was reached %d times", hits.Load())
	}

	// the TMDB key is locked: an update is ignored, the rest of the form is saved
	_, before := call(t, e.app, "GET", "/api/v1/admin/settings", "", admin)
	resp, after := call(t, e.app, "PUT", "/api/v1/admin/settings", `{"tmdbApiKey":"","boxofficeRegions":"FR"}`, admin)
	if resp.StatusCode != 200 || !strings.Contains(after, `"boxofficeRegions":"FR"`) {
		t.Fatalf("update = %d %s", resp.StatusCode, after)
	}
	if strings.Contains(before, `"tmdbConfigured":true`) != strings.Contains(after, `"tmdbConfigured":true`) {
		t.Fatalf("tmdb key changed: %s -> %s", before, after)
	}
}

// One-click sign-in exists only on a demo server.
func TestDemoLoginOnlyInDemo(t *testing.T) {
	e := newEnv(t, true)
	if resp, _ := call(t, e.app, "POST", "/api/v1/auth/demo", `{"role":"admin"}`); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("demo login outside a demo = %d", resp.StatusCode)
	}
}

// A demo admin may add a Radarr and approve a request: dry-run is on for good, so nothing is written to it.
func TestDemoServarrNeverWritten(t *testing.T) {
	e := newEnvOpts(t, true, "", true)
	_, admin, bob := setupUsers(t, e)
	_ = e.st.SetSetting(context.Background(), auth.SettingTMDBKey, "k123") // locked in the API: set it as main does
	radarr := newFakeArr(t, "radarr")
	addInstance(t, e, admin, "radarr", radarr)
	resp, body := call(t, e.app, "POST", "/api/v1/requests", `{"type":"movie","tmdbId":5}`, bob)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("request = %d %s", resp.StatusCode, body)
	}
	id := strings.Split(strings.Split(body, `"id":"`)[1], `"`)[0]
	resp, body = call(t, e.app, "POST", "/api/v1/requests/"+id+"/approve", `{}`, admin)
	if resp.StatusCode != 200 || !strings.Contains(body, `"dryRun":true`) {
		t.Fatalf("approve = %d %s", resp.StatusCode, body)
	}
	if w := radarr.writes(); len(w) != 0 {
		t.Fatalf("writes reached Radarr: %v", w)
	}
}
