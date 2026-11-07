package server_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestWebhookEgressPolicyAPI(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)

	resp, body := call(t, e.app, "POST", "/api/v1/admin/webhooks", `{"name":"lan","url":"http://10.255.255.1:9/hook","enabled":true}`, admin)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create = %d %s", resp.StatusCode, body)
	}
	var w struct{ ID string }
	_ = json.Unmarshal([]byte(body), &w)

	// the test button says why without echoing the dial error
	resp, body = call(t, e.app, "POST", "/api/v1/admin/webhooks/"+w.ID+"/test", "", admin)
	if resp.StatusCode != http.StatusBadGateway || !strings.Contains(body, "refused by the outbound policy") || strings.Contains(body, "10.255.255.1") {
		t.Fatalf("test = %d %s", resp.StatusCode, body)
	}
	if _, body = call(t, e.app, "GET", "/api/v1/admin/webhooks", "", admin); !strings.Contains(body, `"blockedByPolicy":true`) {
		t.Fatalf("list = %s", body)
	}
	if _, body = call(t, e.app, "GET", "/api/v1/admin/webhooks/egress", "", admin); !strings.Contains(body, `"name":"lan"`) {
		t.Fatalf("egress = %s", body)
	}
	if resp, _ = call(t, e.app, "GET", "/api/v1/admin/webhooks/egress", "", bob); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("egress as user = %d", resp.StatusCode)
	}
	if resp, _ = call(t, e.app, "PUT", "/api/v1/admin/webhooks/egress", `{"allowedHosts":["http://x/"]}`, admin); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("bad entry = %d", resp.StatusCode)
	}
	resp, body = call(t, e.app, "PUT", "/api/v1/admin/webhooks/egress", `{"allowedHosts":["10.255.255.0/24"," ntfy.lan "]}`, admin)
	if resp.StatusCode != 200 || !strings.Contains(body, `"allowedHosts":["10.255.255.0/24","ntfy.lan"]`) || !strings.Contains(body, `"blocked":[]`) {
		t.Fatalf("set egress = %d %s", resp.StatusCode, body)
	}
}

func TestServarrProbeAndKeyGuards(t *testing.T) {
	e := newEnv(t, true)
	_, admin, _ := setupUsers(t, e)

	// metadata address: refused, generic message
	resp, body := call(t, e.app, "POST", "/api/v1/admin/servarr/probe", `{"kind":"radarr","url":"http://169.254.169.254/latest","apiKey":"k"}`, admin)
	if resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(body, "address_refused") || strings.Contains(body, "169.254") {
		t.Fatalf("probe metadata = %d %s", resp.StatusCode, body)
	}

	arr := newFakeArr(t, "radarr")
	addInstance(t, e, admin, "radarr", arr)
	_, body = call(t, e.app, "GET", "/api/v1/admin/servarr", "", admin)
	var list []struct{ ID string }
	_ = json.Unmarshal([]byte(body), &list)
	id := list[0].ID

	// another host without a key: refused; same host, other path: kept
	resp, body = call(t, e.app, "PUT", "/api/v1/admin/servarr/"+id,
		`{"kind":"radarr","name":"r","url":"http://evil.example:7878","qualityProfileId":4,"rootFolder":"/data/media","isDefault":true}`, admin)
	if resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(body, "retype_key") {
		t.Fatalf("host change = %d %s", resp.StatusCode, body)
	}
	resp, body = call(t, e.app, "PUT", "/api/v1/admin/servarr/"+id,
		`{"kind":"radarr","name":"r","url":"`+arr.srv.URL+`/radarr","qualityProfileId":4,"rootFolder":"/data/media","isDefault":true}`, admin)
	if resp.StatusCode != 200 {
		t.Fatalf("path change = %d %s", resp.StatusCode, body)
	}
	resp, _ = call(t, e.app, "PUT", "/api/v1/admin/servarr/"+id,
		`{"kind":"radarr","name":"r","url":"http://evil.example:7878","apiKey":"new","qualityProfileId":4,"rootFolder":"/data/media","isDefault":true}`, admin)
	if resp.StatusCode != 200 {
		t.Fatalf("host change with key = %d", resp.StatusCode)
	}
}
