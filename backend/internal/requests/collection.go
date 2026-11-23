package requests

import (
	"context"
	"errors"
	"fmt"

	"github.com/Rombond/tipsarr/backend/internal/media"
	"github.com/Rombond/tipsarr/backend/internal/store"
)

// ErrNotInCollection means a picked title is not a part of the collection.
var ErrNotInCollection = errors.New("this title is not part of the collection")

// PartOutcome is what happened to one part of a collection: a request (View) or the reason it was skipped (Err).
type PartOutcome struct {
	Item media.Item
	View *View
	Err  error
}

// Missing reports whether a collection part can still be requested: not in the library, no active
// request, and already released (TMDB lists announced films without a date or with a future one).
func Missing(it media.Item, today string) bool {
	return it.Availability != media.AvailabilityAvailable && it.RequestStatus == "" && it.ReleaseDate != "" && it.ReleaseDate <= today
}

// RequestCollection requests parts of a TMDB movie collection, one normal request per part: the same
// rights, quota, 4K and approval rules as a single request (nothing bypasses Create). tmdbIDs picks the
// parts; empty means every released part that is neither in the library nor requested. Once a part fails
// for a reason that holds for all of them (quota, rights, no Radarr), the rest are skipped with it.
func (s *Service) RequestCollection(ctx context.Context, u *store.User, o media.Opts, id int, tmdbIDs []int, ov *Overrides) ([]PartOutcome, error) {
	if !u.Mask().MayRequest("movie") {
		return nil, ErrForbidden
	}
	c, err := s.media.Collection(ctx, o, id)
	if err != nil {
		return nil, err
	}
	var picked []media.Item
	foreign := map[int]bool{} // picked ids that are not parts of the collection
	if len(tmdbIDs) == 0 {
		today := s.now().UTC().Format("2006-01-02")
		for _, p := range c.Parts {
			if Missing(p, today) {
				picked = append(picked, p)
			}
		}
	} else {
		parts := make(map[int]media.Item, len(c.Parts))
		for _, p := range c.Parts {
			parts[p.TMDBID] = p
		}
		seen := map[int]bool{}
		for _, tid := range tmdbIDs {
			if seen[tid] {
				continue
			}
			seen[tid] = true
			p, ok := parts[tid]
			if !ok {
				p, foreign[tid] = media.Item{Type: "movie", TMDBID: tid}, true
			}
			picked = append(picked, p)
		}
	}

	out := make([]PartOutcome, 0, len(picked))
	var stop error
	for _, p := range picked {
		res := PartOutcome{Item: p}
		switch {
		case foreign[p.TMDBID]:
			res.Err = fmt.Errorf("%w: %d", ErrNotInCollection, p.TMDBID)
		case stop != nil:
			res.Err = stop
		default:
			res.View, res.Err = s.Create(ctx, u, CreateParams{Type: "movie", TMDBID: p.TMDBID, Overrides: ov})
			if errors.Is(res.Err, ErrQuota) || errors.Is(res.Err, ErrForbidden) || errors.Is(res.Err, ErrNo4K) || errors.Is(res.Err, ErrNoInstance) {
				stop = res.Err
			}
		}
		out = append(out, res)
	}
	return out, nil
}
