package boxoffice

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/Rombond/tipsarr/backend/internal/clients/servarr"
	"github.com/Rombond/tipsarr/backend/internal/media"
	"github.com/Rombond/tipsarr/backend/internal/store"
)

const (
	SettingRegions = "boxoffice.regions"
	DefaultRegion  = "US" // US & Canada domestic chart (no area parameter)
	defaultBase    = "https://www.boxofficemojo.com"
	topN           = 10
	keepWeeks      = 8 // weekends of history kept per region
	userAgent      = "Mozilla/5.0 (compatible; Tipsarr; self-hosted media dashboard)"
	radarrCacheTTL = time.Minute
)

var regionRe = regexp.MustCompile(`^[A-Z]{2,3}$`)

type Service struct {
	store   *store.Store
	media   *media.Service
	baseURL string
	http    *http.Client
	pause   time.Duration // politeness delay between page fetches
	now     func() time.Time

	mu     sync.Mutex
	radarr map[string]radarrCache
}

type radarrCache struct {
	at     time.Time
	movies map[int]servarr.Movie
}

func New(s *store.Store, m *media.Service, baseURL string) *Service {
	if baseURL == "" {
		baseURL = defaultBase
	}
	return &Service{
		store: s, media: m, baseURL: strings.TrimRight(baseURL, "/"), http: &http.Client{Timeout: 25 * time.Second},
		pause: time.Second, now: time.Now, radarr: map[string]radarrCache{},
	}
}

// SetPause changes the delay between page fetches (tests).
func (s *Service) SetPause(d time.Duration) { s.pause = d }

// SetNow overrides the clock (tests).
func (s *Service) SetNow(f func() time.Time) { s.now = f }

// Regions returns the configured chart regions (default: US domestic only).
func (s *Service) Regions(ctx context.Context) []string {
	v, _ := s.store.GetSetting(ctx, SettingRegions)
	return ParseRegions(v)
}

// ParseRegions turns "US, GB,fr" into ["US","GB","FR"]; invalid codes are dropped.
func ParseRegions(v string) []string {
	var out []string
	seen := map[string]bool{}
	for _, r := range strings.Split(v, ",") {
		r = strings.ToUpper(strings.TrimSpace(r))
		if regionRe.MatchString(r) && !seen[r] {
			seen[r] = true
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return []string{DefaultRegion}
	}
	return out
}

// WeekKey returns the Box Office Mojo weekend key (e.g. "2026W40") of the most recent
// Sunday on or before t, offset weeks further back.
func WeekKey(t time.Time, back int) string {
	t = t.UTC()
	for t.Weekday() != time.Sunday {
		t = t.AddDate(0, 0, -1)
	}
	t = t.AddDate(0, 0, -7*back)
	y, w := t.ISOWeek()
	return fmt.Sprintf("%04dW%02d", y, w)
}

func (s *Service) pageURL(region, weekKey string) string {
	u := s.baseURL + "/weekend/" + weekKey + "/"
	if region != "" && region != DefaultRegion {
		u += "?area=" + url.QueryEscape(region)
	}
	return u
}

func (s *Service) fetch(ctx context.Context, region, weekKey string) (string, []Entry, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.pageURL(region, weekKey), nil)
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := s.http.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("box office source unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("box office source answered HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", nil, err
	}
	label, entries, err := ParseWeekend(body)
	if len(entries) > topN {
		entries = entries[:topN]
	}
	return label, entries, err
}

// Refresh fetches the latest chart(s) of every configured region and stores them. The latest
// two weekends are always re-read (Monday's actuals replace Sunday's estimates); older weekends
// are filled in until `keepWeeks` are stored, so the page can show a short history.
func (s *Service) Refresh(ctx context.Context) (string, error) {
	var saved, failed int
	var firstErr error
	for _, region := range s.Regions(ctx) {
		stored, err := s.store.BoxOfficeWeeks(ctx, region)
		if err != nil {
			return "", err
		}
		have := map[string]bool{}
		for _, w := range stored {
			have[w.WeekKey] = true
		}
		count := len(stored)
		fetched := 0
		for back := 0; back < keepWeeks+4; back++ { // a few extra tries for weekends without a chart
			key := WeekKey(s.now(), back)
			refresh := back < 2
			if !refresh && (have[key] || count >= keepWeeks) {
				if count >= keepWeeks {
					break
				}
				continue
			}
			if fetched > 0 {
				select {
				case <-time.After(s.pause):
				case <-ctx.Done():
					return "", ctx.Err()
				}
			}
			label, entries, err := s.fetch(ctx, region, key)
			if err != nil {
				if errors.Is(err, ErrNoChart) {
					continue // that weekend has no data (yet)
				}
				failed++
				if firstErr == nil {
					firstErr = fmt.Errorf("%s %s: %w", region, key, err)
				}
				break
			}
			fetched++
			if !have[key] {
				have[key] = true
				count++
			}
			if err := s.saveWeek(ctx, region, key, label, entries); err != nil {
				return "", err
			}
			saved++
		}
	}
	msg := fmt.Sprintf("%d charts stored", saved)
	if failed > 0 {
		msg += fmt.Sprintf(", %d failed", failed)
	}
	return msg, firstErr
}

