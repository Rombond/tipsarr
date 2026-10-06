// Package library mirrors what Jellyfin has (for "available" badges) and who watched
// what (for suggestions). It only ever READS from Jellyfin.
package library

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/clients/jellyfin"
	"github.com/Rombond/tipsarr/backend/internal/store"
)

const (
	SettingJellyfinURL    = "jellyfin.url"
	SettingJellyfinAPIKey = "jellyfin.api_key"

	debounceDelay = 10 * time.Second
)

var ErrNoAPIKey = errors.New("Jellyfin API key is not configured")

type Service struct {
	// OnHistoryChanged (optional) is called for every user whose watch history changed during a sync.
	OnHistoryChanged func(userID string)

	store       *store.Store
	newJellyfin func(baseURL string) *jellyfin.Client
	debounce    time.Duration

	mu     sync.Mutex
	timers map[string]*time.Timer
}

func New(s *store.Store) *Service {
	return &Service{store: s, newJellyfin: jellyfin.New, debounce: debounceDelay, timers: map[string]*time.Timer{}}
}

func (s *Service) client(ctx context.Context) (*jellyfin.Client, error) {
	base, err := s.store.GetSetting(ctx, SettingJellyfinURL)
	if err != nil {
		return nil, err
	}
	key, err := s.store.GetSetting(ctx, SettingJellyfinAPIKey)
	if err != nil {
		return nil, err
	}
	if base == "" || key == "" {
		return nil, ErrNoAPIKey
	}
	return s.newJellyfin(base).WithToken(key), nil
}

// SetDebounce changes the webhook debounce window (tests).
func (s *Service) SetDebounce(d time.Duration) { s.debounce = d }

// Configured reports whether syncing is possible (URL + API key saved).
func (s *Service) Configured(ctx context.Context) bool {
	_, err := s.client(ctx)
	return err == nil
}

type LibraryResult struct {
	Movies, Shows, Episodes, Unmatched int
}

func (r LibraryResult) String() string {
	return fmt.Sprintf("%d movies, %d shows, %d episodes (%d without a TMDB id)", r.Movies, r.Shows, r.Episodes, r.Unmatched)
}

// SyncLibrary replaces the stored library snapshot with Jellyfin's current content.
func (s *Service) SyncLibrary(ctx context.Context) (LibraryResult, error) {
	var res LibraryResult
	jf, err := s.client(ctx)
	if err != nil {
		return res, err
	}

	items := map[string]store.LibraryItem{} // key: type+tmdb, dedupes multi-version items
	showByJellyfin := map[string]int64{}
	err = jf.Items(ctx, "", url.Values{
		"IncludeItemTypes": {"Movie,Series"}, "Fields": {"ProviderIds"}, "EnableUserData": {"false"},
	}, func(it jellyfin.Item) error {
		mt := map[string]string{"Movie": "movie", "Series": "tv"}[it.Type]
		id := it.TMDBID()
		if mt == "" {
			return nil
		}
		if id == 0 {
			res.Unmatched++
			return nil
		}
		items[fmt.Sprintf("%s:%d", mt, id)] = store.LibraryItem{MediaType: mt, TMDBID: int64(id), JellyfinID: it.ID, Title: it.Name}
		if mt == "tv" {
			showByJellyfin[it.ID] = int64(id)
		}
		return nil
	})
	if err != nil {
		return res, fmt.Errorf("list library: %w", err)
	}

	counts := map[[2]int64]int{} // (show tmdb, season) -> episodes
	err = jf.Items(ctx, "", url.Values{
		"IncludeItemTypes": {"Episode"}, "IsMissing": {"false"}, "EnableUserData": {"false"},
	}, func(it jellyfin.Item) error {
		if tmdb, ok := showByJellyfin[it.SeriesID]; ok {
			counts[[2]int64{tmdb, int64(it.ParentIndexNumber)}]++
			res.Episodes++
		}
		return nil
	})
	if err != nil {
		return res, fmt.Errorf("list episodes: %w", err)
	}

	rows := make([]store.LibraryItem, 0, len(items))
	for _, it := range items {
		rows = append(rows, it)
		if it.MediaType == "movie" {
			res.Movies++
		} else {
			res.Shows++
		}
	}
	seasons := make([]store.LibrarySeason, 0, len(counts))
	for k, n := range counts {
		seasons = append(seasons, store.LibrarySeason{TMDBID: k[0], SeasonNumber: int(k[1]), EpisodeCount: n})
	}
	return res, s.store.ReplaceLibrary(ctx, rows, seasons)
}

type HistoryResult struct {
	Users, Changed, Titles int
}

