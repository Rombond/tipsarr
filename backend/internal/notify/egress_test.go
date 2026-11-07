package notify

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/Rombond/tipsarr/backend/internal/store"
)

func TestServarrGuardResolvesHosts(t *testing.T) {
	s, st := newService(t)
	ctx := context.Background()
	if err := st.SaveServarr(ctx, &store.ServarrInstance{ID: "r", Kind: "radarr", Name: "r", URL: "http://localhost:7878"}); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveServarr(ctx, &store.ServarrInstance{ID: "s", Kind: "sonarr", Name: "s", URL: "https://10.0.0.9"}); err != nil {
		t.Fatal(err)
	}
	for _, u := range []string{"http://127.0.0.1:7878/api", "http://LOCALHOST:7878", "https://10.0.0.9:443/x", "http://10.0.0.9:443"} {
		if _, err := s.Render(ctx, store.Webhook{URL: u}, sample(Test), nil); !errors.Is(err, ErrServarrTarget) {
			t.Errorf("%s: err = %v", u, err)
		}
	}
	for _, u := range []string{"http://127.0.0.1:9000", "https://10.0.0.9:8443", "https://e.org"} {
		if _, err := s.Render(ctx, store.Webhook{URL: u}, sample(Test), nil); err != nil {
			t.Errorf("%s: err = %v", u, err)
		}
	}
}

func TestDeliveryRefusedByPolicy(t *testing.T) {
	s, st := newService(t)
	ctx := context.Background()
	m := &Message{URL: "http://10.255.255.1:9/x", Body: []byte("{}"), Headers: http.Header{}}
	err := s.Post(ctx, m)
	if txt := FailureText(err); !strings.Contains(txt, "refused by the outbound policy") || strings.Contains(txt, "10.255") {
		t.Fatalf("err = %v / %s", err, txt)
	}
	_ = st.SaveWebhook(ctx, &store.Webhook{ID: "w", Name: "lan", URL: "http://10.255.255.1:9/x", Enabled: 1})
	if b := s.BlockedWebhooks(ctx); len(b) != 1 {
		t.Fatalf("blocked = %v", b)
	}
	_ = st.SetSetting(ctx, SettingAllowedHosts, "10.255.255.0/24")
	if b := s.BlockedWebhooks(ctx); len(b) != 0 {
		t.Fatalf("allowed but blocked = %v", b)
	}
}
