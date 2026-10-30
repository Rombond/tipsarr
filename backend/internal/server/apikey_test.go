package server_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// keyCall sends a request with only the X-Api-Key header (no session).
func keyCall(t *testing.T, base, method, path, body, key string) (*http.Response, string) {
	t.Helper()
	req, _ := http.NewRequest(method, base+path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if key != "" {
		req.Header.Set("X-Api-Key", key)
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

// TestAPIKey checks the read-only dashboard key: off by default, reads what an admin reads, never writes, and
// replacing or switching it off stops the old one at once.
func TestAPIKey(t *testing.T) {
	jf := fakeJellyfin(t)
	e := newEnv(t, true)
	admin := loginAs(t, e.app, jf.URL, "alice", "secret")
	base := e.app.URL

	// off by default: no key is saved and an invented one opens nothing
	_, body := call(t, e.app, "GET", "/api/v1/admin/settings", "", admin)
	var s struct {
		APIKey string `json:"apiKey"`
	}
	if err := json.Unmarshal([]byte(body), &s); err != nil || s.APIKey != "" {
		t.Fatalf("key before turning on = %q (%v)", s.APIKey, err)
	}
	if resp, _ := keyCall(t, base, "GET", "/api/v1/requests/counts", "", "guess"); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unknown key = %d, want 401", resp.StatusCode)
	}

	// turn it on
	_, body = call(t, e.app, "PUT", "/api/v1/admin/settings", `{"apiKeyEnabled":true}`, admin)
	if err := json.Unmarshal([]byte(body), &s); err != nil || len(s.APIKey) != 32 {
		t.Fatalf("key after turning on = %q (%v)", s.APIKey, err)
	}
	key := s.APIKey

	// it reads what an admin reads, even the admin-only pages
	for _, path := range []string{"/api/v1/requests/counts", "/api/v1/admin/users", "/api/v1/admin/settings"} {
		if resp, b := keyCall(t, base, "GET", path, "", key); resp.StatusCode != http.StatusOK {
			t.Fatalf("GET %s with the key = %d: %s", path, resp.StatusCode, b)
		}
	}
	// a wrong key still opens nothing
	if resp, _ := keyCall(t, base, "GET", "/api/v1/requests/counts", "", key+"0"); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong key = %d, want 401", resp.StatusCode)
	}
	// it never writes
	for _, m := range []string{"POST", "PUT", "PATCH", "DELETE"} {
		resp, b := keyCall(t, base, m, "/api/v1/admin/settings", `{"apiKeyEnabled":false}`, key)
		if resp.StatusCode != http.StatusForbidden || !strings.Contains(b, "api_key_read_only") {
			t.Fatalf("%s with the key = %d: %s", m, resp.StatusCode, b)
		}
	}
	if _, body = call(t, e.app, "GET", "/api/v1/admin/settings", "", admin); !strings.Contains(body, key) {
		t.Fatal("the key was changed by a refused write")
	}

	// a new key replaces the old one
	_, body = call(t, e.app, "PUT", "/api/v1/admin/settings", `{"regenerateApiKey":true}`, admin)
	if err := json.Unmarshal([]byte(body), &s); err != nil || s.APIKey == "" || s.APIKey == key {
		t.Fatalf("regenerated key = %q (%v)", s.APIKey, err)
	}
	if resp, _ := keyCall(t, base, "GET", "/api/v1/requests/counts", "", key); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("old key = %d, want 401", resp.StatusCode)
	}
	if resp, _ := keyCall(t, base, "GET", "/api/v1/requests/counts", "", s.APIKey); resp.StatusCode != http.StatusOK {
		t.Fatalf("new key = %d", resp.StatusCode)
	}

	// off deletes it
	_, body = call(t, e.app, "PUT", "/api/v1/admin/settings", `{"apiKeyEnabled":false}`, admin)
	if err := json.Unmarshal([]byte(body), &s); err != nil || s.APIKey != "" {
		t.Fatalf("key after turning off = %q (%v)", s.APIKey, err)
	}
	if resp, _ := keyCall(t, base, "GET", "/api/v1/requests/counts", "", key); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("deleted key = %d, want 401", resp.StatusCode)
	}
}
