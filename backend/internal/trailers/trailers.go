package trailers

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/i18n"
	"github.com/Rombond/tipsarr/backend/internal/jobs"
	"github.com/Rombond/tipsarr/backend/internal/store"
)

// Settings keys.
const (
	SettingEnabled   = "teasers.files.enabled" // "true" when on (off by default)
	SettingURL       = "teasers.files.url"
	SettingKey       = "teasers.files.key" // secret, never returned by the API
	SettingMaxHeight = "teasers.files.max_height"

	DefaultMaxHeight = 720
	// DemoKey is a short public video the admin "test" button resolves when no key is given.
	DemoKey = "jNQXAC9IVRw"
)

const (
	healthTTL    = 30 * time.Second
	healthGrace  = 10 * time.Minute // a failed health check keeps the last good answer this long (a slow or restarting resolver)
	maxAttempts  = 8
	finalFor     = 7 * 24 * time.Hour // a permanent failure is asked again after this (when the resolver gives no time)
	retryWindow  = 24 * time.Hour     // a failed key is retried only if the feed asked for it within this time
	touchWindow  = 2 * time.Minute
	syncBatch    = 60
	requestedGap = 10 * time.Minute // how often an already known key refreshes its "asked for" time
)

// MaxHeights are the sizes an admin can choose.
var MaxHeights = []int{360, 480, 720, 1080}

type config struct {
	url       string
	key       string
	maxHeight int
}

// Service holds the trailer files state: settings, the resolver client, the shared folder and the database rows.
type Service struct {
	store *store.Store
	dir   string
	now   func() time.Time

	mu         sync.Mutex
	health     *Health
	healthErr  string
	healthAt   time.Time
	healthBase string
	goodHealth *Health // last answer from the resolver, for healthGrace
	goodAt     time.Time
	goodBase   string // address and key that answered goodHealth

	fetchMu sync.Mutex // one health call at a time when the cache expires
}

// New builds the service. dir is the shared folder the resolver writes <key>.mp4 files into (mounted read-only here).
func New(st *store.Store, dir string) *Service {
	return &Service{store: st, dir: dir, now: time.Now}
}

// SetNow replaces the clock (tests).
func (s *Service) SetNow(f func() time.Time) { s.now = f }

// Dir is the shared folder holding the files.
func (s *Service) Dir() string { return s.dir }

func (s *Service) load(ctx context.Context) (config, bool) {
	on, _ := s.store.GetSetting(ctx, SettingEnabled)
	u, _ := s.store.GetSetting(ctx, SettingURL)
	k, _ := s.store.GetSetting(ctx, SettingKey)
	if on != "true" || u == "" || k == "" {
		return config{}, false
	}
	h := DefaultMaxHeight
	if v, _ := s.store.GetSetting(ctx, SettingMaxHeight); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			h = n
		}
	}
	return config{url: u, key: k, maxHeight: h}, true
}

func (s *Service) client(ctx context.Context) (*Client, config, error) {
	cfg, ok := s.load(ctx)
	if !ok {
		return nil, cfg, ErrDisabled
	}
	return NewClient(cfg.url, cfg.key), cfg, nil
}

// Enabled reports whether the feature is switched on with an address and a key. A nil service is off.
func (s *Service) Enabled(ctx context.Context) bool {
	if s == nil {
		return false
	}
	_, ok := s.load(ctx)
	return ok
}

// UpdateTools is the job that keeps yt-dlp current: YouTube changes often and an old yt-dlp stops working. It asks
// the resolver to update its binary and replaces it directly when a newer release exists. Off when trailer files are off.
func (s *Service) UpdateTools(ctx context.Context) (i18n.Msg, error) {
	c, _, err := s.client(ctx)
	if err != nil {
		return i18n.M("srv.job.skip.teasers_off"), jobs.ErrSkipped
	}
	res, err := c.Update(ctx)
	if err != nil {
		return i18n.Msg{}, err
	}
	s.ResetHealth() // the next status check shows the new version
	if res.Updated || (res.Before != "" && res.Before != res.After) {
		return i18n.M("srv.job.ytdlp.updated", "from", res.Before, "to", res.After), nil
	}
	return i18n.M("srv.job.ytdlp.current", "version", res.After), nil
}

