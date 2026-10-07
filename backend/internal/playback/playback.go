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
	"unicode"

	"github.com/Rombond/tipsarr/backend/internal/clients/jellyfin"
	"github.com/Rombond/tipsarr/backend/internal/media"
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

// TMDB is the part of the metadata service used to recognise titles that left the library.
type TMDB interface {
	SearchTitle(ctx context.Context, mediaType, language, title string) ([]media.Item, error)
	Detail(ctx context.Context, o media.Opts, mediaType string, id int) (*media.Detail, error)
}

type Service struct {
	store       *store.Store
	newJellyfin func(baseURL string) *jellyfin.Client
	tmdb        TMDB // optional
}

func New(s *store.Store, tmdb TMDB) *Service {
	return &Service{store: s, newJellyfin: jellyfin.New, tmdb: tmdb}
}

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
	relinked := s.relink(ctx, jf, byJF)
	resolved := s.resolveTitles(ctx)
	if relinked > 0 {
		return fmt.Sprintf("%d new plays, %d titles linked to the library, %d removed titles recognised", total, relinked, resolved), nil
	}
	if resolved > 0 {
		return fmt.Sprintf("%d new plays, %d removed titles recognised", total, resolved), nil
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

const (
	resolvePerRun = 150
	retryAfter    = 7 * 24 * time.Hour
)

// resolveTitles looks up on TMDB the titles that were played but are no longer in the Jellyfin
// library (clean-up tools remove them), so their plays can still be shown with a poster, year and
// genres. A title is searched once; "no match" is retried after a week. It returns how many were found.
func (s *Service) resolveTitles(ctx context.Context) int {
	if s.tmdb == nil {
		return 0
	}
	pending, err := s.store.UnknownPlayedTitles(ctx, time.Now().Add(-retryAfter).Unix(), resolvePerRun)
	if err != nil || len(pending) == 0 {
		return 0
	}
	appLang, _ := s.store.GetSetting(ctx, media.SettingDefaultLanguage)
	found := 0
	for _, p := range pending {
		typ, name := p[0], p[1]
		w := &store.WatchTitle{MediaType: typ, Title: name, CheckedAt: time.Now().Unix()}
		hit, err := s.match(ctx, typ, name, appLang)
		if errors.Is(err, media.ErrNotConfigured) {
			return found
		}
		if err != nil {
			continue // TMDB trouble: ask again on the next run
		}
		if hit != nil {
			w.TMDBID = int64(hit.TMDBID)
			w.Poster, w.Rating10 = hit.PosterPath, int(hit.VoteAverage*10+0.5)
			if len(hit.ReleaseDate) >= 4 {
				w.Year, _ = strconv.Atoi(hit.ReleaseDate[:4])
			}
			if d, err := s.tmdb.Detail(ctx, media.Opts{}, typ, hit.TMDBID); err == nil {
				var g []string
				for _, x := range d.Genres {
					g = append(g, strings.ReplaceAll(x.Name, "|", " "))
				}
				if len(g) > 0 {
					w.Genres = "|" + strings.Join(g, "|") + "|"
				}
			}
			found++
		}
		_ = s.store.SaveWatchTitle(ctx, w)
	}
	return found
}

// match finds the TMDB title for a Jellyfin title, in English first and then in the app's language
// (Jellyfin names follow its own metadata language). Only a result whose title says the same counts.
func (s *Service) match(ctx context.Context, typ, name, appLang string) (*media.Item, error) {
	// Jellyfin names follow its metadata language, so the app's language goes first: in English
	// "Là-haut" is the title of another, older film, while in French it is "Up".
	langs := []string{""}
	if appLang != "" && !strings.HasPrefix(appLang, "en") {
		langs = []string{appLang, ""}
	}
	want := normalise(name)
	for _, lang := range langs {
		items, err := s.tmdb.SearchTitle(ctx, typ, lang, name)
		if err != nil {
			return nil, err
		}
		for i := range items {
			// the same title, and a picture: an entry TMDB has no poster for is rarely the film that was played
			if got := normalise(items[i].Title); got != "" && got == want && items[i].PosterPath != "" {
				return &items[i], nil
			}
		}
	}
	return nil, nil
}

// normalise lower-cases a title and keeps only its letters and digits ("Pacific Rim : Uprising" = "Pacific Rim: Uprising").
func normalise(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// relink links plays that were imported before their title was in the library (a play is imported
// once, but the library changes: a title added, or a collection that used to be hidden) to the
// library title now. Episodes are mapped to their show through Jellyfin. It returns how many items were linked.
func (s *Service) relink(ctx context.Context, jf *jellyfin.Client, byJF map[string]store.LibraryItem) int {
	items, err := s.store.UnlinkedPlayedItems(ctx)
	if err != nil || len(items) == 0 {
		return 0
	}
	seriesOf := map[string]string{}
	var episodes [][]string
	for _, it := range items {
		if it[0] == "tv" {
			episodes = append(episodes, []string{"", "", "", it[1], "Episode", "", ""})
		}
	}
	if len(episodes) > 0 {
		if err := s.resolveEpisodes(ctx, jf, episodes, seriesOf); err != nil {
			return 0
		}
	}
	n := 0
	for _, it := range items {
		typ, id := it[0], it[1]
		lookup := id
		if typ == "tv" {
			lookup = seriesOf[id]
		}
		lib, ok := byJF[lookup]
		if !ok || lib.MediaType != typ {
			continue
		}
		if err := s.store.LinkWatchEvents(ctx, typ, id, lib.TMDBID, lib.Title, lib.JellyfinID); err == nil {
			n++
		}
	}
	return n
}
