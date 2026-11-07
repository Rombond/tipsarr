package server_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestWebhookTemplatesAPI(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)

	// chat ids on the profile
	if resp, body := call(t, e.app, "PATCH", "/api/v1/me", `{"discordId":"123456","telegramChatId":"-42"}`, bob); resp.StatusCode != 200 || !strings.Contains(body, `"discordId":"123456"`) {
		t.Fatalf("me = %d %s", resp.StatusCode, body)
	}
	if resp, _ := call(t, e.app, "PATCH", "/api/v1/me", `{"discordId":"abc"}`, bob); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("bad discord id = %d", resp.StatusCode)
	}

	if resp, body := call(t, e.app, "GET", "/api/v1/admin/webhooks/presets", "", admin); resp.StatusCode != 200 || !strings.Contains(body, `"discord"`) || !strings.Contains(body, "issue.reopened") {
		t.Fatalf("presets = %d %s", resp.StatusCode, body)
	}

	// a syntax error is refused when saving
	if resp, _ := call(t, e.app, "POST", "/api/v1/admin/webhooks", `{"name":"x","url":"https://e.org","template":"{% if x %}","enabled":true}`, admin); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("broken template = %d", resp.StatusCode)
	}

	// preview renders without sending and masks the secrets
	resp, body := call(t, e.app, "POST", "/api/v1/admin/webhooks/preview?event=request.approved",
		`{"name":"x","url":"https://e.org/{{ media.type }}","template":"{\"t\": {{ subject | json }}}","enabled":true,"authType":"bearer","authSecret":"tok"}`, admin)
	var pv struct {
		URL       string `json:"url"`
		Body      string `json:"body"`
		ValidJSON bool   `json:"validJson"`
		Error     string `json:"error"`
		Headers   []struct{ Name, Value string }
	}
	_ = json.Unmarshal([]byte(body), &pv)
	if resp.StatusCode != 200 || pv.URL != "https://e.org/movie" || !pv.ValidJSON || !strings.Contains(pv.Body, "Dune") || strings.Contains(body, "tok") {
		t.Fatalf("preview = %d %s", resp.StatusCode, body)
	}
	_, body = call(t, e.app, "POST", "/api/v1/admin/webhooks/preview", `{"name":"x","url":"https://e.org","template":"{{ x | nosuch }}","enabled":true}`, admin)
	if !strings.Contains(body, `"error":"body template`) {
		t.Fatalf("preview error = %s", body)
	}

	// the variables help lists what deliveries fill, with sample values and descriptions
	type variable struct {
		Path, Type, Example, Description string
		Empty, PerRecipient              bool
	}
	vars := func(q string) map[string]variable {
		resp, body := call(t, e.app, "GET", "/api/v1/admin/webhooks/variables"+q, "", admin)
		if resp.StatusCode != 200 {
			t.Fatalf("variables%s = %d %s", q, resp.StatusCode, body)
		}
		var list []variable
		_ = json.Unmarshal([]byte(body), &list)
		out := map[string]variable{}
		for _, v := range list {
			if v.Description == "" {
				t.Errorf("variable %s has no description", v.Path)
			}
			out[v.Path] = v
		}
		return out
	}
	rv := vars("?event=request.created&lang=fr")
	if v := rv["media.title"]; v.Example != "Dune" || v.Empty || v.Type != "string" {
		t.Fatalf("media.title = %+v", v)
	}
	if v := rv["media.tmdbId"]; v.Type != "number" || v.Example != "438631" {
		t.Fatalf("media.tmdbId = %+v", v)
	}
	if v := rv["issue.kind"]; !v.Empty {
		t.Fatalf("issue.kind on a request event = %+v", v)
	}
	if v := rv["notifyUser.name"]; !v.PerRecipient || v.Empty {
		t.Fatalf("notifyUser.name = %+v", v)
	}
	if v := rv["requestedBy.discordId"]; v.Path == "" || !strings.Contains(v.Description, "Discord") || !strings.Contains(v.Description, "demandeur") {
		t.Fatalf("requestedBy.discordId = %+v", v)
	}
	if v := vars("?event=issue.created")["issue.kind"]; v.Empty || v.Example != "audio" {
		t.Fatalf("issue.kind on an issue event = %+v", v)
	}
	if resp, _ := call(t, e.app, "GET", "/api/v1/admin/webhooks/variables?event=nope", "", admin); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("unknown event = %d", resp.StatusCode)
	}
	if resp, _ := call(t, e.app, "GET", "/api/v1/admin/webhooks/variables", "", bob); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("variables as a user = %d", resp.StatusCode)
	}

	// a templated webhook gets the rendered body and the auth header
	var mu sync.Mutex
	var bodies []string
	var auth string
	recv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		bodies = append(bodies, string(b))
		auth = r.Header.Get("Authorization")
		mu.Unlock()
	}))
	defer recv.Close()
	resp, body = call(t, e.app, "POST", "/api/v1/admin/webhooks", `{"name":"t","url":"`+recv.URL+`","events":["request.created"],"enabled":true,"perRecipient":true,"authType":"basic","authUsername":"u","authSecret":"p",
		"template":"{\"to\": {{ notifyUser.name | json }}, \"by\": {{ requestedBy.discordId | json }}, \"title\": {{ media.title | json }}}"}`, admin)
	if resp.StatusCode != http.StatusCreated || strings.Contains(body, `"p"`) {
		t.Fatalf("create = %d %s", resp.StatusCode, body)
	}
	id := strings.Split(strings.Split(body, `"id":"`)[1], `"`)[0]
	if resp, body := call(t, e.app, "POST", "/api/v1/admin/webhooks/"+id+"/test", "", admin); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("test = %d %s", resp.StatusCode, body)
	}
	call(t, e.app, "POST", "/api/v1/requests", `{"type":"movie","tmdbId":5}`, bob)
	// test + one delivery to the admin (bob made the request, so he is skipped)
	waitFor(t, "templated delivery", func() bool { mu.Lock(); defer mu.Unlock(); return len(bodies) == 2 })
	mu.Lock()
	defer mu.Unlock()
	if !strings.Contains(bodies[1], `"to": "alice"`) || !strings.Contains(bodies[1], `"by": "123456"`) || auth == "" {
		t.Fatalf("delivery = %v auth=%q", bodies, auth)
	}

	// broken render reported by the test button
	call(t, e.app, "PUT", "/api/v1/admin/webhooks/"+id, `{"name":"t","url":"`+recv.URL+`","enabled":true,"template":"{{ x | nosuch }}"}`, admin)
	if resp, body := call(t, e.app, "POST", "/api/v1/admin/webhooks/"+id+"/test", "", admin); resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(body, "template_error") {
		t.Fatalf("broken test = %d %s", resp.StatusCode, body)
	}
}
