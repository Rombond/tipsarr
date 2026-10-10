package media

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/store"
)

// With TMDB down, an expired cached copy is served instead of an error; a real 404 is not masked.
func TestFetchServesStaleWhenTMDBFails(t *testing.T) {
	var status atomic.Int32
	status.Store(200)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(int(status.Load()))
		if status.Load() == 200 {
			_, _ = w.Write([]byte(`{"ok":true}`))
		}
	}))
	defer srv.Close()
	ctx := context.Background()
	st, err := store.Open(ctx, "sqlite:"+filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.SetSetting(ctx, settingTMDBKey, "k"); err != nil {
		t.Fatal(err)
	}
	s := New(st, srv.URL)

	// cached with a TTL that is already over
	if _, err := s.fetch(ctx, "/movie/1", nil, -time.Second); err != nil {
		t.Fatal(err)
	}
	status.Store(503)
	body, err := s.fetch(ctx, "/movie/1", nil, time.Hour)
	if err != nil || string(body) != `{"ok":true}` {
		t.Fatalf("stale fallback = %q, %v", body, err)
	}
	if _, err := s.fetch(ctx, "/movie/2", nil, time.Hour); err == nil {
		t.Fatal("an uncached path must still fail while TMDB is down")
	}
	status.Store(404)
	if _, err := s.fetch(ctx, "/movie/1", nil, time.Hour); !errors.Is(err, ErrNotFound) {
		t.Fatalf("404 must not be masked by the stale copy, got %v", err)
	}
}

func TestPickTrailerPrefersTheUsersLanguage(t *testing.T) {
	video := func(key, lang string, official bool) struct {
		Key      string `json:"key"`
		Site     string `json:"site"`
		Type     string `json:"type"`
		Official bool   `json:"official"`
		Language string `json:"iso_639_1"`
	} {
		return struct {
			Key      string `json:"key"`
			Site     string `json:"site"`
			Type     string `json:"type"`
			Official bool   `json:"official"`
			Language string `json:"iso_639_1"`
		}{key, "YouTube", "Trailer", official, lang}
	}
	build := func(vs ...struct {
		Key      string `json:"key"`
		Site     string `json:"site"`
		Type     string `json:"type"`
		Official bool   `json:"official"`
		Language string `json:"iso_639_1"`
	}) rawDetail {
		var raw rawDetail
		raw.Videos.Results = vs
		return raw
	}
	cases := []struct {
		name string
		raw  rawDetail
		lang string
		want string
	}{
		{"official in the user's language wins", build(video("en-off", "en", true), video("fr-off", "fr", true), video("fr-any", "fr", false)), "fr-FR", "fr-off"},
		{"any trailer in the user's language beats official English", build(video("en-off", "en", true), video("fr-any", "fr", false)), "fr-FR", "fr-any"},
		{"falls back to official English", build(video("en-any", "en", false), video("en-off", "en", true)), "fr-FR", "en-off"},
		{"then any English", build(video("de", "de", true), video("en-any", "en", false)), "fr-FR", "en-any"},
		{"then a video without a language", build(video("de", "de", true), video("none", "", false)), "fr-FR", "none"},
		{"English users get English", build(video("fr-off", "fr", true), video("en-any", "en", false)), "en-US", "en-any"},
		{"no trailer", build(), "fr-FR", ""},
	}
	for _, c := range cases {
		if got := pickTrailer(c.raw, c.lang); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
	if got := videoLanguages("fr-FR"); got != "fr,en,null" {
		t.Errorf("videoLanguages fr = %q", got)
	}
	if got := videoLanguages("en-US"); got != "en,null" {
		t.Errorf("videoLanguages en = %q", got)
	}
}
