// Package notify delivers outgoing webhooks (ntfy, Discord, anything that takes a JSON POST).
package notify

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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

	"github.com/Rombond/tipsarr/backend/internal/store"
)

// Event names sent to webhooks.
const (
	RequestCreated  = "request.created"
	RequestApproved = "request.approved"
	RequestDeclined = "request.declined"
	RequestFailed   = "request.failed"
	MediaAvailable  = "media.available"
	IssueCreated    = "issue.created"
	IssueCommented  = "issue.commented"
	IssueResolved   = "issue.resolved"
)

var AllEvents = []string{RequestCreated, RequestApproved, RequestDeclined, RequestFailed, MediaAvailable, IssueCreated, IssueCommented, IssueResolved}

type Service struct {
	store  *store.Store
	http   *http.Client
	dryRun bool
	delays []time.Duration // waits before retry 2, 3, ...
}

func New(s *store.Store, dryRun bool) *Service {
	return &Service{store: s, http: &http.Client{Timeout: 10 * time.Second}, dryRun: dryRun, delays: []time.Duration{time.Second, 5 * time.Second}}
}

// SetRetryDelays overrides the retry schedule (tests).
func (s *Service) SetRetryDelays(d ...time.Duration) { s.delays = d }

type envelope struct {
	Event     string `json:"event"`
	Timestamp string `json:"timestamp"`
	DryRun    bool   `json:"dryRun" `
	Data      any    `json:"data"`
}

// Dispatch sends the event to every enabled webhook subscribed to it, in the background.
func (s *Service) Dispatch(event string, data any) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		hooks, err := s.store.ListWebhooks(ctx)
		if err != nil {
			slog.Warn("list webhooks", "err", err)
			return
		}
		var wg sync.WaitGroup
		for _, h := range hooks {
			if h.Enabled == 1 && Subscribed(h.Events, event) {
				wg.Add(1)
				go func() {
					defer wg.Done()
					s.deliverWithRetry(ctx, h, event, data)
				}()
			}
		}
		wg.Wait() // keep ctx alive until every delivery (with retries) is done

	}()
}

// Subscribed reports whether a comma-separated events list includes the event ("" or "*" = all).
func Subscribed(events, event string) bool {
	events = strings.TrimSpace(events)
	if events == "" || events == "*" {
		return true
	}
	for _, e := range strings.Split(events, ",") {
		if strings.TrimSpace(e) == event {
			return true
		}
	}
	return false
}

func (s *Service) deliverWithRetry(ctx context.Context, h store.Webhook, event string, data any) {
	var err error
	for attempt := 0; ; attempt++ {
		if err = s.Deliver(ctx, h, event, data); err == nil {
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
	slog.Warn("webhook delivery failed", "webhook", h.Name, "event", event, "err", err)
}

// targetsServarr reports whether rawURL's host:port is one of the configured Radarr/Sonarr instances.
func (s *Service) targetsServarr(ctx context.Context, rawURL string) bool {
	target, err := url.Parse(rawURL)
	if err != nil {
		return true
	}
	instances, err := s.store.ListServarr(ctx)
	if err != nil {
		return true // fail closed
	}
	for _, in := range instances {
		if u, err := url.Parse(in.URL); err == nil && strings.EqualFold(u.Host, target.Host) {
			return true
		}
	}
	return false
}

// Deliver performs one POST. It returns an error for transport failures and non-2xx answers.
func (s *Service) Deliver(ctx context.Context, h store.Webhook, event string, data any) error {
	if s.targetsServarr(ctx, h.URL) {
		// a webhook pointed at Radarr/Sonarr could write to them outside the dry-run choke point
		return errors.New("webhook URL points at a configured Radarr/Sonarr instance; refused")
	}
	body, err := json.Marshal(envelope{Event: event, Timestamp: time.Now().UTC().Format(time.RFC3339), DryRun: s.dryRun, Data: data})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Tipsarr")
	req.Header.Set("X-Tipsarr-Event", event)
	if h.Secret != "" {
		mac := hmac.New(sha256.New, []byte(h.Secret))
		mac.Write(body)
		req.Header.Set("X-Tipsarr-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("webhook answered HTTP %d", resp.StatusCode)
	}
	return nil
}