func (s *Service) saveWeek(ctx context.Context, region, key, label string, entries []Entry) error {
	rows := make([]store.BoxOfficeEntry, 0, len(entries))
	for _, e := range entries {
		row := store.BoxOfficeEntry{
			Region: region, WeekKey: key, Pos: e.Pos, Title: e.Title, WeekendGross: e.WeekendGross,
			TotalGross: e.TotalGross, WeeksInRelease: e.WeeksInRelease,
		}
		if m := s.match(ctx, e.Title); m != nil {
			row.TMDBID, row.PosterPath, row.ReleaseDate = int64(m.TMDBID), m.PosterPath, m.ReleaseDate
			row.VoteTenths, row.Overview = int(m.VoteAverage*10), truncate(m.Overview, 400)
		}
		rows = append(rows, row)
	}
	return s.store.SaveBoxOffice(ctx, store.BoxOfficeWeek{Region: region, WeekKey: key, Label: label, FetchedAt: s.now().Unix()}, rows)
}

// match finds the TMDB movie for a chart title: exact title (case-insensitive) preferring
// recent releases, else the first recent result. Returns nil when TMDB is unconfigured/unreachable
// or nothing fits.
func (s *Service) match(ctx context.Context, title string) *media.Item {
	if id, err := s.store.BoxOfficeAlias(ctx, title); err == nil && id > 0 { // a manual match always wins
		if d, err := s.media.Detail(ctx, media.Opts{}, "movie", int(id)); err == nil {
			return &d.Item
		}
	}
	results, err := s.media.SearchMovies(ctx, media.Opts{}, title)
	if err != nil {
		if !errors.Is(err, media.ErrNotConfigured) {
			slog.Warn("box office: TMDB search failed", "title", title, "err", err)
		}
		return nil
	}
	recentYear := s.now().Year() - 2
	year := func(it media.Item) int {
		if len(it.ReleaseDate) >= 4 {
			var y int
			fmt.Sscanf(it.ReleaseDate[:4], "%d", &y)
			return y
		}
		return 0
	}
	var exactAny, recentAny *media.Item
	for i := range results {
		it := &results[i]
		if it.PosterPath == "" {
			continue
		}
		exact := strings.EqualFold(it.Title, title)
		recent := year(*it) >= recentYear
		switch {
		case exact && recent:
			return it
		case exact && exactAny == nil:
			exactAny = it
		case recent && recentAny == nil:
			recentAny = it
		}
	}
	if exactAny != nil {
		return exactAny
	}
	return recentAny
}

// truncate shortens s to at most n bytes at a word boundary, adding an ellipsis.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	cut := s[:n]
	if i := strings.LastIndexAny(cut, " \t\n"); i > n/2 {
		cut = cut[:i]
	}
	// never split a multi-byte character
	for len(cut) > 0 && !utf8.ValidString(cut) {
		cut = cut[:len(cut)-1]
	}
	return strings.TrimRight(cut, " ,.;:-") + "…"
}

// ---- reading -------------------------------------------------------------------------------

type ChartEntry struct {
	Position       int         `json:"position"`
	Title          string      `json:"title"`
	WeekendGross   int64       `json:"weekendGross" doc:"USD (or the region's currency) for the weekend"`
	TotalGross     int64       `json:"totalGross"`
	WeeksInRelease int         `json:"weeksInRelease"`
	Item           *media.Item `json:"item,omitempty" doc:"Matched TMDB title; absent when no match was found"`
	InRadarr       bool        `json:"inRadarr" doc:"Radarr already tracks this movie (a request would be redundant)"`
	HasFile        bool        `json:"hasFile" doc:"Radarr already has the file"`
}

type WeekRef struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type Chart struct {
	Region    string       `json:"region"`
	Regions   []string     `json:"regions"`
	Week      string       `json:"week" doc:"Week key, e.g. 2026W40; empty when nothing is stored yet"`
	Label     string       `json:"label,omitempty"`
	FetchedAt int64        `json:"fetchedAt"`
	Weeks     []WeekRef    `json:"weeks"`
	Entries   []ChartEntry `json:"entries"`
}

