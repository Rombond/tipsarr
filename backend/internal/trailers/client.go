// Package trailers talks to the optional trailer-resolver sidecar, a separate container that fetches trailer video
// files (see the vault note "Tipsarr - Plan Trailer Files"). Tipsarr only ever sends it an 11-character YouTube key,
// reads the finished files from a shared folder and serves them to the apps. Off by default.
package trailers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var keyRe = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

// ValidURL checks a resolver address typed by an admin.
func ValidURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("the resolver address must start with http:// or https://")
	}
	return nil
}

// ValidKey reports whether s looks like a YouTube video key.
func ValidKey(s string) bool { return keyRe.MatchString(s) }

var (
	ErrDisabled     = errors.New("trailer files are off")
	ErrBadKey       = errors.New("not a YouTube key")
	ErrUnreachable  = errors.New("the trailer resolver cannot be reached")
	ErrUnauthorized = errors.New("the trailer resolver refused the key")
)

// Status is what the resolver reports for one key.
type Status struct {
	Status    string `json:"status" enum:"queued,running,ready,failed"`
	Bytes     int64  `json:"bytes"`
	Height    int    `json:"height"`
	Duration  int    `json:"duration"`
	Error     string `json:"error,omitempty"`
	Permanent bool   `json:"permanent"`
}

// Counts are the resolver's queue and cache sizes.
type Counts struct {
	Queued  int `json:"queued"`
	Running int `json:"running"`
	Ready   int `json:"ready"`
	Failed  int `json:"failed"`
}

// Breaker is the resolver's protection against being blocked: open means it pauses until the given unix time.
type Breaker struct {
	State string `json:"state" enum:"closed,open"`
	Until int64  `json:"until"`
}

// Health is the resolver's own report.
type Health struct {
	OK              bool    `json:"ok"`
	YtdlpVersion    string  `json:"ytdlpVersion"`
	DenoVersion     string  `json:"denoVersion"`
	FFmpegVersion   string  `json:"ffmpegVersion"`
	CacheBytes      int64   `json:"cacheBytes"`
	CacheLimitBytes int64   `json:"cacheLimitBytes"`
	Counts          Counts  `json:"counts"`
	Breaker         Breaker `json:"breaker"`
	LastError       string  `json:"lastError,omitempty"`
}

// Client is a small HTTP client for the resolver API. Every call carries the shared key as a bearer token.
type Client struct {
	base string
	key  string
	http *http.Client
}

func NewClient(base, key string) *Client {
	return &Client{base: strings.TrimRight(base, "/"), key: key, http: &http.Client{Timeout: 10 * time.Second}}
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) (int, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, rdr)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err
		}
		return 0, fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		_, _ = io.Copy(io.Discard, resp.Body)
		return resp.StatusCode, ErrUnauthorized
	}
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		return resp.StatusCode, fmt.Errorf("trailer resolver answered %d %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return resp.StatusCode, nil
	}
	return resp.StatusCode, json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(out)
}

// Resolve queues a key (or reports it ready).
func (c *Client) Resolve(ctx context.Context, key string, maxHeight int) (*Status, error) {
	var st Status
	_, err := c.do(ctx, http.MethodPost, "/v1/resolve", map[string]any{"key": key, "maxHeight": maxHeight}, &st)
	return &st, err
}

// Status asks about one key; found is false when the resolver does not know it (a restart or a cleared cache).
func (c *Client) Status(ctx context.Context, key string) (st *Status, found bool, err error) {
	st = new(Status)
	code, err := c.do(ctx, http.MethodGet, "/v1/status/"+key, nil, st)
	if code == http.StatusNotFound {
		return nil, false, nil
	}
	return st, err == nil, err
}

func (c *Client) Health(ctx context.Context) (*Health, error) {
	var h Health
	_, err := c.do(ctx, http.MethodGet, "/v1/health", nil, &h)
	return &h, err
}

// Touch tells the resolver which files were played, so it evicts the least recently used ones first.
func (c *Client) Touch(ctx context.Context, keys []string) error {
	_, err := c.do(ctx, http.MethodPost, "/v1/touch", map[string]any{"keys": keys}, nil)
	return err
}

// UpdateResult is what the resolver reports after asking yt-dlp to update itself.
type UpdateResult struct {
	Before  string `json:"before"`
	After   string `json:"after"`
	Updated bool   `json:"updated"`
}

// Update asks the resolver to replace its yt-dlp binary with the newest release. It can take a while.
func (c *Client) Update(ctx context.Context) (*UpdateResult, error) {
	long := *c
	long.http = &http.Client{Timeout: 3 * time.Minute}
	var out UpdateResult
	_, err := long.do(ctx, http.MethodPost, "/v1/update", nil, &out)
	return &out, err
}

// ClearCache deletes every file the resolver holds.
func (c *Client) ClearCache(ctx context.Context) error {
	_, err := c.do(ctx, http.MethodDelete, "/v1/cache", nil, nil)
	return err
}
