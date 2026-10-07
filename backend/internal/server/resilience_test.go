package server_test

import (
	"net/http"
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
