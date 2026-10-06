// Package tmdb is a thin client for The Movie Database v3 API.
package tmdb

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	DefaultBaseURL  = "https://api.themoviedb.org/3"
	DefaultImageURL = "https://image.tmdb.org/t/p"
)

var ErrNotFound = errors.New("tmdb: not found")

type Client struct {
	BaseURL string
	Key     string // v3 API key, or a v4 read-access token (starts with "eyJ")
	http    *http.Client
}

func New(baseURL, key string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), Key: key, http: &http.Client{Timeout: 15 * time.Second}}
}

// Get performs a GET and returns the raw JSON body.
func (c *Client) Get(ctx context.Context, path string, q url.Values) ([]byte, error) {
	if q == nil {
		q = url.Values{}
	}
	bearer := strings.HasPrefix(c.Key, "eyJ")
	if !bearer {
		q.Set("api_key", c.Key)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if bearer {
		req.Header.Set("Authorization", "Bearer "+c.Key)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		// *url.Error embeds the full request URL, which carries the API key for v3 keys: unwrap it
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err
		}
		return nil, fmt.Errorf("tmdb unreachable: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return nil, ErrNotFound
	case resp.StatusCode == http.StatusUnauthorized:
		return nil, errors.New("tmdb: invalid API key")
	case resp.StatusCode >= 300:
		return nil, fmt.Errorf("tmdb returned HTTP %d", resp.StatusCode)
	}
	return body, nil
}