// Health asks the resolver now.
func (s *Service) Health(ctx context.Context) (*Health, error) {
	c, _, err := s.client(ctx)
	if err != nil {
		return nil, err
	}
	return c.Health(ctx)
}

// CachedHealth is Health remembered for 30 seconds, so /status stays cheap. The text is the last error ("" when fine).
func (s *Service) CachedHealth(ctx context.Context) (*Health, string) {
	cfg, ok := s.load(ctx)
	if !ok {
		return nil, ErrDisabled.Error()
	}
	cached := func() (*Health, string, bool) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.healthBase == cfg.url && s.now().Sub(s.healthAt) < healthTTL {
			return s.health, s.healthErr, true
		}
		return nil, "", false
	}
	if h, e, ok := cached(); ok {
		return h, e
	}
	s.fetchMu.Lock()
	defer s.fetchMu.Unlock()
	if h, e, ok := cached(); ok { // another caller refreshed it while this one waited
		return h, e
	}
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	h, err := NewClient(cfg.url, cfg.key).Health(ctx)
	msg := ""
	switch {
	case err != nil:
		h, msg = nil, err.Error()
		s.mu.Lock()
		if s.goodHealth != nil && s.goodBase == cfg.url+"\x00"+cfg.key && s.now().Sub(s.goodAt) < healthGrace {
			h = s.goodHealth // one slow answer must not hide the Teasers tab
		}
		s.mu.Unlock()
	case !h.OK:
		msg = "the resolver reports a problem"
		if h.LastError != "" {
			msg = h.LastError
		}
	}
	s.mu.Lock()
	if err == nil {
		s.goodHealth, s.goodAt, s.goodBase = h, s.now(), cfg.url+"\x00"+cfg.key
	}
	s.health, s.healthErr, s.healthAt, s.healthBase = h, msg, s.now(), cfg.url
	s.mu.Unlock()
	return h, msg
}

// Available is Enabled and the resolver answered its health check lately (/status features.teasers): in the last
// 30 seconds, or within healthGrace when the latest check failed.
// Its own verdict (h.OK) is not needed: with the circuit breaker open or yt-dlp broken it cannot download, but the
// files it already holds still play, so the Teasers tab stays.
func (s *Service) Available(ctx context.Context) bool {
	if !s.Enabled(ctx) {
		return false
	}
	h, _ := s.CachedHealth(ctx)
	return h != nil
}

// ResetHealth forgets the cached health (settings changed, tests).
func (s *Service) ResetHealth() {
	s.mu.Lock()
	s.healthAt = time.Time{}
	s.mu.Unlock()
}

// Path is where the file of a valid key lives.
func (s *Service) Path(key string) string { return filepath.Join(s.dir, key+".mp4") }

// fileOK reports whether the finished file is visible in the shared folder.
func (s *Service) fileOK(key string) bool {
	fi, err := os.Lstat(s.Path(key))
	return err == nil && fi.Mode().IsRegular() && fi.Size() > 0
}

// Ready returns the row of a key whose file is finished and visible here.
func (s *Service) Ready(ctx context.Context, key string) (*store.TrailerFile, bool) {
	if s == nil || !ValidKey(key) {
		return nil, false
	}
	row, err := s.store.TrailerFileByKey(ctx, key)
	if err != nil || row == nil || row.Status != store.TrailerReady || !s.fileOK(key) {
		return nil, false
	}
	return row, true
}

// Pending reports whether a key can still become a file: false only for a video that failed for good.
func (s *Service) Pending(ctx context.Context, key string) bool {
	row, err := s.store.TrailerFileByKey(ctx, key)
	if err != nil {
		return false
	}
	if row == nil || !(row.Status == store.TrailerFailed && row.Permanent == 1) {
		return true
	}
	return s.now().Unix() >= row.NextTryAt // a final failure whose time is over: Ensure starts it afresh
}

