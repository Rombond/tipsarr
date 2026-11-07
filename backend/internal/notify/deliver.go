package notify

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/i18n"
	"github.com/Rombond/tipsarr/backend/internal/netguard"
	"github.com/Rombond/tipsarr/backend/internal/perms"
	"github.com/Rombond/tipsarr/backend/internal/store"
)

// SettingPublicURL is the address people open Tipsarr at; webhook links ({{ url }}) start with it.
// ErrOffline is returned by Post in demo mode.
var ErrOffline = errors.New("webhooks are offline in the demo")

const SettingPublicURL = "app.public_url"

// settingLanguage mirrors media.SettingDefaultLanguage (importing media here would make a cycle).
const settingLanguage = "app.language"

const posterBase = "https://image.tmdb.org/t/p/w500"

type envelope struct {
	Event     string `json:"event"`
	Timestamp string `json:"timestamp"`
	DryRun    bool   `json:"dryRun"`
	Data      any    `json:"data"`
}

// Media is what an event is about, flattened for templates (media.*).
type Media struct {
	TMDBID     int
	IMDBID     string
	TVDBID     int
	Type       string // movie or tv
	Title      string
	Year       string
	Status     string
	Is4K       bool
	PosterPath string
}

// Comment is an issue comment (comment.*).
type Comment struct {
	Message string
	UserID  string
	Name    string
}

// Event is one notification. Data is the "data" of the raw envelope; the rest feeds templates and
// "one delivery per recipient".
type Event struct {
	Name        string
	Data        any
	Media       *Media
	Request     any    // request view (request.*)
	RequestedBy string // user id (requestedBy.*)
	Issue       any    // issue view (issue.*)
	IssueID     string
	Comment     *Comment
	Reason      string   // decline reason or error, appended to the message
	Users       []string // people concerned
	Admins      bool     // also admins and whoever has one of Needs
	Needs       int64
	Skip        string // user id who caused it (no per-recipient delivery to them)
}

// Dispatch sends an event with only envelope data (kept for simple callers).
func (s *Service) Dispatch(event string, data any) { s.Send(Event{Name: event, Data: data}) }

// Send delivers the event to every enabled webhook subscribed to it, in the background.
func (s *Service) Send(ev Event) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		hooks, err := s.store.ListWebhooks(ctx)
		if err != nil {
			slog.Warn("list webhooks", "err", err)
			return
		}
		var wg sync.WaitGroup
		var recipients []store.User
		var once sync.Once
		for _, h := range hooks {
			if h.Enabled != 1 || !Subscribed(h.Events, ev.Name) {
				continue
			}
			targets := []*store.User{nil}
			if h.PerRecipient == 1 {
				once.Do(func() { recipients = s.Recipients(ctx, ev) })
				targets = targets[:0]
				for i := range recipients {
					targets = append(targets, &recipients[i])
				}
			}
			for _, u := range targets {
				wg.Add(1)
				go func() {
					defer wg.Done()
					s.deliverWithRetry(ctx, h, ev, u)
				}()
			}
		}
		wg.Wait() // keep ctx alive until every delivery (with retries) is done
	}()
}

// Subscribed reports whether a comma-separated events list includes the event ("" or "*" = all).
// request.approved also subscribes to request.auto_approved.
func Subscribed(events, event string) bool {
	events = strings.TrimSpace(events)
	if events == "" || events == "*" {
		return true
	}
	for _, e := range strings.Split(events, ",") {
		e = strings.TrimSpace(e)
		if e == event || (e == RequestApproved && event == RequestAutoApproved) {
			return true
		}
	}
	return false
}

// Recipients lists the people an event concerns: Users, plus the admins (and holders of Needs) when Admins,
// minus Skip.
func (s *Service) Recipients(ctx context.Context, ev Event) []store.User {
	want := map[string]bool{}
	for _, id := range ev.Users {
		if id != "" {
			want[id] = true
		}
	}
	all, err := s.store.ListUsers(ctx)
	if err != nil {
		slog.Warn("webhook recipients", "err", err)
		return nil
	}
	var out []store.User
	for _, u := range all {
		if u.ID == ev.Skip {
			continue
		}
		admin := ev.Admins && (u.Role == store.RoleAdmin || (ev.Needs != 0 && u.Mask()&perms.Mask(ev.Needs) != 0))
		if want[u.ID] || admin {
			out = append(out, u)
		}
	}
	return out
}

