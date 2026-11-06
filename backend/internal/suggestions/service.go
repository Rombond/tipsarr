// Package suggestions builds the Netflix-style rows shown on Discover, per Jellyfin user:
//
//   - one account row ("Recommended for you"), generated the first time it is needed and
//     recomputed only when that user's watch history changed;
//   - one "Because you watched X" row per recent watch.
//
// A user with no history is seeded from what the whole server watches, then from trending.
// Suggestions only ever DISPLAY titles; nothing is requested automatically.
package suggestions

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/Rombond/tipsarr/backend/internal/events"
	"github.com/Rombond/tipsarr/backend/internal/media"
	"github.com/Rombond/tipsarr/backend/internal/store"
)

const (
	maxBecauseRows     = 6
	itemsPerRow        = 20
	accountItems       = 40
	accountSeeds       = 15
	minRowItems        = 4
	staleNoHistory     = 7 * 24 * time.Hour
	refreshCooldown    = time.Minute
	backgroundCooldown = 30 * time.Second
	queueDelay         = 5 * time.Second
	fetchConcurrency   = 6
	KindAccount        = "account"
	KindBecause        = "because"
	labelPersonal      = "Recommended for you"
	labelServer        = "Popular on this server"
	labelTrending      = "Trending now"
)

var ErrTooSoon = errors.New("recommendations were refreshed a moment ago")

type Service struct {
	store *store.Store
	media *media.Service
	hub   *events.Hub // optional

	mu             sync.Mutex
	userLocks      map[string]*sync.Mutex
	lastForced     map[string]time.Time
	lastBackground map[string]time.Time
	timers         map[string]*time.Timer
	delay          time.Duration
}

func New(s *store.Store, m *media.Service, h *events.Hub) *Service {
	return &Service{store: s, media: m, hub: h, userLocks: map[string]*sync.Mutex{}, lastForced: map[string]time.Time{}, lastBackground: map[string]time.Time{}, timers: map[string]*time.Timer{}, delay: queueDelay}
}

// SetQueueDelay changes the debounce used by QueueRefresh (tests).
func (s *Service) SetQueueDelay(d time.Duration) { s.delay = d }

type Seed struct {
	Type   string `json:"type" enum:"movie,tv"`
	TMDBID int    `json:"tmdbId"`
	Title  string `json:"title"`
}

type Row struct {
	ID          string       `json:"id"`
	Kind        string       `json:"kind" enum:"account,because"`
	Title       string       `json:"title" doc:"English display title (clients should translate from variant + seed)"`
	Variant     string       `json:"variant" enum:"personal,server,trending,because" doc:"What the row is based on, so clients can show it in the user's language"`
	Seed        *Seed        `json:"seed,omitempty"`
	Personal    bool         `json:"personal" doc:"Built from this user's own watch history"`
	GeneratedAt int64        `json:"generatedAt"`
	Items       []media.Item `json:"items"`
}

type Result struct {
	Rows       []Row `json:"rows"`
	Generating bool  `json:"generating" doc:"A refresh is running in the background; a suggestions.updated event follows"`
}

func (s *Service) userLock(id string) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()
	l := s.userLocks[id]
	if l == nil {
		l = &sync.Mutex{}
		s.userLocks[id] = l
	}
	return l
}

// ---- reading ----------------------------------------------------------------------------

// historyScope is whose watch history a person's rows come from: their own, plus the accounts an admin mapped to
// them (one level, no chains). Get and Generate must both read it, so they agree on the version.
type historyScope struct {
	ids        []string // the person first, then the sources
	version    int64    // staleness token of the whole scope
	hasHistory bool
	borrowed   bool // there is at least one source
}

func (s *Service) scopeOf(ctx context.Context, userID string) (historyScope, error) {
	sources, err := s.store.HistorySources(ctx, userID)
	if err != nil {
		return historyScope{}, err
	}
	sc := historyScope{ids: []string{userID}}
	for _, id := range sources {
		if id != userID {
			sc.ids = append(sc.ids, id)
		}
	}
	sc.borrowed = len(sc.ids) > 1
	own, err := s.store.HistoryVersion(ctx, userID)
	if err != nil {
		return sc, err
	}
	if !sc.borrowed {
		sc.version = own // exactly what it was before sources existed, so no row goes stale
		sc.hasHistory, err = s.store.HasHistory(ctx, userID)
		return sc, err
	}
	// the source ids are part of the hash, so adding, changing or removing a source changes the version too
	h := fnv.New64a()
	fmt.Fprintf(h, "%s:%d", userID, own)
	rest := append([]string{}, sc.ids[1:]...)
	sort.Strings(rest)
	for _, id := range rest {
		v, err := s.store.HistoryVersion(ctx, id)
		if err != nil {
			return sc, err
		}
		fmt.Fprintf(h, "|%s:%d", id, v)
	}
	sc.version = int64(h.Sum64() >> 1) // keep it positive: 0 is "never synced"
	if sc.version == 0 {
		sc.version = 1
	}
	sc.hasHistory, err = s.store.HasHistoryOf(ctx, sc.ids)
	return sc, err
}