// KeepWarm tells the resolver these files were just used, so its cache clean-up deletes other files first.
func (s *Service) KeepWarm(ctx context.Context, keys []string) error {
	c, _, err := s.client(ctx)
	if err != nil {
		return err
	}
	for len(keys) > 0 {
		n := min(len(keys), 200)
		if err := c.Touch(ctx, keys[:n]); err != nil {
			return err
		}
		keys = keys[n:]
	}
	return nil
}

// Touch records that a person played the file (at most once every few minutes per key).
func (s *Service) Touch(ctx context.Context, row *store.TrailerFile) {
	now := s.now().Unix()
	if now-row.LastUsedAt < int64(touchWindow/time.Second) {
		return
	}
	if err := s.store.TouchTrailerFile(ctx, row.YoutubeKey, now); err != nil {
		slog.Warn("touch trailer file", "key", row.YoutubeKey, "err", err)
	}
}

func backoff(attempts int) time.Duration {
	d := 5 * time.Minute << min(max(attempts-1, 0), 6)
	return min(d, 6*time.Hour)
}

func clip(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 250 {
		s = s[:250]
	}
	return s
}

// markFailed records a failure. count false leaves Attempts alone (a problem on this side, not the video's).
// A permanent failure waits until retryAt (from the resolver) or finalFor, then Ensure may ask again.
func (s *Service) markFailed(row *store.TrailerFile, msg string, permanent, count bool, retryAt int64) {
	now := s.now()
	row.Status = store.TrailerFailed
	row.Error = clip(msg)
	if count {
		row.Attempts++
	}
	if permanent || row.Attempts >= maxAttempts {
		row.Permanent = 1
		switch {
		case permanent && retryAt > now.Unix():
			row.NextTryAt = retryAt
		case permanent && retryAt > 0:
			// the resolver's time is already over here (clocks a little apart): ask again soon, not in 7 days
			row.NextTryAt = now.Add(backoff(1)).Unix()
		default:
			row.NextTryAt = now.Add(finalFor).Unix()
		}
		return
	}
	row.NextTryAt = now.Add(backoff(max(row.Attempts, 1))).Unix()
}

// apply copies what the resolver reported into the row.
func (s *Service) apply(row *store.TrailerFile, st *Status) {
	switch st.Status {
	case store.TrailerReady:
		if !s.fileOK(row.YoutubeKey) {
			// a mount problem on this side: not the video's fault, so it never makes the key final
			s.markFailed(row, "the file is not visible in the shared folder", false, false, 0)
			return
		}
		row.Status, row.Bytes, row.Height, row.Duration = store.TrailerReady, st.Bytes, st.Height, st.Duration
		row.Error, row.Permanent, row.Attempts, row.ReadyAt = "", 0, 0, s.now().Unix()
	case store.TrailerFailed:
		s.markFailed(row, st.Error, st.Permanent, true, st.RetryAt)
	case store.TrailerRunning:
		row.Status = store.TrailerRunning
	default:
		row.Status = store.TrailerQueued
	}
}

// Ensure makes sure the resolver has been asked for a key. It never waits for the download: a queued key is picked up
// by the sync job. A key that is already ready, in progress, or permanently failed costs no resolver call.
func (s *Service) Ensure(ctx context.Context, key string) error {
	if !ValidKey(key) {
		return ErrBadKey
	}
	c, cfg, err := s.client(ctx)
	if err != nil {
		return err
	}
	now := s.now().Unix()
	row, err := s.store.TrailerFileByKey(ctx, key)
	if err != nil {
		return err
	}
	fresh := false
	if row == nil {
		if fresh, err = s.store.AddTrailerFile(ctx, key, now); err != nil {
			return err
		}
		if row, err = s.store.TrailerFileByKey(ctx, key); err != nil || row == nil {
			return err
		}
	}
	askAgain := fresh
	switch row.Status {
	case store.TrailerReady:
		if s.fileOK(key) {
			return s.refreshRequested(ctx, row, now)
		}
		// evicted by the resolver or not visible: fetch it again (an eviction is no failure, Attempts stays)
		row.Status = store.TrailerQueued
		askAgain = true
	case store.TrailerFailed:
		if now < row.NextTryAt {
			return nil
		}
		if row.Permanent == 1 { // its time is over: a fresh start
			row.Permanent, row.Attempts = 0, 0
		}
		row.Status = store.TrailerQueued
		askAgain = true
	}
	row.RequestedAt = now
	if askAgain {
		st, err := c.Resolve(ctx, key, cfg.maxHeight)
		if err != nil {
			row.NextTryAt = now + 30
			_ = s.store.SaveTrailerFile(ctx, row)
			return err
		}
		s.apply(row, st)
	}
	return s.store.SaveTrailerFile(ctx, row)
}

