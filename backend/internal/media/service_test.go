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
