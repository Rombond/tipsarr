package notify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/perms"
	"github.com/Rombond/tipsarr/backend/internal/store"
)

func newService(t *testing.T) (*Service, *store.Store) {
	t.Helper()
	st, err := store.Open(context.Background(), "sqlite:"+filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return New(st, true), st
}

func addUser(t *testing.T, st *store.Store, id, name string, admin bool, discord string) *store.User {
	t.Helper()
	ctx := context.Background()
	u, err := st.UpsertUser(ctx, id, name, store.AdminGrant{Admin: admin, Source: store.AdminFromJellyfin})
	if err != nil {
		t.Fatal(err)
	}
	u.DiscordID = discord
	if err := st.UpdatePrefs(ctx, u); err != nil {
		t.Fatal(err)
	}
	return u
}

func sample(name string) Event {
	return Event{Name: name, RequestedBy: "u1", Users: []string{"u1"},
		Media:   &Media{TMDBID: 603, Type: "movie", Title: `The "Matrix"` + "\nReloaded", Year: "1999", Status: "pending", PosterPath: "/p.jpg"},
		Request: map[string]any{"id": "r1", "tmdbId": 603},
		Data:    map[string]any{"request": map[string]any{"title": "x"}}}
}

func TestTemplateEscapesFreeText(t *testing.T) {
	s, st := newService(t)
	addUser(t, st, "u1", `Bob "the" \builder`, false, "")
	h := store.Webhook{URL: "https://example.org/x", Template: `{"title": {{ media.title | json }}, "who": {{ requestedBy.name | json }}, "id": {{ media.tmdbId }}, "missing": {{ issue.title | json }}, "c": {{ comment.message | default: "none" | json }}}`}
	m, err := s.Render(context.Background(), h, sample(RequestCreated), nil)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(m.Body, &got); err != nil {
		t.Fatalf("invalid JSON %s: %v", m.Body, err)
	}
	if got["title"] != "The \"Matrix\"\nReloaded" || got["who"] != `Bob "the" \builder` || got["id"] != float64(603) || got["missing"] != nil || got["c"] != "none" {
		t.Fatalf("rendered %s", m.Body)
	}
}

func TestTemplateErrors(t *testing.T) {
	if err := CheckTemplate("body", `{% if x %}never closed`); err == nil {
		t.Fatal("bad syntax accepted")
	}
	s, _ := newService(t)
	_, err := s.Render(context.Background(), store.Webhook{URL: "https://e.org", Template: `{{ x | nosuchfilter }}`}, sample(Test), nil)
	var te *TemplateError
	if !errors.As(err, &te) || te.Part != "body" {
		t.Fatalf("err = %v", err)
	}
	_, err = s.Render(context.Background(), store.Webhook{URL: "{{ media.title }}"}, sample(Test), nil)
	if !errors.As(err, &te) || te.Part != "url" {
		t.Fatalf("url err = %v", err)
	}
}

func TestRawEnvelopeByDefault(t *testing.T) {
	s, _ := newService(t)
	m, err := s.Render(context.Background(), store.Webhook{URL: "https://e.org", Secret: "k"}, sample(RequestCreated), nil)
	if err != nil {
		t.Fatal(err)
	}
	var env struct {
		Event  string         `json:"event"`
		DryRun bool           `json:"dryRun"`
		Data   map[string]any `json:"data"`
	}
	if json.Unmarshal(m.Body, &env) != nil || env.Event != RequestCreated || !env.DryRun || env.Data["request"] == nil {
		t.Fatalf("envelope %s", m.Body)
	}
	if !strings.HasPrefix(m.Headers.Get("X-Tipsarr-Signature"), "sha256=") {
		t.Fatal("not signed")
	}
}

func TestPresetsRenderValidJSON(t *testing.T) {
	s, st := newService(t)
	addUser(t, st, "u1", "Bob", false, "1234")
	ctx := context.Background()
	_ = st.SetSetting(ctx, SettingPublicURL, "https://tipsarr.example.org/")
	issue := sample(IssueCommented)
	issue.IssueID, issue.Issue, issue.Comment = "i1", map[string]any{"id": "i1"}, &Comment{Message: "a \"quote\"\nline", Name: "Bob"}
	bare := Event{Name: Test} // no media, no request: optional sections missing
	for _, p := range Presets {
		for _, ev := range []Event{sample(RequestApproved), issue, bare} {
			for _, to := range []*store.User{nil, {ID: "u1", Name: "Bob", DiscordID: "1234", TelegramChatID: "-99"}} {
				m, err := s.Render(ctx, store.Webhook{URL: "https://e.org", Template: p.Template}, ev, to)
				if err != nil {
					t.Fatalf("%s/%s: %v", p.ID, ev.Name, err)
				}
				if !json.Valid(m.Body) {
					t.Fatalf("%s/%s: invalid JSON:\n%s", p.ID, ev.Name, m.Body)
				}
			}
		}
	}
	// spot checks
	disc := Presets[1]
	m, _ := s.Render(ctx, store.Webhook{URL: "https://e.org", Template: disc.Template}, sample(RequestApproved), &store.User{ID: "u1", DiscordID: "1234"})
	if !strings.Contains(string(m.Body), `<@1234>`) || !strings.Contains(string(m.Body), "https://image.tmdb.org/t/p/w500/p.jpg") || !strings.Contains(string(m.Body), "https://tipsarr.example.org/media/movie/603") {
		t.Fatalf("discord = %s", m.Body)
	}
}

func TestAuthAndCustomHeaders(t *testing.T) {
	s, _ := newService(t)
	ctx := context.Background()
	m, _ := s.Render(ctx, store.Webhook{URL: "https://e.org", AuthType: "bearer", AuthSecret: "tok", Headers: `[{"name":"X-Gotify-Key","value":"abc"},{"name":"Content-Type","value":"text/plain"}]`}, sample(Test), nil)
	if m.Headers.Get("Authorization") != "Bearer tok" || m.Headers.Get("X-Gotify-Key") != "abc" || m.Headers.Get("Content-Type") != "text/plain" {
		t.Fatalf("headers %v", m.Headers)
	}
	m, _ = s.Render(ctx, store.Webhook{URL: "https://e.org", AuthType: "basic", AuthUsername: "me", AuthSecret: "pw"}, sample(Test), nil)
	if m.Headers.Get("Authorization") != "Basic "+base64.StdEncoding.EncodeToString([]byte("me:pw")) {
		t.Fatalf("basic %v", m.Headers)
	}
}

func TestURLTemplateCannotTargetServarr(t *testing.T) {
	s, st := newService(t)
	ctx := context.Background()
	if err := st.SaveServarr(ctx, &store.ServarrInstance{ID: "r", Kind: "radarr", Name: "r", URL: "http://radarr:7878"}); err != nil {
		t.Fatal(err)
	}
	_, err := s.Render(ctx, store.Webhook{URL: `http://{% if true %}radarr{% endif %}:7878/api/v3/movie`}, sample(Test), nil)
	if !errors.Is(err, ErrServarrTarget) {
		t.Fatalf("err = %v", err)
	}
}

func TestPerRecipientFanOut(t *testing.T) {
	s, st := newService(t)
	s.SetRetryDelays()
	ctx := context.Background()
	addUser(t, st, "admin", "Alice", true, "111")
	addUser(t, st, "u1", "Bob", false, "")
	addUser(t, st, "u2", "Carol", false, "333")

	var mu sync.Mutex
	got := map[string]string{} // notifyUser.id -> discord id
	recv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var v map[string]string
		_ = json.Unmarshal(b, &v)
		mu.Lock()
		got[v["id"]] = v["discord"]
		mu.Unlock()
	}))
	defer recv.Close()
	if err := st.SaveWebhook(ctx, &store.Webhook{ID: "w", Name: "w", URL: recv.URL, Enabled: 1, PerRecipient: 1,
		Template: `{"id": {{ notifyUser.id | json }}, "discord": {{ notifyUser.discordId | default: "-" | json }}}`}); err != nil {
		t.Fatal(err)
	}
	// a comment by Carol: the reporter (Bob) and the admins hear about it, not Carol
	s.Send(Event{Name: IssueCommented, Users: []string{"u1", "u2"}, Admins: true, Needs: int64(perms.ManageIssues), Skip: "u2"})
	deadline := time.Now().Add(3 * time.Second)
	for {
		mu.Lock()
		n := len(got)
		mu.Unlock()
		if n >= 2 || time.Now().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 2 || got["admin"] != "111" || got["u1"] != "-" {
		t.Fatalf("deliveries %v", got)
	}
}

func TestSubscribedAutoApprovedCompat(t *testing.T) {
	if !Subscribed("request.approved", RequestAutoApproved) || Subscribed("request.created", RequestAutoApproved) || !Subscribed("request.auto_approved", RequestAutoApproved) {
		t.Fatal("subscription rules")
	}
}