func (s *Service) refreshRequested(ctx context.Context, row *store.TrailerFile, now int64) error {
	if now-row.RequestedAt < int64(requestedGap/time.Second) {
		return nil
	}
	row.RequestedAt = now
	return s.store.SaveTrailerFile(ctx, row)
}

// EnsureMany calls Ensure for each key and stops at the first resolver error (a dead resolver is not asked 24 times).
func (s *Service) EnsureMany(ctx context.Context, keys []string) error {
	for _, k := range keys {
		if err := s.Ensure(ctx, k); err != nil {
			if errors.Is(err, ErrBadKey) {
				continue
			}
			return err
		}
	}
	return nil
}

// Sync is the background job: it follows queued and running keys, retries temporary failures that were asked for
// recently, and tells the resolver which files were played.
func (s *Service) Sync(ctx context.Context) (i18n.Msg, error) {
	c, cfg, err := s.client(ctx)
	if err != nil {
		return i18n.M("srv.job.skip.teasers_off"), jobs.ErrSkipped
	}
	now := s.now().Unix()
	var firstErr error
	note := func(err error) {
		if firstErr == nil {
			firstErr = err
		}
	}

	active, err := s.store.TrailerFilesByStatus(ctx, syncBatch, store.TrailerQueued, store.TrailerRunning)
	if err != nil {
		return i18n.Msg{}, err
	}
	for i := range active {
		row := &active[i]
		st, found, err := c.Status(ctx, row.YoutubeKey)
		if err != nil {
			note(err)
			if errors.Is(err, ErrUnreachable) || errors.Is(err, ErrUnauthorized) {
				break
			}
			continue
		}
		if !found { // the resolver restarted or cleared its cache: ask again
			if st, err = c.Resolve(ctx, row.YoutubeKey, cfg.maxHeight); err != nil {
				note(err)
				if errors.Is(err, ErrUnreachable) || errors.Is(err, ErrUnauthorized) {
					break
				}
				continue
			}
		}
		s.apply(row, st)
		if err := s.store.SaveTrailerFile(ctx, row); err != nil {
			note(err)
		}
	}

	retry, err := s.store.TrailerFilesToRetry(ctx, now, now-int64(retryWindow/time.Second), syncBatch)
	if err != nil {
		note(err)
	}
	for i := range retry {
		row := &retry[i]
		row.Status = store.TrailerQueued // RequestedAt stays: only the feed asking renews the retry window
		st, err := c.Resolve(ctx, row.YoutubeKey, cfg.maxHeight)
		if err != nil {
			row.NextTryAt = now + 60
			_ = s.store.SaveTrailerFile(ctx, row)
			note(err)
			if errors.Is(err, ErrUnreachable) || errors.Is(err, ErrUnauthorized) {
				break
			}
			continue
		}
		s.apply(row, st)
		if err := s.store.SaveTrailerFile(ctx, row); err != nil {
			note(err)
		}
	}

	if keys, err := s.store.RecentTrailerKeys(ctx, now-int64(touchWindow/time.Second), 200); err == nil && len(keys) > 0 {
		if err := c.Touch(ctx, keys); err != nil {
			note(err)
		}
	}

	counts, _ := s.store.TrailerCounts(ctx)
	msg := i18n.M("srv.job.trailers.synced", "ready", counts[store.TrailerReady], "pending", counts[store.TrailerQueued]+counts[store.TrailerRunning], "failed", counts[store.TrailerFailed])
	return msg, firstErr
}

