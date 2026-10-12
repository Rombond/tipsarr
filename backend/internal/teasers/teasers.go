// Package teasers builds the vertical trailer feed (Teasers tab). Every card plays a trailer file the server already
// holds: the server keeps a pool of about 100 trailers ready (75 movies, 25 shows, per language and region) and mixes
// them with the person's recommendations, two pool titles for one recommendation.
// It only displays titles; nothing is requested automatically.
package teasers

import (
	"context"
	"encoding/json"
	"log/slog"
	"math/rand/v2"
	"strconv"
	"sync"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/i18n"
	"github.com/Rombond/tipsarr/backend/internal/jobs"
	"github.com/Rombond/tipsarr/backend/internal/media"
	"github.com/Rombond/tipsarr/backend/internal/store"
	"github.com/Rombond/tipsarr/backend/internal/suggestions"
	"github.com/Rombond/tipsarr/backend/internal/trailers"
)

const (
	DefaultLimit = 8
	MaxLimit     = 20
	seenFor      = 60 * 24 * time.Hour
	maxLookups   = 40 // detail lookups (cached 24 h) for recommendations per call
	lookupBatch  = 4
	ensureBudget = 4 * time.Second // time the feed spends telling the resolver about missing files
	ensureMax    = 8               // missing files asked for per call

	// PoolMovies and PoolShows are how many titles with a trailer the server keeps ready for each language and region.
	PoolMovies  = 75
	PoolShows   = 25
	poolLookups = 120 // detail lookups per kind while building a pool
)

// Teaser is one card of the feed.
type Teaser struct {
	media.Item
	TrailerKey  string `json:"trailerKey" doc:"YouTube video key of the trailer, in the person's language when there is one"`
	Watchlisted bool   `json:"watchlisted"`
	VideoURL    string `json:"videoUrl" doc:"Trailer file served by this server (needs the same login as the API)"`
	VideoHeight int    `json:"videoHeight,omitempty"`
}

type Feed struct {
	Items []Teaser `json:"items"`
	Done  bool     `json:"done" doc:"True when there is nothing new left to show. No items and not done: trailers are still being prepared, ask again in a few seconds"`
}

type Service struct {
	store *store.Store
	media *media.Service
	sugg  *suggestions.Service
	files *trailers.Service

	// pick returns a number in [0, n); tests replace it.
	pick func(n int) int

	buildMu  sync.Mutex
	building map[string]bool
}

func New(st *store.Store, m *media.Service, s *suggestions.Service, files *trailers.Service) *Service {
	return &Service{store: st, media: m, sugg: s, files: files, pick: rand.IntN, building: map[string]bool{}}
}

// Enabled reports whether the Teasers feed is switched on (trailer files set up). Without them the apps hide the tab.
func (s *Service) Enabled(ctx context.Context) bool { return s.files.Enabled(ctx) }

// MarkSeen records that the person swiped past a title.
func (s *Service) MarkSeen(ctx context.Context, userID, mediaType string, tmdbID int) error {
	return s.store.MarkTeaserSeen(ctx, userID, mediaType, int64(tmdbID))
}

// HistoryItem is a title the person swiped past, with when.
type HistoryItem struct {
	media.Item
	SeenAt int64 `json:"seenAt" doc:"Unix time of the swipe"`
}

// History lists what a person (ownerID) swiped past, newest first, with the titles in the viewer's language.
// A title TMDB no longer answers for is left out of the page (the total still counts it).
func (s *Service) History(ctx context.Context, viewer *store.User, ownerID string, limit, offset int) ([]HistoryItem, int, error) {
	u := viewer
	rows, total, err := s.store.TeaserSeenPage(ctx, ownerID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	o := media.Opts{Language: u.Language, Region: u.Region}
	found := make([]*HistoryItem, len(rows))
	sem := make(chan struct{}, 6)
	var wg sync.WaitGroup
	for i, r := range rows {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			d, err := s.media.Detail(ctx, o, r.MediaType, int(r.TMDBID))
			if err != nil {
				return
			}
			found[i] = &HistoryItem{Item: d.Item, SeenAt: r.SeenAt}
		}()
	}
	wg.Wait()
	out := make([]HistoryItem, 0, len(rows))
	for _, h := range found {
		if h != nil {
			out = append(out, *h)
		}
	}
	return out, total, nil
}

// Forget makes one title eligible for the feed again.
func (s *Service) Forget(ctx context.Context, userID, mediaType string, tmdbID int) error {
	_, err := s.store.DeleteTeaserSeen(ctx, userID, mediaType, int64(tmdbID))
	return err
}

