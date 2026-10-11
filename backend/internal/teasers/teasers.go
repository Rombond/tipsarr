// Package teasers builds the vertical trailer feed (Teasers tab): titles the person has not seen
// yet, from their suggestions, trending and upcoming, each with a trailer in their language.
// It only displays titles; nothing is requested automatically.
package teasers

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/media"
	"github.com/Rombond/tipsarr/backend/internal/store"
	"github.com/Rombond/tipsarr/backend/internal/suggestions"
)

const (
	DefaultLimit = 8
	MaxLimit     = 20
	seenFor      = 60 * 24 * time.Hour
	maxLookups   = 40 // detail lookups (cached 24 h) per call
	lookupBatch  = 4
)

// Teaser is one card of the feed.
type Teaser struct {
	media.Item
	TrailerKey  string `json:"trailerKey" doc:"YouTube video key of the trailer, in the person's language when there is one"`
	Watchlisted bool   `json:"watchlisted"`
}

type Feed struct {
	Items []Teaser `json:"items"`
	Done  bool     `json:"done" doc:"True when there is nothing new left to show"`
}

type Service struct {
	store *store.Store
	media *media.Service
	sugg  *suggestions.Service
}

func New(st *store.Store, m *media.Service, s *suggestions.Service) *Service {
	return &Service{store: st, media: m, sugg: s}
}

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

	out := &Feed{Items: []Teaser{}}
	lookups := 0
	for len(queue) > 0 && len(out.Items) < limit && lookups < maxLookups {
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
				found[i] = &Teaser{Item: it, TrailerKey: d.TrailerKey, Watchlisted: watchlist[store.WatchKey(it.Type, int64(it.TMDBID))]}
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
			if t != nil && len(out.Items) < limit {
				out.Items = append(out.Items, *t)
			}
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
