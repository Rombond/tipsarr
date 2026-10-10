package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type pushItem struct {
	Platform string `json:"platform"`
	Token    string `json:"token"`
	Event    string `json:"event"`
	ID       string `json:"id"`
	Lang     string `json:"lang"`
}

// fakeRelay records what Tipsarr sends to the push relay.
type fakeRelay struct {
	srv   *httptest.Server
	mu    sync.Mutex
	items []pushItem
}

func newFakeRelay(t *testing.T) *fakeRelay {
	t.Helper()
	r := &fakeRelay{}
	r.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Authorization") != "Bearer pr_relaykey" {
			w.WriteHeader(401)
			return
		}
		var body struct{ Items []pushItem }
		_ = json.NewDecoder(req.Body).Decode(&body)
		r.mu.Lock()
		r.items = append(r.items, body.Items...)
		r.mu.Unlock()
		out := make([]map[string]string, len(body.Items))
		for i := range out {
			out[i] = map[string]string{"status": "sent"}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"items": out})
	}))
	t.Cleanup(r.srv.Close)
	return r
}

func (r *fakeRelay) waitFor(t *testing.T, n int) []pushItem {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		r.mu.Lock()
		got := append([]pushItem(nil), r.items...)
		r.mu.Unlock()
		if len(got) >= n {
			return got
		}
		time.Sleep(10 * time.Millisecond)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	t.Fatalf("relay got %d items, want %d: %+v", len(r.items), n, r.items)
	return nil
}

func (r *fakeRelay) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.items)
}

const (
	iosTok     = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	androidTok = "fAkEfCmToKeN_1234567890:APA91bFakeFakeFakeFakeFakeFake"
)

