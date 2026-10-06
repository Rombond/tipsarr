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
	"strings"
	"time"
)

var ErrInvalidCredentials = errors.New("invalid Jellyfin credentials")

const authHeader = `MediaBrowser Client="Tipsarr", Device="Tipsarr", DeviceId="tipsarr", Version="0.1.0"`

type Client struct {
	baseURL string
	http    *http.Client
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

func (c *Client) do(ctx context.Context, method, path string, body []byte, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("X-Emby-Authorization", authHeader)
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