func (s *Service) deliverWithRetry(ctx context.Context, h store.Webhook, ev Event, to *store.User) {
	msg, err := s.Render(ctx, h, ev, to)
	if err != nil {
		slog.Warn("webhook not sent", "webhook", h.Name, "event", ev.Name, "err", err)
		return
	}
	for attempt := 0; ; attempt++ {
		if err = s.Post(ctx, msg); err == nil {
			return
		}
		if attempt >= len(s.delays) {
			break
		}
		select {
		case <-time.After(s.delays[attempt]):
		case <-ctx.Done():
			return
		}
	}
	slog.Warn("webhook delivery failed", "webhook", h.Name, "event", ev.Name, "err", err)
}

// targetsServarr reports whether rawURL reaches one of the configured Radarr/Sonarr instances: same
// host name and port, or any common resolved address on the same port (default ports filled in), so
// an alias, a literal IP or another name of the same box is refused too. Fails closed.
func (s *Service) targetsServarr(ctx context.Context, rawURL string) bool {
	host, port, ok := netguard.HostPort(rawURL)
	if !ok {
		return true
	}
	instances, err := s.store.ListServarr(ctx)
	if err != nil {
		return true // fail closed
	}
	var targetIPs []netip.Addr
	resolved := false
	for _, in := range instances {
		ih, ip, ok := netguard.HostPort(in.URL)
		if !ok {
			continue
		}
		if ip != port {
			continue
		}
		if ih == host {
			return true
		}
		if !resolved {
			targetIPs, resolved = netguard.Resolve(ctx, host), true
		}
		for _, a := range netguard.Resolve(ctx, ih) {
			for _, b := range targetIPs {
				if a == b || (a.IsLoopback() && b.IsLoopback()) {
					return true
				}
			}
		}
	}
	return false
}

// Message is a rendered delivery, ready to POST.
type Message struct {
	Event   string
	URL     string
	Body    []byte
	Headers http.Header
}

// ErrServarrTarget refuses a webhook whose (rendered) URL points at Radarr/Sonarr.
var ErrServarrTarget = errors.New("webhook URL points at a configured Radarr/Sonarr instance; refused")

