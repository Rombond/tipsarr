// Package teasers builds the vertical trailer feed (Teasers tab): titles the person has not seen
// yet, from their suggestions, trending and upcoming, each with a trailer in their language.
// It only displays titles; nothing is requested automatically.
package teasers

import (
	"context"
	"log/slog"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/media"
	"github.com/Rombond/tipsarr/backend/internal/store"
	"github.com/Rombond/tipsarr/backend/internal/suggestions"
	"github.com/Rombond/tipsarr/backend/internal/trailers"
)

const (
	DefaultLimit = 8
	MaxLimit     = 20
	seenFor      = 60 * 24 * time.Hour
	maxLookups   = 40 // detail lookups (cached 24 h) per call
	lookupBatch  = 4
	aheadBatches = 2               // with trailer files on: how many batches beyond the one returned are queued for download
	ensureBudget = 4 * time.Second // time the feed spends telling the resolver about those keys
)

// Teaser is one card of the feed.
type Teaser struct {
	media.Item
	TrailerKey  string `json:"trailerKey" doc:"YouTube video key of the trailer, in the person's language when there is one"`
	Watchlisted bool   `json:"watchlisted"`
	// Player says how to play the card: "file" when videoUrl is ready, else the YouTube player with trailerKey.
	Player      string `json:"player" enum:"file,youtube"`
	VideoURL    string `json:"videoUrl,omitempty" doc:"Trailer file served by this server (needs the same login as the API); set when player is file"`
	VideoHeight int    `json:"videoHeight,omitempty"`
}

type Feed struct {
	Items []Teaser `json:"items"`
	Done  bool     `json:"done" doc:"True when there is nothing new left to show"`
}

type Service struct {
	store *store.Store
	media *media.Service
	sugg  *suggestions.Service
	files *trailers.Service // optional: trailer files from the resolver sidecar
}

func New(st *store.Store, m *media.Service, s *suggestions.Service, files *trailers.Service) *Service {
	return &Service{store: st, media: m, sugg: s, files: files}
}

// Enabled reports whether the Teasers feed is switched on (trailer files set up). Without them the apps hide the tab.
func (s *Service) Enabled(ctx context.Context) bool { return s.files.Enabled(ctx) }

// MarkSeen records that the person swiped past a title.
func (s *Service) MarkSeen(ctx context.Context, userID, mediaType string, tmdbID int) error {
	return s.store.MarkTeaserSeen(ctx, userID, mediaType, int64(tmdbID))
}

// Next returns up to limit unseen titles that have a trailer. exclude holds "type:id" keys the
// client already has but has not swiped past yet.
func (s *Service) Next(ctx context.Context, u *store.User, limit int, exclude map[string]bool) (*Feed, error) {
	if limit <= 0 {
		limit = DefaultLimit
	}
	limit = min(limit, MaxLimit)
	o := media.Opts{Language: u.Language, Region: u.Region}

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

	cands := s.candidates(ctx, u, o)
	s.media.Annotate(ctx, cands)
	var queue []media.Item
	for _, it := range cands {
		key := store.WatchKey(it.Type, int64(it.TMDBID))
		switch {
		case seen[key], exclude[key], watched[key], blocked[key]:
		case it.Availability == media.AvailabilityAvailable, it.RequestStatus != "":
		default:
			queue = append(queue, it)
		}
	}

	useFiles := s.files.Enabled(ctx)
	lookups := 0
	// collect takes titles from the queue until it holds want of them that have a trailer
	collect := func(want int) []Teaser {
		var got []Teaser
		for len(queue) > 0 && len(got) < want && lookups < maxLookups {
			n := min(lookupBatch, len(queue), maxLookups-lookups)
			batch := queue[:n]
			queue = queue[n:]
			lookups += n
			found := make([]*Teaser, n)
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
					found[i] = &Teaser{Item: it, TrailerKey: d.TrailerKey, Player: "youtube", Watchlisted: watchlist[store.WatchKey(it.Type, int64(it.TMDBID))]}
					if d.BackdropPath != "" {
						found[i].BackdropPath = d.BackdropPath
					}
					if d.Overview != "" {
						found[i].Overview = d.Overview
					}
				}()
			}
			wg.Wait()
			for _, t := range found {
				if t != nil && len(got) < want {
					got = append(got, *t)
				}
			}
		}
		return got
	}

	out := &Feed{Items: collect(limit)}
	if out.Items == nil {
		out.Items = []Teaser{}
	}
	if useFiles {
		// ask the resolver for these cards and the next two batches, so the files are ready by the time they are shown
		ahead := collect(aheadBatches * limit)
		keys := make([]string, 0, len(out.Items)+len(ahead))
		for _, t := range out.Items {
			keys = append(keys, t.TrailerKey)
		}
		for _, t := range ahead {
			keys = append(keys, t.TrailerKey)
		}
		ectx, cancel := context.WithTimeout(ctx, ensureBudget)
		if err := s.files.EnsureMany(ectx, keys); err != nil {
			slog.Warn("trailer files: cannot queue", "err", err)
		}
		cancel()
		ready := 0
		for i := range out.Items {
			if row, ok := s.files.Ready(ctx, out.Items[i].TrailerKey); ok {
				out.Items[i].Player = "file"
				out.Items[i].VideoURL = "/api/v1/teasers/video/" + out.Items[i].TrailerKey
				out.Items[i].VideoHeight = row.Height
				ready++
			}
		}
		if ready > 0 { // cards that start at once come first
			sort.SliceStable(out.Items, func(a, b int) bool { return out.Items[a].Player == "file" && out.Items[b].Player != "file" })
		}
	}
	out.Done = len(out.Items) == 0 && len(queue) == 0
	return out, nil
}

// candidates mixes the person's suggestions (twice as often) with trending and upcoming titles.
func (s *Service) candidates(ctx context.Context, u *store.User, o media.Opts) []media.Item {
	var personal, trending, upcoming []media.Item
	if res, err := s.sugg.Get(ctx, u); err == nil {
		for _, r := range res.Rows {
			personal = append(personal, r.Items...)
		}
	}
	for page := 1; page <= 3; page++ {
		if l, err := s.media.Trending(ctx, o, page); err == nil {
			trending = append(trending, l.Items...)
		}
	}
	for page := 1; page <= 2; page++ {
		if l, err := s.media.Upcoming(ctx, o, page); err == nil {
			upcoming = append(upcoming, l.Items...)
		}
	}
	return interleave(personal, trending, upcoming)
}

// interleave takes two personal titles, then one trending, then one upcoming, and so on until all
// lists are empty; a title that appears in several lists is kept once.
func interleave(personal, trending, upcoming []media.Item) []media.Item {
	var out []media.Item
	seen := map[string]bool{}
	take := func(l *[]media.Item) {
		for len(*l) > 0 {
			it := (*l)[0]
			*l = (*l)[1:]
			k := it.Type + ":" + strconv.Itoa(it.TMDBID)
			if seen[k] {
				continue
			}
			seen[k] = true
			out = append(out, it)
			return
		}
	}
	for len(personal)+len(trending)+len(upcoming) > 0 {
		take(&personal)
		take(&personal)
		take(&trending)
		take(&upcoming)
	}
	return out
}