// ForgetAll clears the whole history of a person.
func (s *Service) ForgetAll(ctx context.Context, userID string) error {
	_, err := s.store.ClearTeaserSeen(ctx, userID)
	return err
}

// scopeOf is the pool a person reads from: one per language (fr, en...). Trailers depend on the language, not on the
// region, so people of one language share the same 100 files.
func scopeOf(language string) string { return media.LanguageCode(language) }

// cand is a title with the YouTube key of its trailer.
type cand struct {
	item media.Item
	key  string
}

// Next returns up to limit unseen titles whose trailer file is ready. exclude holds "type:id" keys the
// client already has but has not swiped past yet. Cards come in groups of three: two from the pool and one
// recommendation, in random order inside the group, each drawn at random from the top quarter of its list.
func (s *Service) Next(ctx context.Context, u *store.User, limit int, exclude map[string]bool) (*Feed, error) {
	if limit <= 0 {
		limit = DefaultLimit
	}
	limit = min(limit, MaxLimit)
	o := media.Opts{Language: u.Language, Region: u.Region}
	scope := scopeOf(u.Language)

	seen, err := s.store.TeaserSeenSet(ctx, u.ID, time.Now().Add(-seenFor).Unix())
	if err != nil {
		return nil, err
	}
	watched, err := s.store.WatchedSet(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	blocked, err := s.store.MarkSet(ctx, u.ID, store.MarkBlocklist)
	if err != nil {
		return nil, err
	}
	watchlist, err := s.store.MarkSet(ctx, u.ID, store.MarkWatchlist)
	if err != nil {
		return nil, err
	}
	skip := func(it media.Item) bool {
		k := store.WatchKey(it.Type, int64(it.TMDBID))
		return seen[k] || exclude[k] || watched[k] || blocked[k]
	}
	usable := func(it media.Item) bool {
		return it.Availability != media.AvailabilityAvailable && it.RequestStatus == ""
	}

	// pool titles
	rows, err := s.store.TeaserPool(ctx, scope)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		s.buildAsync(u.Language)
	}
	var poolItems []media.Item
	var poolKeys []string
	for _, r := range rows {
		var it media.Item
		if json.Unmarshal([]byte(r.Item), &it) != nil || skip(it) {
			continue
		}
		poolItems = append(poolItems, it)
		poolKeys = append(poolKeys, r.YoutubeKey)
	}
	s.media.Annotate(ctx, poolItems)
	var pool []cand
	for i, it := range poolItems {
		if usable(it) {
			pool = append(pool, cand{it, poolKeys[i]})
		}
	}

	// recommendations: the person's suggestion rows, with the trailer looked up in their language
	var recItems []media.Item
	taken := map[string]bool{}
	if res, err := s.sugg.Get(ctx, u); err == nil {
		for _, r := range res.Rows {
			for _, it := range r.Items {
				k := store.WatchKey(it.Type, int64(it.TMDBID))
				if taken[k] || skip(it) {
					continue
				}
				taken[k] = true
				recItems = append(recItems, it)
			}
		}
	}
	s.media.Annotate(ctx, recItems)
	var recQueue []media.Item
	for _, it := range recItems {
		if usable(it) {
			recQueue = append(recQueue, it)
		}
	}
	recs := s.withTrailers(ctx, o, recQueue, 1<<30, maxLookups)

	// split what is playable from what still downloads
	var readyPool, readyRecs []Teaser
	var missing []string
	pending := 0
	wl := func(it media.Item) bool { return watchlist[store.WatchKey(it.Type, int64(it.TMDBID))] }
	split := func(list []cand, into *[]Teaser) {
		for _, c := range list {
			if row, ok := s.files.Ready(ctx, c.key); ok {
				*into = append(*into, Teaser{Item: c.item, TrailerKey: c.key, Watchlisted: wl(c.item), VideoURL: "/api/v1/teasers/video/" + c.key, VideoHeight: row.Height})
			} else if s.files.Pending(ctx, c.key) {
				pending++
				missing = append(missing, c.key)
			}
		}
	}
	split(pool, &readyPool)
	split(recs, &readyRecs)

	if len(missing) > 0 {
		missing = missing[:min(len(missing), ensureMax)]
		ectx, cancel := context.WithTimeout(ctx, ensureBudget)
		if err := s.files.EnsureMany(ectx, missing); err != nil {
			slog.Warn("trailer files: cannot queue", "err", err)
		}
		cancel()
	}

	out := &Feed{Items: s.compose(readyPool, readyRecs, limit)}
	out.Done = len(out.Items) == 0 && pending == 0 && len(rows) > 0
	return out, nil
}

