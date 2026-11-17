// Package github looks up the newest Tipsarr release on GitHub.
package github

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

// latestReleaseURL is the GitHub API page of the newest published server release.
var latestReleaseURL = "https://api.github.com/repos/Rombond/tipsarr/releases/latest"

// latestCheckEvery is how long a lookup (or a failed one) is kept before asking GitHub again.
const latestCheckEvery = 6 * time.Hour

// Latest remembers the newest release tag seen on GitHub. Lookups run in the
// background so /status never waits on GitHub; until the first one ends it reports "".
type Latest struct {
	mu      sync.Mutex
	tag     string
	checked time.Time
	running bool
	client  *http.Client
	agent   string
}

// NewLatest builds a checker; version goes in the User-Agent.
func NewLatest(version string) *Latest {
	return &Latest{client: &http.Client{Timeout: 10 * time.Second}, agent: "tipsarr/" + version}
}

// Get returns the cached tag and starts a refresh when it is stale.
func (l *Latest) Get() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.running && time.Since(l.checked) > latestCheckEvery {
		l.running = true
		go l.refresh()
	}
	return l.tag
}

func (l *Latest) refresh() {
	tag := l.fetch()
	l.mu.Lock()
	defer l.mu.Unlock()
	if tag != "" {
		l.tag = tag
	}
	l.checked = time.Now()
	l.running = false
}

func (l *Latest) fetch() string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestReleaseURL, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", l.agent)
	resp, err := l.client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	var body struct {
		TagName string `json:"tag_name"`
	}
	if json.NewDecoder(resp.Body).Decode(&body) != nil {
		return ""
	}
	return strings.TrimSpace(body.TagName)
}