func TestPushRelayEndToEnd(t *testing.T) {
	e := newEnv(t, false)
	jfURL, admin, bobCookie := setupUsers(t, e)
	base := e.app.URL
	relay := newFakeRelay(t)
	radarr := newFakeArr(t, "radarr")
	addInstance(t, e, admin, "radarr", radarr)
	bob := appLogin(t, base, "bob", "hunter2", "Bob's iPhone")
	alice := appLogin(t, base, "alice", "secret", "Alice's Pixel")
	_ = jfURL

	status := func() bool {
		_, body := call(t, e.app, "GET", "/api/v1/status", "", bobCookie)
		var s struct{ Features struct{ Push bool } }
		_ = json.Unmarshal([]byte(body), &s)
		return s.Features.Push
	}

	// --- admin settings -------------------------------------------------------------------
	if resp, _ := call(t, e.app, "PUT", "/api/v1/admin/settings", `{"pushEnabled":true}`, bobCookie); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("user settings = %d", resp.StatusCode)
	}
	if resp, body := call(t, e.app, "PUT", "/api/v1/admin/settings", `{"pushEnabled":true}`, admin); resp.StatusCode != 422 || !strings.Contains(body, "push_not_configured") {
		t.Fatalf("enable without relay = %d %s", resp.StatusCode, body)
	}
	if resp, body := call(t, e.app, "PUT", "/api/v1/admin/settings", `{"pushRelayUrl":"not a url"}`, admin); resp.StatusCode != 422 || !strings.Contains(body, "invalid_push_url") {
		t.Fatalf("bad url = %d %s", resp.StatusCode, body)
	}
	resp, body := call(t, e.app, "PUT", "/api/v1/admin/settings", `{"pushRelayUrl":"`+relay.srv.URL+`/","pushRelayKey":"pr_relaykey"}`, admin)
	if resp.StatusCode != 200 || strings.Contains(body, "pr_relaykey") || !strings.Contains(body, `"pushRelayKeyConfigured":true`) || !strings.Contains(body, `"pushEnabled":false`) {
		t.Fatalf("save relay = %d %s", resp.StatusCode, body)
	}
	if status() {
		t.Fatal("features.push must stay false until push is enabled")
	}
	if resp, body := call(t, e.app, "PUT", "/api/v1/admin/settings", `{"pushEnabled":true}`, admin); resp.StatusCode != 200 || !strings.Contains(body, `"pushEnabled":true`) {
		t.Fatalf("enable = %d %s", resp.StatusCode, body)
	}
	if !status() {
		t.Fatal("features.push should be true")
	}

	// --- device registration ----------------------------------------------------------------
	reg := func(token, tok, body string) (*http.Response, string) {
		return bearerCall(t, base, "PUT", "/api/v1/me/devices/current", body, token)
	}
	if resp, _ := call(t, e.app, "PUT", "/api/v1/me/devices/current", `{"pushToken":"`+iosTok+`"}`, bobCookie); resp.StatusCode != 422 {
		t.Fatalf("web session registered a device: %d", resp.StatusCode)
	}
	if resp, body := reg(bob, "", `{"pushToken":"zzzz-not-an-apns-token-zzzz-zzzz"}`); resp.StatusCode != 422 || !strings.Contains(body, "invalid_push_token") {
		t.Fatalf("bad token = %d %s", resp.StatusCode, body)
	}
	if resp, _ := bearerCall(t, base, "GET", "/api/v1/me/devices/current", "", bob); resp.StatusCode != 404 {
		t.Fatalf("before register = %d", resp.StatusCode)
	}
	resp, body = reg(bob, "", `{"pushToken":"`+iosTok+`","language":"fr-CA","sandbox":true}`)
	if resp.StatusCode != 200 || strings.Contains(body, iosTok) || !strings.Contains(body, `"categories":7`) || !strings.Contains(body, `"language":"fr"`) || !strings.Contains(body, `"platform":"ios"`) {
		t.Fatalf("register = %d %s", resp.StatusCode, body)
	}
	// Alice's login used platform ios in appLogin; register an Android-looking token via a second login
	body2 := `{"username":"alice","password":"secret","platform":"android","deviceName":"Pixel","appVersion":"1.0.0"}`
	_, out := bearerCall(t, base, "POST", "/api/v1/auth/token", body2, "")
	var tb struct{ Token string }
	_ = json.Unmarshal([]byte(out), &tb)
	if resp, body := reg(tb.Token, "", `{"pushToken":"`+androidTok+`","language":"en"}`); resp.StatusCode != 200 || !strings.Contains(body, `"platform":"android"`) {
		t.Fatalf("android register = %d %s", resp.StatusCode, body)
	}
	_ = alice
	// refresh keeps the toggles unless they are sent
	reg(bob, "", `{"pushToken":"`+iosTok+`","categories":1}`)
	if _, body := reg(bob, "", `{"pushToken":"`+iosTok+`"}`); !strings.Contains(body, `"categories":1`) {
		t.Fatalf("refresh changed categories: %s", body)
	}
	if resp, _ := reg(bob, "", `{"pushToken":"`+iosTok+`","categories":9}`); resp.StatusCode != 422 {
		t.Fatalf("categories out of range = %d", resp.StatusCode)
	}
	if resp, body := bearerCall(t, base, "GET", "/api/v1/me/devices/current", "", bob); resp.StatusCode != 200 || strings.Contains(body, iosTok) {
		t.Fatalf("get device = %d %s", resp.StatusCode, body)
	}

	// --- events reach the right devices --------------------------------------------------------
	resp, body = call(t, e.app, "POST", "/api/v1/requests", `{"type":"movie","tmdbId":5}`, bobCookie)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create = %d %s", resp.StatusCode, body)
	}
	v := decodeReq(t, body)
	got := relay.waitFor(t, 1) // the new request goes to the admin's Android phone, not to bob
	if got[0].Token != androidTok || got[0].Event != "request.created" || got[0].ID != v.ID || got[0].Platform != "android" || got[0].Lang != "en" {
		t.Fatalf("admin push = %+v", got[0])
	}

	// bob only wants category 1 (his requests), so the decline reaches him, in French
	if resp, body := call(t, e.app, "POST", "/api/v1/requests/"+v.ID+"/decline", `{"reason":"no"}`, admin); resp.StatusCode != 200 {
		t.Fatalf("decline = %d %s", resp.StatusCode, body)
	}
	got = relay.waitFor(t, 2)
	if got[1].Token != iosTok || got[1].Event != "request.declined" || got[1].ID != v.ID || got[1].Lang != "fr" {
		t.Fatalf("requester push = %+v", got[1])
	}
	time.Sleep(100 * time.Millisecond)
	if relay.count() != 2 {
		t.Fatalf("the admin who declined must not be notified: %+v", relay.items)
	}

	// --- signing out ends the registration ----------------------------------------------------
	if resp, _ := bearerCall(t, base, "DELETE", "/api/v1/me/devices/current", "", bob); resp.StatusCode != 204 {
		t.Fatalf("unregister = %d", resp.StatusCode)
	}
	if resp, _ := bearerCall(t, base, "GET", "/api/v1/me/devices/current", "", bob); resp.StatusCode != 404 {
		t.Fatalf("after unregister = %d", resp.StatusCode)
	}
	reg(bob, "", `{"pushToken":"`+iosTok+`"}`)
	if resp, _ := bearerCall(t, base, "POST", "/api/v1/auth/logout", "", bob); resp.StatusCode/100 != 2 {
		t.Fatalf("logout = %d", resp.StatusCode)
	}
	devs, err := e.st.DevicesFor(context.Background(), nil, true)
	if err != nil || len(devs) != 1 || devs[0].PushToken != androidTok {
		t.Fatalf("admin devices %+v %v", devs, err)
	}
	var left int
	_ = e.st.DB.NewSelect().TableExpr("devices").ColumnExpr("count(*)").Scan(context.Background(), &left)
	if left != 1 {
		t.Fatalf("devices after logout = %d, want 1", left)
	}

	// --- turning push off stops everything ----------------------------------------------------
	call(t, e.app, "PUT", "/api/v1/admin/settings", `{"pushEnabled":false}`, admin)
	if status() {
		t.Fatal("features.push should be false")
	}
	before := relay.count()
	call(t, e.app, "POST", "/api/v1/requests", `{"type":"movie","tmdbId":6}`, bobCookie)
	time.Sleep(150 * time.Millisecond)
	if relay.count() != before {
		t.Fatal("push sent while disabled")
	}
	// clearing the relay key also switches it off
	call(t, e.app, "PUT", "/api/v1/admin/settings", `{"pushEnabled":true}`, admin)
	if resp, body := call(t, e.app, "PUT", "/api/v1/admin/settings", `{"pushRelayKey":""}`, admin); resp.StatusCode != 200 || !strings.Contains(body, `"pushEnabled":false`) {
		t.Fatalf("clear key = %d %s", resp.StatusCode, body)
	}
}