// Chart returns a stored chart. region/week may be empty: the first configured region (or the
// user's preferred one if configured) and the latest stored week are used.
func (s *Service) Chart(ctx context.Context, preferred, region, week string) (*Chart, error) {
	regions := s.Regions(ctx)
	if region == "" {
		region = regions[0]
		for _, r := range regions {
			if r == strings.ToUpper(preferred) {
				region = r
			}
		}
	}
	out := &Chart{Region: region, Regions: regions, Weeks: []WeekRef{}, Entries: []ChartEntry{}}
	weeks, err := s.store.BoxOfficeWeeks(ctx, region)
	if err != nil {
		return nil, err
	}
	for _, w := range weeks {
		out.Weeks = append(out.Weeks, WeekRef{Key: w.WeekKey, Label: w.Label})
	}
	if week == "" && len(weeks) > 0 {
		week = weeks[0].WeekKey
	}
	if week == "" {
		return out, nil
	}
	w, entries, err := s.store.BoxOffice(ctx, region, week)
	if errors.Is(err, store.ErrNotFound) {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	out.Week, out.Label, out.FetchedAt = w.WeekKey, w.Label, w.FetchedAt

	items := make([]media.Item, 0, len(entries))
	idx := map[int]int{} // entry index -> item index
	for i, e := range entries {
		if e.TMDBID == 0 {
			continue
		}
		idx[i] = len(items)
		items = append(items, media.Item{
			Type: "movie", TMDBID: int(e.TMDBID), Title: e.Title, PosterPath: e.PosterPath, ReleaseDate: e.ReleaseDate,
			VoteAverage: float64(e.VoteTenths) / 10, Overview: e.Overview, Availability: media.AvailabilityNone,
		})
	}
	s.media.Annotate(ctx, items)
	radarr := s.radarrMovies(ctx)
	for i, e := range entries {
		ce := ChartEntry{Position: e.Pos, Title: e.Title, WeekendGross: e.WeekendGross, TotalGross: e.TotalGross, WeeksInRelease: e.WeeksInRelease}
		if j, ok := idx[i]; ok {
			it := items[j]
			ce.Item = &it
			if m, ok := radarr[it.TMDBID]; ok {
				ce.InRadarr, ce.HasFile = true, m.HasFile
			}
		}
		out.Entries = append(out.Entries, ce)
	}
	return out, nil
}

// radarrMovies lists the movies the default Radarr tracks (cached briefly). Read-only; empty
// when no Radarr is configured or it is unreachable.
func (s *Service) radarrMovies(ctx context.Context) map[int]servarr.Movie {
	inst, err := s.store.DefaultServarr(ctx, servarr.KindRadarr)
	if err != nil {
		return nil
	}
	s.mu.Lock()
	if c, ok := s.radarr[inst.ID]; ok && time.Since(c.at) < radarrCacheTTL {
		s.mu.Unlock()
		return c.movies
	}
	s.mu.Unlock()
	movies, err := servarr.New(servarr.KindRadarr, inst.URL, inst.APIKey, true).Movies(ctx) // dry-run client: reads only
	if err != nil {
		slog.Warn("box office: radarr movie list failed", "err", err)
		return nil
	}
	m := make(map[int]servarr.Movie, len(movies))
	for _, mv := range movies {
		m[mv.TMDBID] = mv
	}
	s.mu.Lock()
	s.radarr[inst.ID] = radarrCache{at: time.Now(), movies: m}
	s.mu.Unlock()
	return m
}

var ErrBadTitle = errors.New("title is required")

// SetAlias pins a chart title to a TMDB movie and applies it to every stored week.
func (s *Service) SetAlias(ctx context.Context, title string, tmdbID int) error {
	if strings.TrimSpace(title) == "" {
		return ErrBadTitle
	}
	d, err := s.media.Detail(ctx, media.Opts{}, "movie", tmdbID)
	if err != nil {
		return err // media.ErrNotFound for an unknown id
	}
	if err := s.store.SetBoxOfficeAlias(ctx, title, int64(tmdbID)); err != nil {
		return err
	}
	return s.store.SetBoxOfficeMatch(ctx, title, entrySnapshot(&d.Item))
}

// RemoveAlias forgets a manual match and re-runs the automatic matching for that title.
func (s *Service) RemoveAlias(ctx context.Context, title string) error {
	if err := s.store.DeleteBoxOfficeAlias(ctx, title); err != nil {
		return err
	}
	var snap store.BoxOfficeEntry
	if m := s.match(ctx, title); m != nil {
		snap = entrySnapshot(m)
	}
	return s.store.SetBoxOfficeMatch(ctx, title, snap)
}

func entrySnapshot(m *media.Item) store.BoxOfficeEntry {
	return store.BoxOfficeEntry{
		TMDBID: int64(m.TMDBID), PosterPath: m.PosterPath, ReleaseDate: m.ReleaseDate,
		VoteTenths: int(m.VoteAverage * 10), Overview: truncate(m.Overview, 400),
	}
}
