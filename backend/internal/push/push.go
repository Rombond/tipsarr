// Package push sends mobile push notifications through the relay (see the push-relay repository).
// The relay holds the APNs/FCM credentials; Tipsarr only sends it an event name and an id per device,
// and the app fetches the details itself, so nothing private passes through Apple or Google.
package push

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/i18n"
	"github.com/Rombond/tipsarr/backend/internal/notify"
	"github.com/Rombond/tipsarr/backend/internal/store"
)

// Settings keys.
const (
	SettingEnabled = "push.enabled" // "true" when on (off by default)
	SettingURL     = "push.relay_url"
	SettingKey     = "push.relay_key" // secret, never returned by the API
	SettingText    = "push.text"      // "true": alerts carry the title of the media (the relay must allow free text)
	// SettingInstallID is this server's random id towards the relay's open registration, generated once and kept.
	SettingInstallID = "push.install_id"
	// SettingKeyPublic is "true" when the key came from POST /v1/register (templates only, daily quota).
	SettingKeyPublic = "push.key_public"
	// SettingDailyQuota is the daily quota the relay announced for a public key (0 = unlimited).
	SettingDailyQuota = "push.daily_quota"
	// SettingPausedUntil (unix seconds) stops sending until then, after the relay answered quota_exceeded.
	SettingPausedUntil = "push.paused_until"
)

// DefaultRelayURL is the relay run by the app publisher.
const DefaultRelayURL = "https://push.brebond.com"

// AppName is the app id at the relay.
const AppName = "tipsarr"

const batchSize = 100

// category of each event, as a store.Push* bit.
var categories = map[string]int{
	"request.approved":  store.PushRequests,
	"request.declined":  store.PushRequests,
	"request.failed":    store.PushRequests,
	"request.available": store.PushRequests,
	"request.created":   store.PushAdmin,
	"issue.created":     store.PushIssues,
	"issue.comment":     store.PushIssues,
	"issue.resolved":    store.PushIssues,
}

type Service struct {
	store  *store.Store
	http   *http.Client
	dryRun bool
	delays []time.Duration // waits before attempt 2, 3, ...

	// Version and ServerURL are sent with a registration (abuse investigation at the relay). Optional.
	Version   string
	ServerURL func(ctx context.Context) string

	regMu      sync.Mutex
	regFailed  time.Time     // last automatic re-registration that failed
	regBackoff time.Duration // no automatic re-registration again before this has passed
}

func New(s *store.Store, dryRun bool) *Service {
	return &Service{store: s, http: &http.Client{Timeout: 15 * time.Second}, dryRun: dryRun,
		delays: []time.Duration{2 * time.Second, 10 * time.Second}, regBackoff: time.Hour}
}

// SetRetryDelays overrides the retry schedule (tests).
func (s *Service) SetRetryDelays(d ...time.Duration) { s.delays = d }

type config struct {
	url, key string
	text     bool // alerts carry the media title
	public   bool // key from open registration
}

// load returns the relay settings, ok=false when push is off or incomplete.
func (s *Service) load(ctx context.Context) (config, bool) {
	on, _ := s.store.GetSetting(ctx, SettingEnabled)
	u, _ := s.store.GetSetting(ctx, SettingURL)
	k, _ := s.store.GetSetting(ctx, SettingKey)
	if on != "true" || u == "" || k == "" {
		return config{}, false
	}
	text, _ := s.store.GetSetting(ctx, SettingText)
	public, _ := s.store.GetSetting(ctx, SettingKeyPublic)
	// a public key never carries free text: the relay ignores it, so do not send the title through it at all
	return config{url: strings.TrimRight(u, "/"), key: k, text: text == "true" && public != "true", public: public == "true"}, true
}

// Configured reports whether push is switched on with a relay address and key (/status features.push).
func (s *Service) Configured(ctx context.Context) bool {
	_, ok := s.load(ctx)
	return ok
}

// ValidURL checks a relay address typed by an admin.
func ValidURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("the relay address must start with http:// or https://")
	}
	return nil
}

// Send delivers the event to the matching devices in the background. It never blocks the caller.
func (s *Service) Send(ev notify.PushEvent) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if err := s.send(ctx, ev); err != nil {
			slog.Warn("push failed", "event", ev.Name, "err", err)
		}
	}()
}

// Targets returns the devices that should receive the event.
func (s *Service) Targets(ctx context.Context, ev notify.PushEvent) ([]store.Device, error) {
	bit, ok := categories[ev.Name]
	if !ok {
		return nil, fmt.Errorf("unknown push event %q", ev.Name)
	}
	rows, err := s.store.DevicesForManagers(ctx, ev.Users, ev.Admins, ev.Needs)
	if err != nil {
		return nil, err
	}
	out := rows[:0]
	for _, d := range rows {
		if d.UserID == ev.Skip || d.Categories&bit == 0 {
			continue
		}
		out = append(out, d)
	}
	return out, nil
}

