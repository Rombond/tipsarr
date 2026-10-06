// Package marks implements per-user watchlist and blocklist. The blocklist hides a title
// from that user's suggestions; the watchlist is a personal "to look at later" list.
package marks

import (
	"context"
	"errors"

	"github.com/Rombond/tipsarr/backend/internal/media"
	"github.com/Rombond/tipsarr/backend/internal/store"
)

var ErrInvalid = errors.New("invalid kind or media type")

type Service struct {
	store *store.Store
	media *media.Service
}

func New(s *store.Store, m *media.Service) *Service { return &Service{store: s, media: m} }

func valid(kind, mediaType string) bool {
	return (kind == store.MarkWatchlist || kind == store.MarkBlocklist) && (mediaType == "movie" || mediaType == "tv")
}

// Add puts a title on a list (idempotent). The title is looked up on TMDB for a snapshot.
func (s *Service) Add(ctx context.Context, u *store.User, kind, mediaType string, tmdbID int) error {
	if !valid(kind, mediaType) || tmdbID <= 0 {
		return ErrInvalid
	}
	d, err := s.media.Detail(ctx, media.Opts{Language: u.Language, Region: u.Region}, mediaType, tmdbID)
	if err != nil {
		return err
	}
	return s.store.AddMark(ctx, &store.UserMark{
		UserID: u.ID, Kind: kind, MediaType: mediaType, TMDBID: int64(tmdbID), Title: d.Title,
		PosterPath: d.PosterPath, ReleaseDate: d.ReleaseDate, VoteTenths: int(d.VoteAverage * 10),
	})
}

func (s *Service) Remove(ctx context.Context, u *store.User, kind, mediaType string, tmdbID int) error {
	if !valid(kind, mediaType) {
		return ErrInvalid
	}
	return s.store.RemoveMark(ctx, u.ID, kind, mediaType, int64(tmdbID))
}

// List returns the user's titles of one kind, newest first, with availability applied.
func (s *Service) List(ctx context.Context, u *store.User, kind string) ([]media.Item, error) {
	if !valid(kind, "movie") {
		return nil, ErrInvalid
	}
	rows, err := s.store.Marks(ctx, u.ID, kind)
	if err != nil {
		return nil, err
	}
	items := make([]media.Item, 0, len(rows))
	for _, r := range rows {
		items = append(items, media.Item{
			Type: r.MediaType, TMDBID: int(r.TMDBID), Title: r.Title, PosterPath: r.PosterPath, ReleaseDate: r.ReleaseDate,
			VoteAverage: float64(r.VoteTenths) / 10, Availability: media.AvailabilityNone,
		})
	}
	s.media.Annotate(ctx, items)
	return items, nil
}

type Flags struct {
	Watchlisted bool `json:"watchlisted"`
	Blocklisted bool `json:"blocklisted"`
}

func (s *Service) Flags(ctx context.Context, u *store.User, mediaType string, tmdbID int) (Flags, error) {
	w, b, err := s.store.HasMark(ctx, u.ID, mediaType, int64(tmdbID))
	return Flags{Watchlisted: w, Blocklisted: b}, err
}