// Get returns the user's rows. The first call generates them (synchronously); later calls
// serve what is stored and refresh in the background when the history changed.
func (s *Service) Get(ctx context.Context, u *store.User) (*Result, error) {
	rows, items, err := s.store.SuggestionRows(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		if err := s.Generate(ctx, u.ID); err != nil {
			return nil, err
		}
		if rows, items, err = s.store.SuggestionRows(ctx, u.ID); err != nil {
			return nil, err
		}
	}
	res := &Result{Rows: []Row{}}
	sc, err := s.scopeOf(ctx, u.ID)
	if err != nil {
		// Discover must keep working as it did before sources existed: read the person's own history, ignoring errors
		slog.Warn("suggestions: history scope", "user", u.ID, "err", err)
		sc = historyScope{ids: []string{u.ID}}
		sc.version, _ = s.store.HistoryVersion(ctx, u.ID)
		sc.hasHistory, _ = s.store.HasHistory(ctx, u.ID)
	}
	if stale(rows[0:1], sc.version, sc.hasHistory) {
		res.Generating = true
		if s.claimBackground(u.ID) {
			go s.refreshInBackground(u.ID)
		}
	}

	watched, err := s.hiddenSet(ctx, sc, u.ID)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		row := Row{ID: r.ID, Kind: r.Kind, Personal: r.Personal == 1, GeneratedAt: r.GeneratedAt, Items: []media.Item{}}
		switch {
		case r.Kind == KindBecause:
			row.Variant = "because"
		case r.SeedTitle == labelServer:
			row.Variant = "server"
		case r.SeedTitle == labelTrending:
			row.Variant = "trending"
		default:
			row.Variant = "personal"
		}
		if r.Kind == KindBecause {
			row.Title = "Because you watched " + r.SeedTitle
			row.Seed = &Seed{Type: r.SeedType, TMDBID: int(r.SeedTMDBID), Title: r.SeedTitle}
		} else {
			row.Title = r.SeedTitle
		}
		for _, it := range items[r.ID] {
			if watched[store.WatchKey(it.MediaType, it.TMDBID)] {
				continue // watched since the row was built
			}
			row.Items = append(row.Items, itemFromSnapshot(it))
		}
		if len(row.Items) < minRowItems {
			continue
		}
		s.media.Annotate(ctx, row.Items)
		res.Rows = append(res.Rows, row)
	}
	return res, nil
}

func stale(rows []store.SuggestionRow, currentVersion int64, hasHistory bool) bool {
	if len(rows) == 0 {
		return true
	}
	r := rows[0]
	if r.Personal == 1 {
		return r.HistoryVersion != currentVersion
	}
	// built from server-wide history or trending: refresh when the user gains history of their
	// own (a synced but EMPTY history does not count), or weekly
	return hasHistory || time.Since(time.Unix(r.GeneratedAt, 0)) > staleNoHistory
}

// hiddenSet is everything that must not appear in a user's suggestions: titles they or their source accounts
// watched, and titles they put on their own blocklist (another account's blocklist is never borrowed).
func (s *Service) hiddenSet(ctx context.Context, sc historyScope, userID string) (map[string]bool, error) {
	watched, err := s.store.WatchedSetOf(ctx, sc.ids)
	if err != nil {
		return nil, err
	}
	blocked, err := s.store.MarkSet(ctx, userID, store.MarkBlocklist)
	if err != nil {
		return nil, err
	}
	for k := range blocked {
		watched[k] = true
	}
	return watched, nil
}

func itemFromSnapshot(it store.SuggestionItem) media.Item {
	return media.Item{
		Type: it.MediaType, TMDBID: int(it.TMDBID), Title: it.Title, PosterPath: it.PosterPath, ReleaseDate: it.ReleaseDate,
		VoteAverage: float64(it.VoteTenths) / 10, Overview: it.Overview, Availability: media.AvailabilityNone,
	}
}

func snapshot(rowID string, pos int, it media.Item) store.SuggestionItem {
	ov := shorten(it.Overview, 400)
	return store.SuggestionItem{
		RowID: rowID, Pos: pos, MediaType: it.Type, TMDBID: int64(it.TMDBID), Title: it.Title, PosterPath: it.PosterPath,
		ReleaseDate: it.ReleaseDate, VoteTenths: int(it.VoteAverage * 10), Overview: ov,
	}
}

