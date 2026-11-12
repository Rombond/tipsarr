package requests

import (
	"context"

	"github.com/Rombond/tipsarr/backend/internal/clients/servarr"
	"github.com/Rombond/tipsarr/backend/internal/media"
)

// instance returns a client for the default instance of a kind, or nil when there is none or it cannot be read.
func (s *Service) instance(ctx context.Context, kind string) *servarr.Client {
	inst, err := s.store.DefaultServarr(ctx, kind)
	if err != nil {
		return nil
	}
	return s.newServarr(inst)
}

// Annotate tells what Radarr/Sonarr already do about a title: a movie Radarr monitors without a file, the seasons
// Sonarr monitors. It only reads, and any failure leaves the detail as it is (the Jellyfin counts still apply).
func (s *Service) Annotate(ctx context.Context, d *media.Detail) {
	switch d.Type {
	case "movie":
		c := s.instance(ctx, servarr.KindRadarr)
		if c == nil {
			return
		}
		if m, err := c.MovieByTMDB(ctx, d.TMDBID); err == nil && m != nil {
			d.Monitored = m.Monitored && !m.HasFile
		}
	case "tv":
		// Season monitoring comes with the seasons themselves: media.Detail asks Sonarr (Opts.Seasons) and
		// numbers them like Sonarr does. TMDB-numbered seasons (Sonarr unreachable) cannot be matched by number.
	}
}
