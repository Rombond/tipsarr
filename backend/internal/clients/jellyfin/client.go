// Package jellyfin is a thin typed client for the parts of the Jellyfin API Tipsarr uses.
package jellyfin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var ErrInvalidCredentials = errors.New("invalid Jellyfin credentials")

const authHeader = `MediaBrowser Client="Tipsarr", Device="Tipsarr", DeviceId="tipsarr", Version="0.1.0"`

type Client struct {
	baseURL string
	token   string // optional API key; sent in the Authorization header
	http    *http.Client
}

// WithToken returns a copy of the client that authenticates with a Jellyfin API key.
func (c *Client) WithToken(token string) *Client {
	cp := *c
	cp.token = token
	return &cp
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

type PublicInfo struct {
	ServerName string `json:"ServerName"`
	Version    string `json:"Version"`
	ID         string `json:"Id"`
}

// PublicInfo calls the unauthenticated endpoint; used to validate a Jellyfin URL.
func (c *Client) PublicInfo(ctx context.Context) (*PublicInfo, error) {
	var out PublicInfo
	if err := c.do(ctx, http.MethodGet, "/System/Info/Public", nil, &out); err != nil {
		return nil, err
	}
	if out.ID == "" {
		return nil, errors.New("not a Jellyfin server (no server id in response)")
	}
	return &out, nil
}

type AuthResult struct {
	AccessToken string `json:"AccessToken"`
	User        struct {
		ID     string `json:"Id"`
		Name   string `json:"Name"`
		Policy struct {
			IsAdministrator bool `json:"IsAdministrator"`
		} `json:"Policy"`
	} `json:"User"`
}

// Authenticate checks a username/password against Jellyfin.
func (c *Client) Authenticate(ctx context.Context, username, password string) (*AuthResult, error) {
	body, _ := json.Marshal(map[string]string{"Username": username, "Pw": password})
	var out AuthResult
	err := c.do(ctx, http.MethodPost, "/Users/AuthenticateByName", body, &out)
	var se *statusError
	if errors.As(err, &se) && (se.code == http.StatusUnauthorized || se.code == http.StatusForbidden || se.code == http.StatusBadRequest) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if out.User.ID == "" {
		return nil, errors.New("unexpected Jellyfin response (no user id)")
	}
	return &out, nil
}

type statusError struct{ code int }

func (e *statusError) Error() string { return fmt.Sprintf("jellyfin returned HTTP %d", e.code) }

// IsNotFound reports whether err is Jellyfin answering 404.
func IsNotFound(err error) bool {
	var se *statusError
	return errors.As(err, &se) && se.code == http.StatusNotFound
}

func (c *Client) do(ctx context.Context, method, path string, body []byte, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	hdr := authHeader
	if c.token != "" {
		// Newer Jellyfin only accepts API keys inside the Authorization header (X-Emby-Token is rejected).
		hdr += `, Token="` + c.token + `"`
	}
	req.Header.Set("Authorization", hdr)
	req.Header.Set("X-Emby-Authorization", hdr)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("jellyfin unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return &statusError{code: resp.StatusCode}
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// Image downloads an item's primary image (at most maxWidth pixels wide) and returns its bytes and content type.
func (c *Client) Image(ctx context.Context, itemID, tag string, maxWidth int) ([]byte, string, error) {
	q := url.Values{"maxWidth": {strconv.Itoa(maxWidth)}, "quality": {"85"}}
	if tag != "" {
		q.Set("tag", tag)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/Items/"+url.PathEscape(itemID)+"/Images/Primary?"+q.Encode(), nil)
	if err != nil {
		return nil, "", err
	}
	hdr := authHeader
	if c.token != "" {
		hdr += `, Token="` + c.token + `"`
	}
	req.Header.Set("Authorization", hdr)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("jellyfin unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, "", &statusError{code: resp.StatusCode}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	return body, resp.Header.Get("Content-Type"), err
}