// Render builds the request for one delivery: the templates rendered, the headers and authentication set,
// the body signed. A *TemplateError means the template is broken; ErrServarrTarget a refused URL.
func (s *Service) Render(ctx context.Context, h store.Webhook, ev Event, to *store.User) (*Message, error) {
	vars := s.Vars(ctx, ev, to)
	target := h.URL
	if isTemplate(target) {
		out, err := Render(target, vars)
		if err != nil {
			return nil, &TemplateError{Part: "url", Err: err}
		}
		target = strings.TrimSpace(out)
	}
	if u, err := url.Parse(target); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, &TemplateError{Part: "url", Err: fmt.Errorf("%q is not an http(s) URL", target)}
	}
	if s.targetsServarr(ctx, target) {
		// a webhook pointed at Radarr/Sonarr could write to them outside the dry-run choke point
		return nil, ErrServarrTarget
	}
	var body []byte
	if strings.TrimSpace(h.Template) == "" {
		data := ev.Data
		if data == nil {
			data = map[string]any{}
		}
		b, err := json.Marshal(envelope{Event: ev.Name, Timestamp: vars["timestamp"].(string), DryRun: s.dryRun, Data: data})
		if err != nil {
			return nil, err
		}
		body = b
	} else {
		out, err := Render(h.Template, vars)
		if err != nil {
			return nil, &TemplateError{Part: "body", Err: err}
		}
		body = []byte(out)
	}
	hd := http.Header{}
	hd.Set("Content-Type", "application/json")
	hd.Set("User-Agent", "Tipsarr")
	hd.Set("X-Tipsarr-Event", ev.Name)
	switch h.AuthType {
	case "bearer":
		hd.Set("Authorization", "Bearer "+h.AuthSecret)
	case "basic":
		hd.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(h.AuthUsername+":"+h.AuthSecret)))
	}
	for _, x := range ParseHeaders(h.Headers) {
		if x.Name != "" {
			hd.Set(x.Name, x.Value)
		}
	}
	if h.Secret != "" {
		mac := hmac.New(sha256.New, []byte(h.Secret))
		mac.Write(body)
		hd.Set("X-Tipsarr-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}
	return &Message{Event: ev.Name, URL: target, Body: body, Headers: hd}, nil
}

// ParseHeaders reads the stored header list (bad JSON = none).
func ParseHeaders(raw string) []Header {
	var out []Header
	if strings.TrimSpace(raw) != "" {
		_ = json.Unmarshal([]byte(raw), &out)
	}
	return out
}

// Post performs one POST. It returns an error for transport failures and non-2xx answers.
func (s *Service) Post(ctx context.Context, m *Message) error {
	if s.Offline {
		return ErrOffline
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.URL, bytes.NewReader(m.Body))
	if err != nil {
		return err
	}
	req.Header = m.Headers.Clone()
	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode/100 != 2 {
		return &StatusError{Code: resp.StatusCode}
	}
	return nil
}

// StatusError is a non-2xx answer of the receiver.
type StatusError struct{ Code int }

func (e *StatusError) Error() string { return fmt.Sprintf("webhook answered HTTP %d", e.Code) }

// FailureText is a safe description of a failed delivery for the admin UI: refused by the
// outbound policy, timeout, unreachable or the HTTP status. Never the raw transport error, so the
// test button cannot be used to map the network.
func FailureText(err error) string {
	var se *StatusError
	switch {
	case errors.As(err, &se):
		return se.Error()
	case errors.Is(err, ErrOffline):
		return err.Error()
	case errors.Is(err, ErrServarrTarget):
		return err.Error()
	}
	switch netguard.Classify(err) {
	case netguard.ClassRefused:
		return "refused by the outbound policy (private or local address; an admin can allow it in the webhook settings)"
	case netguard.ClassTimeout:
		return "the receiver did not answer in time"
	}
	return "the receiver could not be reached"
}

// BlockedWebhooks lists saved webhooks whose (fixed) URL the outbound policy refuses now.
func (s *Service) BlockedWebhooks(ctx context.Context) []store.Webhook {
	hooks, err := s.store.ListWebhooks(ctx)
	if err != nil {
		return nil
	}
	var out []store.Webhook
	for _, h := range hooks {
		if !isTemplate(h.URL) && !s.Policy().Allowed(ctx, h.URL) {
			out = append(out, h)
		}
	}
	return out
}

// Deliver renders and performs one delivery (no retry).
func (s *Service) Deliver(ctx context.Context, h store.Webhook, ev Event, to *store.User) error {
	m, err := s.Render(ctx, h, ev, to)
	if err != nil {
		return err
	}
	return s.Post(ctx, m)
}

func person(u *store.User) map[string]any {
	if u == nil {
		return map[string]any{}
	}
	return map[string]any{"id": u.ID, "name": u.Name, "discordId": u.DiscordID, "telegramChatId": u.TelegramChatID, "language": u.Language}
}

// Vars are the template variables of an event for one recipient (nil = not per recipient).
func (s *Service) Vars(ctx context.Context, ev Event, to *store.User) map[string]any {
	public, _ := s.store.GetSetting(ctx, SettingPublicURL)
	public = strings.TrimRight(public, "/")
	lang, _ := s.store.GetSetting(ctx, settingLanguage)
	if to != nil && to.Language != "" {
		lang = to.Language
	}
	v := map[string]any{
		"event": ev.Name, "timestamp": time.Now().UTC().Format(time.RFC3339), "dryRun": s.dryRun, "nl": "\n",
		"image": "", "url": "", "media": map[string]any{}, "request": toMapOr(ev.Request), "issue": toMapOr(ev.Issue),
		"comment": map[string]any{}, "requestedBy": map[string]any{}, "notifyUser": person(to), "data": toMapOr(ev.Data),
	}
	title := ""
	if m := ev.Media; m != nil {
		title = m.Title
		if m.Year != "" {
			title += " (" + m.Year + ")"
		}
		v["media"] = map[string]any{"tmdbId": m.TMDBID, "imdbId": m.IMDBID, "tvdbId": m.TVDBID, "type": m.Type, "title": m.Title,
			"year": m.Year, "status": m.Status, "is4k": m.Is4K}
		if m.PosterPath != "" {
			v["image"] = posterBase + m.PosterPath
		}
		if public != "" && m.TMDBID > 0 {
			v["url"] = fmt.Sprintf("%s/media/%s/%d", public, m.Type, m.TMDBID)
		}
	}
	if ev.IssueID != "" && public != "" {
		v["url"] = public + "/issues/" + ev.IssueID
	}
	user := ""
	if ev.RequestedBy != "" {
		if u, err := s.store.GetUser(ctx, ev.RequestedBy); err == nil {
			v["requestedBy"] = person(u)
			user = u.Name
		}
	}
	params := map[string]any{"title": title, "user": user}
	if c := ev.Comment; c != nil {
		v["comment"] = map[string]any{"message": c.Message, "userId": c.UserID, "name": c.Name}
		params["user"], params["comment"] = c.Name, c.Message
	}
	v["subject"] = text(lang, "webhook."+ev.Name+".subject", params, ev.Name)
	msg := text(lang, "webhook."+ev.Name+".message", params, title)
	if ev.Reason != "" {
		msg += "\n" + ev.Reason
	}
	v["message"] = msg
	return v
}

func toMapOr(x any) any {
	if m := toMap(x); m != nil {
		return m
	}
	return map[string]any{}
}

func text(lang, code string, params map[string]any, fallback string) string {
	if t, ok := i18n.T(lang, code, params); ok {
		return t
	}
	return fallback
}