// ---- refreshing --------------------------------------------------------------------------

// claimBackground allows one background refresh per user every 30 seconds, so a refresh that
// cannot fix the staleness (TMDB down, nothing to recommend) never turns into a loop with
// clients that reload when the refresh finishes.
func (s *Service) claimBackground(userID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, ok := s.lastBackground[userID]; ok && time.Since(t) < backgroundCooldown {
		return false
	}
	s.lastBackground[userID] = time.Now()
	return true
}

// Force starts a background regeneration of a user's rows, at most once per minute; a
// suggestions.updated event follows.
func (s *Service) Force(ctx context.Context, userID string) error {
	s.mu.Lock()
	if t, ok := s.lastForced[userID]; ok && time.Since(t) < refreshCooldown {
		s.mu.Unlock()
		return ErrTooSoon
	}
	s.lastForced[userID] = time.Now()
	s.mu.Unlock()
	go s.refreshInBackground(userID)
	return nil
}

// QueueRefresh schedules a debounced background regeneration (used when history changed), for the person and for
// everyone whose recommendations borrow from them. The version check already refreshes those rows when they are next
// read; this only makes it sooner.
func (s *Service) QueueRefresh(userID string) {
	s.queue(userID)
	deps, err := s.store.UsersBorrowingFrom(context.Background(), userID)
	if err != nil {
		slog.Warn("suggestions: who borrows history", "user", userID, "err", err)
		return
	}
	for _, id := range deps {
		if id != userID {
			s.queue(id) // not QueueRefresh: two accounts mapped to each other would loop
		}
	}
}

func (s *Service) queue(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, ok := s.timers[userID]; ok {
		t.Stop()
	}
	s.timers[userID] = time.AfterFunc(s.delay, func() {
		s.mu.Lock()
		delete(s.timers, userID)
		s.mu.Unlock()
		s.refreshInBackground(userID)
	})
}

func (s *Service) refreshInBackground(userID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if err := s.Generate(ctx, userID); err != nil {
		slog.Warn("suggestions refresh failed", "user", userID, "err", err)
	}
}

type related struct {
	recs, similar []media.Item
	err           error
}

// Generate rebuilds all rows of one user from their watch history. Concurrent calls for the
// same user are serialised.
func (s *Service) Generate(ctx context.Context, userID string) error {
	lock := s.userLock(userID)
	lock.Lock()
	defer lock.Unlock()

	opts := media.Opts{}
	if u, err := s.store.GetUser(ctx, userID); err == nil {
		opts = media.Opts{Language: u.Language, Region: u.Region}
		if opts.Language == "" { // no saved language: the app-wide default
			opts.Language, _ = s.store.GetSetting(ctx, media.SettingDefaultLanguage)
		}
	}
	sc, err := s.scopeOf(ctx, userID)
	if err != nil {
		return err
	}
	var hist []store.WatchHistory
	if sc.borrowed {
		hist, err = s.store.MergedHistory(ctx, sc.ids) // one row per title, newest first: seeds and "because" rows share it
	} else {
		hist, err = s.store.UserHistory(ctx, userID, 0)
	}
	if err != nil {
		return err
	}
	version := sc.version
	watched, err := s.hiddenSet(ctx, sc, userID) // watched (by them and their sources) + their own blocklist
	if err != nil {
		return err
	}

	personal := len(hist) > 0
	seeds := hist
	label := labelPersonal
	if !personal {
		if seeds, err = s.store.GlobalHistorySeeds(ctx, accountSeeds); err != nil {
			return err
		}
		label = labelServer
	}
	if len(seeds) > accountSeeds {
		seeds = seeds[:accountSeeds]
	}

	rel := s.fetchRelated(ctx, opts, seeds)
	for _, r := range rel {
		if errors.Is(r.err, media.ErrNotConfigured) {
			return media.ErrNotConfigured
		}
	}

	now := time.Now().Unix()
	var rows []store.SuggestionRow
	var items []store.SuggestionItem
	add := func(kind, label, seedType string, seedID int64, isPersonal bool, list []media.Item) {
		id := store.NewID()
		rows = append(rows, store.SuggestionRow{
			ID: id, UserID: userID, Kind: kind, SeedType: seedType, SeedTMDBID: seedID, SeedTitle: label,
			Position: len(rows), GeneratedAt: now, HistoryVersion: version, Personal: boolInt(isPersonal),
		})
		for i, it := range list {
			items = append(items, snapshot(id, i, it))
		}
	}

	// account row
	account := rankCandidates(seeds, rel, watched)
	if len(seeds) == 0 || len(account) < minRowItems {
		label = labelTrending
		trending, err := s.media.TrendingItems(ctx, opts, 1)
		if err != nil {
			return err
		}
		account = filterItems(trending, watched, nil)
	}
	if len(account) > accountItems {
		account = account[:accountItems]
	}
	add(KindAccount, label, "", 0, personal, account)

	// "because you watched" rows: only from the user's own history
	if personal {
		for i := 0; i < len(hist) && i < maxBecauseRows && i < len(rel); i++ {
			if rel[i].err != nil {
				continue
			}
			list := filterItems(append(append([]media.Item{}, rel[i].recs...), rel[i].similar...), watched, map[string]bool{})
			if len(list) < minRowItems {
				continue
			}
			if len(list) > itemsPerRow {
				list = list[:itemsPerRow]
			}
			add(KindBecause, s.seedTitle(ctx, opts, hist[i]), hist[i].MediaType, hist[i].TMDBID, true, list)
		}
	}

	if err := s.store.ReplaceSuggestions(ctx, userID, rows, items); err != nil {
		return err
	}
	if s.hub != nil {
		s.hub.Publish("suggestions.updated", userID, map[string]any{"rows": len(rows)}, false)
	}
	return nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// seedTitle prefers the Jellyfin title, then TMDB.
func (s *Service) seedTitle(ctx context.Context, o media.Opts, h store.WatchHistory) string {
	if t := s.store.LibraryTitle(ctx, h.MediaType, h.TMDBID); t != "" {
		return t
	}
	if d, err := s.media.Detail(ctx, o, h.MediaType, int(h.TMDBID)); err == nil {
		return d.Title
	}
	return "a title you watched"
}

func (s *Service) fetchRelated(ctx context.Context, o media.Opts, seeds []store.WatchHistory) []related {
	out := make([]related, len(seeds))
	sem := make(chan struct{}, fetchConcurrency)
	var wg sync.WaitGroup
	for i, sd := range seeds {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			r, sm, err := s.media.Related(ctx, o, sd.MediaType, int(sd.TMDBID))
			out[i] = related{recs: r, similar: sm, err: err}
		}()
	}
	wg.Wait()
	return out
}