// Overview is what the admin card shows.
type Overview struct {
	Enabled    bool           `json:"enabled"`
	Configured bool           `json:"configured" doc:"An address and a key are saved"`
	Available  bool           `json:"available" doc:"The resolver answered its health check and reports no problem (the Teasers tab only needs an answer)"`
	Error      string         `json:"error,omitempty" doc:"Why the resolver is not available"`
	Health     *Health        `json:"health,omitempty"`
	Files      map[string]int `json:"files" doc:"Rows per state in Tipsarr's own table: queued, running, ready, failed"`
}

func (s *Service) Overview(ctx context.Context) (*Overview, error) {
	u, _ := s.store.GetSetting(ctx, SettingURL)
	k, _ := s.store.GetSetting(ctx, SettingKey)
	on, _ := s.store.GetSetting(ctx, SettingEnabled)
	o := &Overview{Enabled: on == "true", Configured: u != "" && k != ""}
	files, err := s.store.TrailerCounts(ctx)
	if err != nil {
		return nil, err
	}
	o.Files = map[string]int{store.TrailerQueued: files[store.TrailerQueued], store.TrailerRunning: files[store.TrailerRunning], store.TrailerReady: files[store.TrailerReady], store.TrailerFailed: files[store.TrailerFailed]}
	if o.Configured {
		// the settings page wants the truth now, not a 30 s old answer; the address works even while the feature is off
		hctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		h, err := NewClient(u, k).Health(hctx)
		switch {
		case err != nil:
			o.Error = err.Error()
		case !h.OK:
			o.Health, o.Error = h, "the resolver reports a problem"
			switch {
			case h.Breaker.State == "open" && h.BreakerError != "":
				o.Error = h.BreakerError // why it paused, not the last video that failed
			case h.LastError != "":
				o.Error = h.LastError
			}
		default:
			o.Health, o.Available = h, o.Enabled
		}
	}
	return o, nil
}

// TestResult is what the admin "test" button reports.
type TestResult struct {
	Key      string  `json:"key"`
	Status   string  `json:"status" enum:"queued,running,ready,failed"`
	Seconds  float64 `json:"seconds" doc:"How long the resolver took (or how long we waited)"`
	Bytes    int64   `json:"bytes"`
	Height   int     `json:"height"`
	Duration int     `json:"duration"`
	Error    string  `json:"error,omitempty"`
}

// Test resolves one video through the resolver and waits up to wait for it. It uses the saved address and key even
// while the feature is off, so an admin can check a new set-up before turning it on.
func (s *Service) Test(ctx context.Context, key string, wait time.Duration) (*TestResult, error) {
	if key == "" {
		key = DemoKey
	}
	if !ValidKey(key) {
		return nil, ErrBadKey
	}
	u, _ := s.store.GetSetting(ctx, SettingURL)
	k, _ := s.store.GetSetting(ctx, SettingKey)
	if u == "" || k == "" {
		return nil, ErrDisabled
	}
	h := DefaultMaxHeight
	if v, _ := s.store.GetSetting(ctx, SettingMaxHeight); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			h = n
		}
	}
	c := NewClient(u, k)
	start := s.now()
	st, err := c.Resolve(ctx, key, h)
	if err != nil {
		return nil, err
	}
	for st.Status == store.TrailerQueued || st.Status == store.TrailerRunning {
		if s.now().Sub(start) >= wait {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
		next, found, err := c.Status(ctx, key)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, errors.New("the resolver forgot the key")
		}
		st = next
	}
	res := &TestResult{Key: key, Status: st.Status, Seconds: s.now().Sub(start).Seconds(), Bytes: st.Bytes, Height: st.Height, Duration: st.Duration, Error: st.Error}
	if res.Status == store.TrailerReady && !s.fileOK(key) {
		res.Status, res.Error = store.TrailerFailed, "the file is not visible in the shared folder "+s.dir
	}
	return res, nil
}

// Clear empties the resolver's cache and forgets every row. The rows stay if the resolver cannot be reached.
func (s *Service) Clear(ctx context.Context) error {
	u, _ := s.store.GetSetting(ctx, SettingURL)
	k, _ := s.store.GetSetting(ctx, SettingKey)
	if u == "" || k == "" {
		return ErrDisabled
	}
	if err := NewClient(u, k).ClearCache(ctx); err != nil {
		return err
	}
	return s.store.DeleteTrailerFiles(ctx)
}
