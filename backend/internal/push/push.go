// Package push sends mobile push notifications through the relay (see the push-relay repository).
// The relay holds the APNs/FCM credentials; Tipsarr only sends it an event name and an id per device,
// and the app fetches the details itself, so nothing private passes through Apple or Google.
package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/notify"
	"github.com/Rombond/tipsarr/backend/internal/store"
)

// Settings keys.
const (
	SettingEnabled = "push.enabled" // "true" when on (off by default)
	SettingURL     = "push.relay_url"
	SettingKey     = "push.relay_key" // secret, never returned by the API
)

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
}

func New(s *store.Store, dryRun bool) *Service {
	return &Service{store: s, http: &http.Client{Timeout: 15 * time.Second}, dryRun: dryRun,
		delays: []time.Duration{2 * time.Second, 10 * time.Second}}
}

// SetRetryDelays overrides the retry schedule (tests).
func (s *Service) SetRetryDelays(d ...time.Duration) { s.delays = d }

type config struct{ url, key string }

// load returns the relay settings, ok=false when push is off or incomplete.
func (s *Service) load(ctx context.Context) (config, bool) {
	on, _ := s.store.GetSetting(ctx, SettingEnabled)
	u, _ := s.store.GetSetting(ctx, SettingURL)
	k, _ := s.store.GetSetting(ctx, SettingKey)
	if on != "true" || u == "" || k == "" {
		return config{}, false
	}
	return config{url: strings.TrimRight(u, "/"), key: k}, true
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
	rows, err := s.store.DevicesFor(ctx, ev.Users, ev.Admins)
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
	for start := 0; start < len(devs); start += batchSize {
		end := min(start+batchSize, len(devs))
		s.deliver(ctx, cfg, ev, devs[start:end])
	}
	return nil
}

// deliver sends one batch, retrying what the relay could not deliver (network, 5xx, per item errors).
func (s *Service) deliver(ctx context.Context, cfg config, ev notify.PushEvent, devs []store.Device) {
	pending := devs
	for attempt := 0; len(pending) > 0; attempt++ {
		results, err := s.post(ctx, cfg, ev, pending)
		var retry []store.Device
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
			case results[i].Status == "rate_limited", results[i].Status == "error":
				retry = append(retry, d)
			}
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
		items[i] = item{Platform: d.Platform, Token: d.PushToken, Sandbox: d.Sandbox, Event: ev.Name, ID: ev.ID, Lang: d.Language}
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
	if resp.StatusCode != http.StatusOK {
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
