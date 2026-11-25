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
	"strconv"
	"sync"
	"time"
)

// SettingLivePausedUntil (unix seconds) stops live updates until then, after the relay answered
// quota_exceeded for one. Alerts have their own quota and keep going.
const SettingLivePausedUntil = "push.live_paused_until"

// LiveItem is one live update for a followed request: an enum and numbers, never text (push-relay v0.5.0).
type LiveItem struct {
	Platform     string
	Token        string
	Sandbox      bool
	ID           string // request id
	Server       string
	Stage        string
	Percent      int
	ETASeconds   int
	Timestamp    int64
	End          bool
	DismissAfter int
	High         bool
}

// LiveResult is the relay's answer for one item. Status "skipped" means nothing was sent (push off,
// dry-run, paused, or a relay without live support).
type LiveResult struct {
	Status string
	Error  string
}

type liveItem struct {
	Platform string    `json:"platform"`
	Token    string    `json:"token"`
	Sandbox  bool      `json:"sandbox"`
	Event    string    `json:"event"`
	ID       string    `json:"id"`
	Server   string    `json:"server,omitempty"`
	Live     liveState `json:"live"`
}

type liveState struct {
	Stage        string `json:"stage"`
	Percent      int    `json:"percent"`
	ETASeconds   int    `json:"etaSeconds"`
	Timestamp    int64  `json:"timestamp"`
	End          bool   `json:"end"`
	DismissAfter int    `json:"dismissAfter"`
	Priority     string `json:"priority"`
}

var liveOff struct {
	mu    sync.Mutex
	until time.Time // a relay older than v0.5.0 answered 400: no live update before this
}

// SendLive posts live updates to the relay, synchronously, and answers one result per item.
func (s *Service) SendLive(ctx context.Context, items []LiveItem) ([]LiveResult, error) {
	out := make([]LiveResult, len(items))
	for i := range out {
		out[i].Status = "skipped"
	}
	cfg, ok := s.load(ctx)
	if !ok || len(items) == 0 {
		return out, nil
	}
	if s.dryRun {
		slog.Info("dry-run: live update not sent", "items", len(items))
		return out, nil
	}
	if until := s.livePausedUntil(ctx); until > 0 {
		return out, nil
	}
	liveOff.mu.Lock()
	off := time.Now().Before(liveOff.until)
	liveOff.mu.Unlock()
	if off {
		return out, nil
	}
	for start := 0; start < len(items); start += batchSize {
		end := min(start+batchSize, len(items))
		res, err := s.postLive(ctx, &cfg, items[start:end])
		if err != nil {
			return out, err
		}
		copy(out[start:end], res)
	}
	return out, nil
}

func (s *Service) postLive(ctx context.Context, cfg *config, items []LiveItem) ([]LiveResult, error) {
	res, err := s.postLiveOnce(ctx, *cfg, items)
	if errors.Is(err, errUnauthorized) && cfg.public {
		key, rerr := s.reregister(ctx, cfg.url, cfg.key)
		if rerr != nil {
			return nil, rerr
		}
		cfg.key = key
		res, err = s.postLiveOnce(ctx, *cfg, items)
	}
	var q *quotaError
	if errors.As(err, &q) {
		s.pauseLive(ctx, q.retryAfter)
		return skipped(len(items)), nil
	}
	if err != nil {
		return nil, err
	}
	return res, nil
}

func skipped(n int) []LiveResult {
	out := make([]LiveResult, n)
	for i := range out {
		out[i].Status = "skipped"
	}
	return out
}

func (s *Service) postLiveOnce(ctx context.Context, cfg config, items []LiveItem) ([]LiveResult, error) {
	body := make([]liveItem, len(items))
	for i, it := range items {
		prio := "low"
		if it.High {
			prio = "high"
		}
		body[i] = liveItem{Platform: it.Platform, Token: it.Token, Sandbox: it.Sandbox, Event: "live", ID: it.ID, Server: it.Server,
			Live: liveState{Stage: it.Stage, Percent: it.Percent, ETASeconds: it.ETASeconds, Timestamp: it.Timestamp,
				End: it.End, DismissAfter: it.DismissAfter, Priority: prio}}
	}
	raw, err := json.Marshal(map[string]any{"items": body})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.url+"/v1/push/batch", bytes.NewReader(raw))
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
	case http.StatusBadRequest:
		// the whole batch refused: a relay that does not know the live field
		io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		liveOff.mu.Lock()
		if time.Now().After(liveOff.until) {
			slog.Warn("push relay does not support live updates (needs push-relay v0.5.0)")
		}
		liveOff.until = time.Now().Add(time.Hour)
		liveOff.mu.Unlock()
		return skipped(len(items)), nil
	case http.StatusTooManyRequests:
		var r result
		_ = json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&r)
		if r.Status == "quota_exceeded" {
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
	res := skipped(len(items))
	for i := range res {
		if i < len(out.Items) {
			res[i] = LiveResult{Status: out.Items[i].Status, Error: out.Items[i].Error}
			if res[i].Status == "quota_exceeded" {
				res[i].Status = "skipped"
				s.pauseLive(ctx, max(out.Items[i].RetryAfter, 1))
			}
		}
	}
	return res, nil
}

func (s *Service) pauseLive(ctx context.Context, retryAfter int) {
	until := time.Now().Add(time.Duration(retryAfter) * time.Second).Unix()
	if err := s.store.SetSetting(ctx, SettingLivePausedUntil, strconv.FormatInt(until, 10)); err != nil {
		slog.Warn("push: save live pause", "err", err)
	}
	slog.Info("push: daily live quota reached, live updates paused", "until", time.Unix(until, 0).UTC())
}

func (s *Service) livePausedUntil(ctx context.Context) int64 {
	v, _ := s.store.GetSetting(ctx, SettingLivePausedUntil)
	n, _ := strconv.ParseInt(v, 10, 64)
	if n <= time.Now().Unix() {
		return 0
	}
	return n
}
