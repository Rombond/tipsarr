package server_test

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"strings"
	"testing"
)

func TestUploadedAvatar(t *testing.T) {
	e := newEnv(t, true)
	_, _, bob := setupUsers(t, e)
	_, body := call(t, e.app, "GET", "/api/v1/me", "", bob)
	var me struct{ ID string }
	_ = json.Unmarshal([]byte(body), &me)

	// no picture yet (the fake Jellyfin has none either)
	if resp, _ := call(t, e.app, "GET", "/api/v1/users/"+me.ID+"/avatar", "", bob); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("avatar before upload = %d", resp.StatusCode)
	}
	img := image.NewRGBA(image.Rect(0, 0, 640, 400)) // not square: it must be cropped
	for x := 0; x < 640; x++ {
		for y := 0; y < 400; y++ {
			img.Set(x, y, color.RGBA{200, 30, 30, 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	if resp, _ := call(t, e.app, "POST", "/api/v1/me/avatar", buf.String(), bob); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("upload = %d", resp.StatusCode)
	}
	resp, _ := call(t, e.app, "GET", "/api/v1/users/"+me.ID+"/avatar", "", bob)
	if resp.StatusCode != 200 || resp.Header.Get("Content-Type") != "image/jpeg" {
		t.Fatalf("avatar = %d %s", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	if _, b := call(t, e.app, "GET", "/api/v1/users/"+me.ID, "", bob); !strings.Contains(b, `"hasUploadedAvatar":true`) {
		t.Fatalf("profile = %s", b)
	}
	// not a picture, and not logged in
	if resp, _ := call(t, e.app, "POST", "/api/v1/me/avatar", "definitely not an image", bob); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("garbage = %d", resp.StatusCode)
	}
	if resp, _ := call(t, e.app, "POST", "/api/v1/me/avatar", buf.String()); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous upload = %d", resp.StatusCode)
	}
	// removing it brings the fallback back
	if resp, _ := call(t, e.app, "DELETE", "/api/v1/me/avatar", "", bob); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete = %d", resp.StatusCode)
	}
	if resp, _ := call(t, e.app, "GET", "/api/v1/users/"+me.ID+"/avatar", "", bob); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("avatar after delete = %d", resp.StatusCode)
	}
}
