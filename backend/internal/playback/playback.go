// Package playback copies the play sessions of Jellyfin's optional Playback Reporting plugin
// into Tipsarr, so the Stats page can show real watch time and timelines. Nothing here writes
// to Jellyfin; Tipsarr only ever sends its own fixed SELECT queries to the plugin.
package playback

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/clients/jellyfin"
	"github.com/Rombond/tipsarr/backend/internal/store"
)

const (
	// SettingPlugin remembers what the last sync found: "installed" or "missing" (empty = never checked).
	SettingPlugin = "playback.plugin"

	batch = 1000
)

var (
	ErrNoAPIKey     = errors.New("Jellyfin API key is not configured")
	ErrNotInstalled = errors.New("the Playback Reporting plugin is not installed in Jellyfin")

	episodeSuffix = regexp.MustCompile(`\s+-\s+s\d+e\d+.*$`)
)

type Service struct {
	store       *store.Store
	newJellyfin func(baseURL string) *jellyfin.Client
}

func New(s *store.Store) *Service { return &Service{store: s, newJellyfin: jellyfin.New} }

func (s *Service) client(ctx context.Context) (*jellyfin.Client, error) {
	base, _ := s.store.GetSetting(ctx, "jellyfin.url")
	key, _ := s.store.GetSetting(ctx, "jellyfin.api_key")
	if base == "" || key == "" {
		return nil, ErrNoAPIKey
	}
	return s.newJellyfin(base).WithToken(key), nil
}

// Sync copies the plugin's new play sessions. It returns a short summary.
func (s *Service) Sync(ctx context.Context) (string, error) {
	jf, err := s.client(ctx)
	if err != nil {
		return "", err
	}
	present, err := jf.PlaybackReportingInstalled(ctx)
	if err != nil {
		return "", err
	}
	state := "missing"
	if present {
		state = "installed"
	}
	if cur, _ := s.store.GetSetting(ctx, SettingPlugin); cur != state {
		_ = s.store.SetSetting(ctx, SettingPlugin, state)
	}
	if !present {
		return "", ErrNotInstalled
	}

	last, err := s.store.MaxWatchEventRow(ctx)
	if err != nil {
		return "", err
	}
	// the plugin's database was reset or recreated: start over rather than mix two histories
	_, head, err := jf.PlaybackQuery(ctx, "SELECT COALESCE(MAX(rowid), 0) FROM PlaybackActivity")
	if err != nil {
		return "", err
	}
	if len(head) == 1 && len(head[0]) == 1 {
		if remote, _ := strconv.ParseInt(head[0][0], 10, 64); remote < last {
			if err := s.store.DeleteAllWatchEvents(ctx); err != nil {
				return "", err
			}
			last = 0
		}
	}

	lib, err := s.store.AllLibrary(ctx)
	if err != nil {
		return "", err
	}
	byJF := map[string]store.LibraryItem{}
	for _, it := range lib {
		byJF[it.JellyfinID] = it
	}
	seriesOf := map[string]string{} // episode id -> show id (this run only)

	total := 0
	for {
		// the only SQL ever sent: fixed text plus one integer
		q := fmt.Sprintf("SELECT rowid, DateCreated, UserId, ItemId, ItemType, ItemName, PlayDuration FROM PlaybackActivity WHERE rowid > %d ORDER BY rowid LIMIT %d", last, batch)
		_, rows, err := jf.PlaybackQuery(ctx, q)
		if err != nil {
			return "", err
		}
		if len(rows) == 0 {
			break
		}
		if err := s.resolveEpisodes(ctx, jf, rows, seriesOf); err != nil {
			return "", err
		}
		events := make([]store.WatchEvent, 0, len(rows))
		for _, r := range rows {
			if len(r) < 7 {
				continue
			}
			rowid, _ := strconv.ParseInt(r[0], 10, 64)
			last = max(last, rowid)
			if ev, ok := toEvent(rowid, r, byJF, seriesOf); ok {
				events = append(events, ev)
			}
		}
		if err := s.store.InsertWatchEvents(ctx, events); err != nil {
			return "", err
		}
		total += len(events)
		if len(rows) < batch {
			break
		}
	}
	return fmt.Sprintf("%d new plays", total), nil
}

// resolveEpisodes asks Jellyfin which show each unknown episode belongs to.
func (s *Service) resolveEpisodes(ctx context.Context, jf *jellyfin.Client, rows [][]string, seriesOf map[string]string) error {
	var need []string
	seen := map[string]bool{}
	for _, r := range rows {
		if len(r) >= 7 && r[4] == "Episode" && r[3] != "" && seriesOf[r[3]] == "" && !seen[r[3]] {
			seen[r[3]] = true
			need = append(need, r[3])
		}
	}
	for i := 0; i < len(need); i += 100 {
		part := need[i:min(i+100, len(need))]
		err := jf.ItemsByID(ctx, part, func(it jellyfin.Item) error {
			if it.SeriesID != "" {
				seriesOf[it.ID] = it.SeriesID
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func toEvent(rowid int64, r []string, byJF map[string]store.LibraryItem, seriesOf map[string]string) (store.WatchEvent, bool) {
	itemID, itemType, name := r[3], r[4], r[5]
	played, ok := parseTime(r[1])
	if !ok {
		return store.WatchEvent{}, false
	}
	secs, _ := strconv.Atoi(r[6])
	ev := store.WatchEvent{SourceRowID: rowid, UserID: r[2], JellyfinID: itemID, PlayedAt: played.Unix(), Seconds: max(secs, 0)}
	switch itemType {
	case "Movie":
		ev.MediaType, ev.Title = "movie", name
		if it, ok := byJF[itemID]; ok && it.MediaType == "movie" {
			ev.TMDBID, ev.Title = it.TMDBID, it.Title
		}
	case "Episode":
		ev.MediaType = "tv"
		ev.Title = episodeSuffix.ReplaceAllString(name, "")
		if it, ok := byJF[seriesOf[itemID]]; ok && it.MediaType == "tv" {
			ev.TMDBID, ev.Title, ev.JellyfinID = it.TMDBID, it.Title, it.JellyfinID
		}
	default:
		return store.WatchEvent{}, false // music, books...
	}
	ev.Title = strings.TrimSpace(ev.Title)
	if len(ev.Title) > 255 {
		ev.Title = ev.Title[:255]
	}
	return ev, true
}

// parseTime reads the plugin's DateCreated, which is the Jellyfin server's local time (the
// plugin stores DateTime.Now); Tipsarr reads it in its own time zone (set TZ the same on both).
func parseTime(v string) (time.Time, bool) {
	v = strings.TrimSpace(v)
	for _, layout := range []string{"2006-01-02 15:04:05.9999999", "2006-01-02 15:04:05", "2006-01-02T15:04:05.9999999", "2006-01-02T15:04:05"} {
		if t, err := time.ParseInLocation(layout, v, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