// compose builds the cards in groups of three (two from the pool, one recommendation, shuffled inside the group).
// A list that runs dry is replaced by the other one. A title appears once.
func (s *Service) compose(pool, recs []Teaser, limit int) []Teaser {
	out := []Teaser{}
	used := map[string]bool{}
	// draw takes a random title from the top quarter of the list
	draw := func(list *[]Teaser) (Teaser, bool) {
		for len(*list) > 0 {
			n := max(1, (len(*list)+3)/4)
			i := s.pick(n)
			t := (*list)[i]
			*list = append((*list)[:i], (*list)[i+1:]...)
			if k := store.WatchKey(t.Type, int64(t.TMDBID)); !used[k] {
				used[k] = true
				return t, true
			}
		}
		return Teaser{}, false
	}
	for len(out) < limit && (len(pool) > 0 || len(recs) > 0) {
		group := []bool{false, false, true} // true = recommendation
		for i := len(group) - 1; i > 0; i-- {
			j := s.pick(i + 1)
			group[i], group[j] = group[j], group[i]
		}
		for _, wantRec := range group {
			if len(out) >= limit {
				break
			}
			first, second := &pool, &recs
			if wantRec {
				first, second = &recs, &pool
			}
			if t, ok := draw(first); ok {
				out = append(out, t)
			} else if t, ok := draw(second); ok {
				out = append(out, t)
			}
		}
	}
	return out
}

// withTrailers looks titles up (details are cached 24 h) until want of them have a trailer, at most lookups of them.
func (s *Service) withTrailers(ctx context.Context, o media.Opts, queue []media.Item, want, lookups int) []cand {
	var got []cand
	done := 0
	for len(queue) > 0 && len(got) < want && done < lookups {
		n := min(lookupBatch, len(queue), lookups-done)
		batch := queue[:n]
		queue = queue[n:]
		done += n
		found := make([]*cand, n)
		var wg sync.WaitGroup
		for i, it := range batch {
			wg.Add(1)
			go func() {
				defer wg.Done()
				d, err := s.media.Detail(ctx, o, it.Type, it.TMDBID)
				if err != nil || d.TrailerKey == "" {
					return
				}
				// the list item may lack fields the detail has; the detail wins
				if d.BackdropPath != "" {
					it.BackdropPath = d.BackdropPath
				}
				if d.Overview != "" {
					it.Overview = d.Overview
				}
				found[i] = &cand{it, d.TrailerKey}
			}()
		}
		wg.Wait()
		for _, c := range found {
			if c != nil && len(got) < want {
				got = append(got, *c)
			}
		}
	}
	return got
}

// ---- the pool ---------------------------------------------------------------------------------------------