// rankCandidates scores titles recommended for the seeds: earlier seeds weigh more, and a
// title recommended by several seeds climbs. Watched titles and titles without a poster are dropped.
func rankCandidates(seeds []store.WatchHistory, rel []related, watched map[string]bool) []media.Item {
	type cand struct {
		item  media.Item
		score float64
	}
	cands := map[string]*cand{}
	bump := func(it media.Item, pts float64) {
		k := store.WatchKey(it.Type, int64(it.TMDBID))
		c := cands[k]
		if c == nil {
			c = &cand{item: it, score: it.VoteAverage / 100}
			cands[k] = c
		}
		c.score += pts
	}
	for i := range seeds {
		if i >= len(rel) || rel[i].err != nil {
			continue
		}
		w := 1.0 / (1 + 0.15*float64(i))
		for pos, it := range rel[i].recs {
			bump(it, w*(1-float64(pos)/25))
		}
		for pos, it := range rel[i].similar {
			bump(it, 0.5*w*(1-float64(pos)/25))
		}
	}
	list := make([]*cand, 0, len(cands))
	for k, c := range cands {
		if watched[k] || c.item.PosterPath == "" {
			continue
		}
		list = append(list, c)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].score != list[j].score {
			return list[i].score > list[j].score
		}
		return list[i].item.TMDBID < list[j].item.TMDBID
	})
	out := make([]media.Item, len(list))
	for i, c := range list {
		out[i] = c.item
	}
	return out
}

// filterItems drops watched titles, titles without a poster and duplicates (seen may be nil).
func filterItems(in []media.Item, watched map[string]bool, seen map[string]bool) []media.Item {
	if seen == nil {
		seen = map[string]bool{}
	}
	var out []media.Item
	for _, it := range in {
		k := store.WatchKey(it.Type, int64(it.TMDBID))
		if watched[k] || seen[k] || it.PosterPath == "" {
			continue
		}
		seen[k] = true
		out = append(out, it)
	}
	return out
}

// shorten cuts text at a word boundary (never mid-character) and adds an ellipsis.
func shorten(s string, n int) string {
	if len(s) <= n {
		return s
	}
	cut := s[:n]
	if i := strings.LastIndexAny(cut, " \t\n"); i > n/2 {
		cut = cut[:i]
	}
	for len(cut) > 0 && !utf8.ValidString(cut) {
		cut = cut[:len(cut)-1]
	}
	return strings.TrimRight(cut, " ,.;:-") + "…"
}
