package github

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLatestGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "tipsarr/test" {
			t.Errorf("user agent %q", r.Header.Get("User-Agent"))
		}
		w.Write([]byte(`{"tag_name":"v1.2.3"}`))
	}))
	defer srv.Close()
	old := latestReleaseURL
	latestReleaseURL = srv.URL
	defer func() { latestReleaseURL = old }()

	l := NewLatest("test")
	if got := l.Get(); got != "" {
		t.Fatalf("first Get = %q, want empty while the lookup runs", got)
	}
	deadline := time.Now().Add(2 * time.Second)
	for l.Get() == "" && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if got := l.Get(); got != "v1.2.3" {
		t.Fatalf("Get = %q, want v1.2.3", got)
	}
}