// RefreshPool is the background job: for every language and region the users have, it rebuilds the pool
// (about 75 movies and 25 shows with a trailer), asks the resolver for every file that is missing and keeps the
// files warm so the resolver's clean-up deletes other ones first.
func (s *Service) RefreshPool(ctx context.Context) (i18n.Msg, error) {
	if !s.files.Enabled(ctx) {
		return i18n.M("srv.job.skip.teasers_off"), jobs.ErrSkipped
	}
	users, err := s.store.ListUsers(ctx)
	if err != nil {
		return i18n.Msg{}, err
	}
	seen := map[string]bool{}
	var scopes []string
	titles := 0
	var firstErr error
	for _, u := range users {
		scope := scopeOf(u.Language)
		if seen[scope] {
			continue
		}
		seen[scope] = true
		scopes = append(scopes, scope)
		n, err := s.refreshScope(ctx, u.Language)
		titles += n
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	// pools nobody reads any more stop being refreshed and kept warm; their files age out of the cache
	if err := s.store.DeleteTeaserPoolExcept(ctx, scopes); err != nil && firstErr == nil {
		firstErr = err
	}
	return i18n.M("srv.job.teaserpool.done", "scopes", len(scopes), "titles", titles), firstErr
}

// buildAsync builds the pool of a scope in the background (a person opened Teasers before the job ran).
func (s *Service) buildAsync(language string) {
	scope := scopeOf(language)
	s.buildMu.Lock()
	if s.building[scope] {
		s.buildMu.Unlock()
		return
	}
	s.building[scope] = true
	s.buildMu.Unlock()
	go func() {
		defer func() {
			s.buildMu.Lock()
			delete(s.building, scope)
			s.buildMu.Unlock()
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if _, err := s.refreshScope(ctx, language); err != nil {
			slog.Warn("teaser pool: build failed", "scope", scope, "err", err)
		}
	}()
}

func (s *Service) refreshScope(ctx context.Context, language string) (int, error) {
	o := media.Opts{Language: language}
	scope := scopeOf(language)
	movies, shows := s.poolCandidates(ctx, o)
	pickedMovies := s.withTrailers(ctx, o, movies, PoolMovies, poolLookups)
	pickedShows := s.withTrailers(ctx, o, shows, PoolShows, poolLookups)

	// three movies, then one show, and so on: a quarter of any list holds both kinds
	var all []cand
	for mi, si := 0, 0; mi < len(pickedMovies) || si < len(pickedShows); {
		for range 3 {
			if mi < len(pickedMovies) {
				all = append(all, pickedMovies[mi])
				mi++
			}
		}
		if si < len(pickedShows) {
			all = append(all, pickedShows[si])
			si++
		}
	}
	if len(all) == 0 {
		return 0, nil // TMDB gave nothing: keep the old pool
	}
	now := time.Now().Unix()
	rows := make([]store.TeaserPoolRow, 0, len(all))
	keys := make([]string, 0, len(all))
	for i, c := range all {
		raw, err := json.Marshal(c.item)
		if err != nil {
			continue
		}
		rows = append(rows, store.TeaserPoolRow{Scope: scope, MediaType: c.item.Type, TMDBID: int64(c.item.TMDBID), YoutubeKey: c.key, SortOrder: i, Item: string(raw), UpdatedAt: now})
		keys = append(keys, c.key)
	}
	if err := s.store.ReplaceTeaserPool(ctx, scope, rows); err != nil {
		return 0, err
	}
	ectx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if err := s.files.EnsureMany(ectx, keys); err != nil {
		return len(rows), err
	}
	_ = s.files.KeepWarm(ectx, keys)
	return len(rows), nil
}

// poolCandidates lists movies (trending and upcoming, one after the other, then popular ones to fill up) and shows
// (trending, then popular), without the titles the library already has.
func (s *Service) poolCandidates(ctx context.Context, o media.Opts) (movies, shows []media.Item) {
	var trendMovies, trendShows, upcoming []media.Item
	for page := 1; page <= 5; page++ {
		if l, err := s.media.Trending(ctx, o, page); err == nil {
			for _, it := range l.Items {
				if it.Type == "tv" {
					trendShows = append(trendShows, it)
				} else {
					trendMovies = append(trendMovies, it)
				}
			}
		}
	}
	for page := 1; page <= 3; page++ {
		if l, err := s.media.Upcoming(ctx, o, page); err == nil {
			upcoming = append(upcoming, l.Items...)
		}
	}
	var popMovies, popShows []media.Item
	for page := 1; page <= 3; page++ {
		if l, err := s.media.Discover(ctx, o, "movie", 0, 0, page); err == nil {
			popMovies = append(popMovies, l.Items...)
		}
		if l, err := s.media.Discover(ctx, o, "tv", 0, 0, page); err == nil {
			popShows = append(popShows, l.Items...)
		}
	}
	movies = unique(alternate(trendMovies, upcoming), popMovies)
	shows = unique(trendShows, popShows)
	s.media.Annotate(ctx, movies)
	s.media.Annotate(ctx, shows)
	return notInLibrary(movies), notInLibrary(shows)
}

func notInLibrary(items []media.Item) []media.Item {
	out := items[:0:0]
	for _, it := range items {
		if it.Availability != media.AvailabilityAvailable {
			out = append(out, it)
		}
	}
	return out
}

// alternate takes one title from each list in turn.
func alternate(a, b []media.Item) []media.Item {
	out := make([]media.Item, 0, len(a)+len(b))
	for i := 0; i < len(a) || i < len(b); i++ {
		if i < len(a) {
			out = append(out, a[i])
		}
		if i < len(b) {
			out = append(out, b[i])
		}
	}
	return out
}

// unique joins lists in order and keeps a title once.
func unique(lists ...[]media.Item) []media.Item {
	var out []media.Item
	seen := map[string]bool{}
	for _, l := range lists {
		for _, it := range l {
			k := it.Type + ":" + strconv.Itoa(it.TMDBID)
			if !seen[k] {
				seen[k] = true
				out = append(out, it)
			}
		}
	}
	return out
}
