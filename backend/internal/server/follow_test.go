package server_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestFollowRequest(t *testing.T) {
	t.Parallel()
	e := newEnv(t, false)
	_, admin, bobCookie := setupUsers(t, e)
	base := e.app.URL
	relay := newFakeRelay(t)
	radarr := newFakeArr(t, "radarr")
	addInstance(t, e, admin, "radarr", radarr)
	bob := appLogin(t, base, "bob", "hunter2", "Bob's iPhone")
	follow := func(id, body string) (*http.Response, string) {
		return bearerCall(t, base, "PUT", "/api/v1/requests/"+id+"/follow", body, bob)
	}

	resp, body := call(t, e.app, "POST", "/api/v1/requests", `{"type":"movie","tmdbId":5}`, bobCookie)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create = %d %s", resp.StatusCode, body)
	}
	v := decodeReq(t, body)
	tokBody := `{"pushToken":"` + iosTok + `","activityId":"ABC-1","server":"tipsarr.example.org"}`

	// push off: nothing to follow with
	if resp, body := follow(v.ID, tokBody); resp.StatusCode != 409 || !strings.Contains(body, "push_unavailable") {
		t.Fatalf("push off = %d %s", resp.StatusCode, body)
	}
	call(t, e.app, "PUT", "/api/v1/admin/settings", `{"pushRelayUrl":"`+relay.srv.URL+`","pushRelayKey":"pr_relaykey","pushEnabled":true}`, admin)
	_, st := call(t, e.app, "GET", "/api/v1/status", "", bobCookie)
	if !strings.Contains(st, `"liveFollow":true`) {
		t.Fatalf("status: %s", st)
	}

	if resp, body := call(t, e.app, "PUT", "/api/v1/requests/"+v.ID+"/follow", tokBody, bobCookie); resp.StatusCode != 422 || !strings.Contains(body, "not_an_app") {
		t.Fatalf("web = %d %s", resp.StatusCode, body)
	}
	if resp, body := follow(v.ID, `{"pushToken":"`+iosTok+`","activityId":"bad id!"}`); resp.StatusCode != 422 || !strings.Contains(body, "invalid_activity_id") {
		t.Fatalf("activity = %d %s", resp.StatusCode, body)
	}
	if resp, _ := follow("nope", tokBody); resp.StatusCode != 404 {
		t.Fatalf("unknown = %d", resp.StatusCode)
	}
	// someone else's request, even for an admin who can see it
	alice := appLogin(t, base, "alice", "secret", "Alice's iPhone")
	if resp, body := bearerCall(t, base, "PUT", "/api/v1/requests/"+v.ID+"/follow", tokBody, alice); resp.StatusCode != 404 {
		t.Fatalf("other user = %d %s", resp.StatusCode, body)
	}
	resp, body = follow(v.ID, tokBody)
	if resp.StatusCode != 200 || !strings.Contains(body, `"stage":"requested"`) || !strings.Contains(body, `"platform":"ios"`) || strings.Contains(body, iosTok) {
		t.Fatalf("follow = %d %s", resp.StatusCode, body)
	}
	_, body = bearerCall(t, base, "GET", "/api/v1/me/follows", "", bob)
	var list struct{ Items []struct{ RequestID string } }
	_ = json.Unmarshal([]byte(body), &list)
	if len(list.Items) != 1 || list.Items[0].RequestID != v.ID {
		t.Fatalf("list = %s", body)
	}

	// a final stage ends the Live Activity and the follow
	if resp, body := call(t, e.app, "POST", "/api/v1/requests/"+v.ID+"/decline", `{"reason":"no"}`, admin); resp.StatusCode != 200 {
		t.Fatalf("decline = %d %s", resp.StatusCode, body)
	}
	got := relay.waitFor(t, 1) // bob has no alert device: only the live update
	live := &got[0]
	if live == nil || live.Live == nil || live.Live.Stage != "declined" || !live.Live.End || live.Live.DismissAfter != 3600 ||
		live.Live.Priority != "high" || live.Token != iosTok || live.ID != v.ID || live.Server != "tipsarr.example.org" {
		t.Fatalf("live = %+v", live)
	}
	_, body = bearerCall(t, base, "GET", "/api/v1/me/follows", "", bob)
	if !strings.Contains(body, `"items":[]`) {
		t.Fatalf("follow kept: %s", body)
	}
	if resp, body := follow(v.ID, tokBody); resp.StatusCode != 409 || !strings.Contains(body, "follow_final") {
		t.Fatalf("final = %d %s", resp.StatusCode, body)
	}
	if resp, _ := bearerCall(t, base, "DELETE", "/api/v1/requests/"+v.ID+"/follow", "", bob); resp.StatusCode != 204 {
		t.Fatalf("unfollow = %d", resp.StatusCode)
	}
}