func (r HistoryResult) String() string {
	return fmt.Sprintf("%d users synced (%d changed), %d watched titles", r.Users, r.Changed, r.Titles)
}

// SyncHistory refreshes watch history for one Jellyfin user, or all users when userID is "".
func (s *Service) SyncHistory(ctx context.Context, userID string) (HistoryResult, error) {
	var res HistoryResult
	jf, err := s.client(ctx)
	if err != nil {
		return res, err
	}
	if m, t, err := s.store.LibraryCounts(ctx); err == nil && m+t == 0 {
		if _, err := s.SyncLibrary(ctx); err != nil { // episodes need the show id map
			return res, err
		}
	}
	showByJellyfin, err := s.store.LibraryJellyfinToTMDB(ctx, "tv")
	if err != nil {
		return res, err
	}

	var users []jellyfin.User
	if userID != "" {
		users = []jellyfin.User{{ID: userID}}
	} else if users, err = jf.Users(ctx); err != nil {
		return res, fmt.Errorf("list users: %w", err)
	}

	for _, u := range users {
		agg := map[string]*store.WatchHistory{}
		err := jf.Items(ctx, u.ID, url.Values{
			"IncludeItemTypes": {"Movie,Episode"}, "Filters": {"IsPlayed"}, "Fields": {"ProviderIds"},
		}, func(it jellyfin.Item) error {
			mt, tmdb := "movie", int64(it.TMDBID())
			if it.Type == "Episode" {
				mt, tmdb = "tv", showByJellyfin[it.SeriesID]
			}
			if tmdb == 0 {
				return nil
			}
			at := parseTime(it.UserData.LastPlayedDate)
			key := fmt.Sprintf("%s:%d", mt, tmdb)
			h := agg[key]
			if h == nil {
				h = &store.WatchHistory{MediaType: mt, TMDBID: tmdb}
				agg[key] = h
			}
			h.PlayCount += max(it.UserData.PlayCount, 1)
			h.LastPlayedAt = max(h.LastPlayedAt, at)
			return nil
		})
		if err != nil {
			return res, fmt.Errorf("history for user %s: %w", u.ID, err)
		}
		rows := make([]store.WatchHistory, 0, len(agg))
		for _, h := range agg {
			rows = append(rows, *h)
		}
		changed, err := s.store.ReplaceUserHistory(ctx, u.ID, rows)
		if err != nil {
			return res, err
		}
		res.Users++
		res.Titles += len(rows)
		if changed {
			res.Changed++
			if s.OnHistoryChanged != nil {
				s.OnHistoryChanged(u.ID)
			}
		}
	}
	return res, nil
}

func parseTime(v string) int64 {
	t, err := time.Parse(time.RFC3339Nano, v)
	if err != nil {
		return 0
	}
	return t.Unix()
}

// QueueUserHistory schedules a debounced history refresh (Jellyfin webhook bursts collapse into one).
// An empty userID refreshes every user.
func (s *Service) QueueUserHistory(userID string) {
	s.queue("history:"+userID, func(ctx context.Context) error {
		_, err := s.SyncHistory(ctx, userID)
		return err
	})
}

// QueueLibrary schedules a debounced library refresh.
func (s *Service) QueueLibrary() {
	s.queue("library", func(ctx context.Context) error {
		_, err := s.SyncLibrary(ctx)
		return err
	})
}

func (s *Service) queue(key string, fn func(context.Context) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, ok := s.timers[key]; ok {
		t.Stop()
	}
	s.timers[key] = time.AfterFunc(s.debounce, func() {
		s.mu.Lock()
		delete(s.timers, key)
		s.mu.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		if err := fn(ctx); err != nil {
			slog.Warn("queued sync failed", "job", key, "err", err)
		}
	})
}

// NormalizeID turns a Jellyfin GUID into the 32-hex form its REST API uses
// (webhooks render GUIDs with dashes, /Users returns them without).
func NormalizeID(id string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(id), "-", ""))
}

// ImportUsers adds Jellyfin users that Tipsarr has not seen yet (read-only on the Jellyfin side).
// It returns how many were created and the Jellyfin total.
func (s *Service) ImportUsers(ctx context.Context) (created, total int, err error) {
	jf, err := s.client(ctx)
	if err != nil {
		return 0, 0, err
	}
	users, err := jf.Users(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("list users: %w", err)
	}
	for _, u := range users {
		isNew, err := s.store.EnsureUser(ctx, NormalizeID(u.ID), u.Name, u.Policy.IsAdministrator)
		if err != nil {
			return created, len(users), err
		}
		if isNew {
			created++
		}
	}
	return created, len(users), nil
}
