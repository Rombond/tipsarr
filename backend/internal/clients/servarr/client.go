// Package servarr talks to Radarr and Sonarr (API v3).
//
// SAFETY: every non-GET request goes through Client.send, the single choke point that
// refuses writes while dry-run is on. Nothing in this package (or its callers) can
// change Radarr/Sonarr state in dry-run mode, whatever code path is taken.
package servarr

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
	"time"
)

const (
	KindRadarr = "radarr"
	KindSonarr = "sonarr"
)

// ErrDryRun is returned instead of performing a write while dry-run is on.
var ErrDryRun = errors.New("dry-run: write not sent")

type Client struct {
	Kind    string
	baseURL string
	apiKey  string
	dryRun  bool
	http    *http.Client
}

func New(kind, baseURL, apiKey string, dryRun bool) *Client {
	return &Client{
		Kind: kind, baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey, dryRun: dryRun,
		http: &http.Client{
			Timeout: 20 * time.Second,
			// never follow redirects: the API key header must not travel to another host
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

type statusError struct {
	code int
	body string
}

func (e *statusError) Error() string {
	if e.body != "" {
		return fmt.Sprintf("HTTP %d: %s", e.code, e.body)
	}
	return fmt.Sprintf("HTTP %d", e.code)
}

// IsNotFound reports whether err is an HTTP 404 from the server.
func IsNotFound(err error) bool {
	var se *statusError
	return errors.As(err, &se) && se.code == http.StatusNotFound
}

// send is the only function that performs HTTP requests.
func (c *Client) send(ctx context.Context, method, path string, body, out any) error {
	if method != http.MethodGet && c.dryRun {
		slog.Info("dry-run: blocked write", "target", c.Kind, "method", method, "path", path)
		return ErrDryRun
	}
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+"/api/v3"+path, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("X-Api-Key", c.apiKey)
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
		return fmt.Errorf("%s unreachable: %w", c.Kind, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 400))
		return &statusError{code: resp.StatusCode, body: strings.TrimSpace(string(b))}
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	return c.send(ctx, http.MethodGet, path, nil, out)
}

// ---- shared reads ----------------------------------------------------------------

type Status struct {
	AppName string `json:"appName"`
	Version string `json:"version"`
}

func (c *Client) Status(ctx context.Context) (*Status, error) {
	var out Status
	return &out, c.get(ctx, "/system/status", &out)
}

type QualityProfile struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type RootFolder struct {
	ID        int    `json:"id"`
	Path      string `json:"path"`
	FreeSpace int64  `json:"freeSpace"`
}

type Tag struct {
	ID    int    `json:"id"`
	Label string `json:"label"`
}

func (c *Client) QualityProfiles(ctx context.Context) ([]QualityProfile, error) {
	out := []QualityProfile{}
	return out, c.get(ctx, "/qualityprofile", &out)
}

func (c *Client) RootFolders(ctx context.Context) ([]RootFolder, error) {
	out := []RootFolder{}
	return out, c.get(ctx, "/rootfolder", &out)
}

func (c *Client) Tags(ctx context.Context) ([]Tag, error) {
	out := []Tag{}
	return out, c.get(ctx, "/tag", &out)
}

// QueueItem is one active download. MediaID is the Radarr movie id / Sonarr series id.
type QueueItem struct {
	MediaID  int
	Title    string
	Size     float64
	SizeLeft float64
	TimeLeft string // "hh:mm:ss" as reported by the app
	Status   string
}

// Percent is the download progress 0-100.
func (q QueueItem) Percent() int {
	if q.Size <= 0 {
		return 0
	}
	return int((q.Size - q.SizeLeft) / q.Size * 100)
}

// ETASeconds parses TimeLeft ("1.02:03:04" or "02:03:04"); 0 when unknown.
func (q QueueItem) ETASeconds() int {
	t := q.TimeLeft
	days := 0
	if i := strings.Index(t, "."); i >= 0 && i < strings.Index(t, ":") {
		fmt.Sscanf(t[:i], "%d", &days)
		t = t[i+1:]
	}
	var h, m, s int
	if n, _ := fmt.Sscanf(t, "%d:%d:%d", &h, &m, &s); n != 3 {
		return 0
	}
	return days*86400 + h*3600 + m*60 + s
}

type queueResponse struct {
	Records []struct {
		MovieID  int     `json:"movieId"`
		SeriesID int     `json:"seriesId"`
		Title    string  `json:"title"`
		Size     float64 `json:"size"`
		SizeLeft float64 `json:"sizeleft"`
		TimeLeft string  `json:"timeleft"`
		Status   string  `json:"status"`
	} `json:"records"`
	TotalRecords int `json:"totalRecords"`
}

// Queue lists active downloads (first 500).
func (c *Client) Queue(ctx context.Context) ([]QueueItem, error) {
	var resp queueResponse
	if err := c.get(ctx, "/queue?page=1&pageSize=500", &resp); err != nil {
		return nil, err
	}
	out := make([]QueueItem, 0, len(resp.Records))
	for _, r := range resp.Records {
		id := r.MovieID
		if c.Kind == KindSonarr {
			id = r.SeriesID
		}
		if id == 0 {
			continue
		}
		out = append(out, QueueItem{MediaID: id, Title: r.Title, Size: r.Size, SizeLeft: r.SizeLeft, TimeLeft: r.TimeLeft, Status: r.Status})
	}
	return out, nil
}

// DeleteMedia removes a movie (Radarr) or series (Sonarr) from the app. Files on disk are kept.
// Like every write it goes through send, so dry-run blocks it.
func (c *Client) DeleteMedia(ctx context.Context, id int) error {
	if c.Kind == KindSonarr {
		return c.send(ctx, http.MethodDelete, fmt.Sprintf("/series/%d?deleteFiles=false&addImportListExclusion=false", id), nil, nil)
	}
	return c.send(ctx, http.MethodDelete, fmt.Sprintf("/movie/%d?deleteFiles=false&addImportExclusion=false", id), nil, nil)
}

// AggregateQueue folds the queue into one entry per movie/series: a show downloading several
// episodes or season packs at once is one download as far as progress goes (sizes add up, the
// ETA is the longest one). Without this the last queue record would win, and a record that has
// just started shows the whole show at 0%.
func AggregateQueue(items []QueueItem) map[int]QueueItem {
	out := map[int]QueueItem{}
	for _, q := range items {
		cur, ok := out[q.MediaID]
		if !ok {
			out[q.MediaID] = q
			continue
		}
		if q.ETASeconds() > cur.ETASeconds() {
			cur.TimeLeft = q.TimeLeft
		}
		cur.Size += q.Size
		cur.SizeLeft += q.SizeLeft
		out[q.MediaID] = cur
	}
	return out
}