type item struct {
	Platform string `json:"platform"`
	Token    string `json:"token"`
	Sandbox  bool   `json:"sandbox"`
	Event    string `json:"event"`
	ID       string `json:"id"`
	Lang     string `json:"lang"`
	Server   string `json:"server,omitempty"`
	Title    string `json:"title,omitempty"` // only with push.text; the relay ignores it unless its app allows free text
	Body     string `json:"body,omitempty"`
}

type result struct {
	Status     string `json:"status"`
	RetryAfter int    `json:"retryAfter"`
	Error      string `json:"error"`
}

func (s *Service) send(ctx context.Context, ev notify.PushEvent) error {
	cfg, ok := s.load(ctx)
	if !ok {
		return nil
	}
	devs, err := s.Targets(ctx, ev)
	if err != nil || len(devs) == 0 {
		return err
	}
	if s.dryRun {
		slog.Info("dry-run: push not sent", "event", ev.Name, "devices", len(devs))
		return nil
	}
	if until := s.PausedUntil(ctx); until > 0 {
		slog.Info("push: daily quota reached, not sent", "event", ev.Name, "until", time.Unix(until, 0).UTC())
		return nil
	}
	for start := 0; start < len(devs); start += batchSize {
		end := min(start+batchSize, len(devs))
		s.deliver(ctx, cfg, ev, devs[start:end])
	}
	return nil
}

// deliver sends one batch, retrying what the relay could not deliver (network, 5xx, per item errors).
func (s *Service) deliver(ctx context.Context, cfg config, ev notify.PushEvent, devs []store.Device) {
	pending := devs
	reregistered := false
	for attempt := 0; len(pending) > 0; attempt++ {
		results, err := s.post(ctx, cfg, ev, pending)
		if errors.Is(err, errUnauthorized) && cfg.public && !reregistered {
			// key revoked or expired at the relay: register again once with the same installId, never in a loop
			reregistered = true
			if key, rerr := s.reregister(ctx, cfg.url, cfg.key); rerr == nil {
				cfg.key = key
				results, err = s.post(ctx, cfg, ev, pending)
			} else {
				slog.Warn("push: re-registration failed", "err", rerr)
			}
		}
		var q *quotaError
		if errors.As(err, &q) {
			s.pause(ctx, q.retryAfter)
			return
		}
		if errors.Is(err, errUnauthorized) {
			slog.Warn("push: the relay refused the key", "event", ev.Name)
			return
		}
		var retry []store.Device
		quota := 0
		for i, d := range pending {
			switch {
			case err != nil:
				retry = append(retry, d)
			case i >= len(results):
				retry = append(retry, d)
			case results[i].Status == "invalid_token":
				if err := s.store.DeleteDevice(ctx, d.ID); err != nil {
					slog.Warn("push: drop device", "err", err)
				}
			case results[i].Status == "quota_exceeded":
				quota = max(quota, results[i].RetryAfter, 1)
			case results[i].Status == "rate_limited", results[i].Status == "error":
				retry = append(retry, d)
			}
		}
		if quota > 0 {
			// daily quota spent: stop here, no retry, nothing more is sent until the relay's next day
			s.pause(ctx, quota)
			return
		}
		pending = retry
		if len(pending) == 0 || attempt >= len(s.delays) {
			if len(pending) > 0 {
				slog.Warn("push: gave up", "event", ev.Name, "devices", len(pending), "err", err)
			}
			return
		}
		select {
		case <-time.After(s.delays[attempt]):
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) post(ctx context.Context, cfg config, ev notify.PushEvent, devs []store.Device) ([]result, error) {
	items := make([]item, len(devs))
	for i, d := range devs {
		items[i] = item{Platform: d.Platform, Token: d.PushToken, Sandbox: d.Sandbox, Event: ev.Name, ID: ev.ID, Lang: d.Language, Server: d.Server}
		if cfg.text && ev.Subject != "" {
			items[i].Title, items[i].Body = alertText(ev.Name, d.Language, ev.Subject)
		}
	}
	body, err := json.Marshal(map[string]any{"items": items})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.url+"/v1/push/batch", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.key)
	req.Header.Set("User-Agent", "Tipsarr")
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized:
		io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return nil, errUnauthorized
	case http.StatusTooManyRequests:
		var r result
		_ = json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&r)
		if r.Status == "quota_exceeded" || r.Error == "quota_exceeded" {
			return nil, &quotaError{retryAfter: max(r.RetryAfter, retryAfter(resp), 1)}
		}
		return nil, fmt.Errorf("relay answered HTTP %d", resp.StatusCode)
	default:
		io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("relay answered HTTP %d", resp.StatusCode)
	}
	var out struct {
		Items []result `json:"items"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
		return nil, err
	}
	return out.Items, nil
}

// alertText returns the title and body of the alert; the title says what happened, the body what it is about.
// Titles are the "push.*" texts in locales/<code>/server.json, in the person's language.
func alertText(event, lang, subject string) (title, body string) {
	t, ok := i18n.T(lang, "push."+event, nil)
	if !ok {
		return "", ""
	}
	return t, subject
}
