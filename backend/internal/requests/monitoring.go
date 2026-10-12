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
		sr := s.series(ctx, d.TVDBID)
		if sr == nil || !sr.Monitored {
			return
		}
		monitored := map[int]bool{}
		for _, sn := range sr.Seasons {
			monitored[sn.SeasonNumber] = sn.Monitored
		}
		for i := range d.Seasons {
			d.Seasons[i].Monitored = monitored[d.Seasons[i].Number]
		}
	}
}

func (s *Service) series(ctx context.Context, tvdbID int) *servarr.Series {
	c := s.instance(ctx, servarr.KindSonarr)
	if c == nil || tvdbID == 0 {
		return nil
	}
	sr, err := c.SeriesByTVDB(ctx, tvdbID)
	if err != nil {
		return nil
	}
	return sr
}

// AnnotateEpisodes marks each episode of a season as available, monitored (Sonarr is looking for it) or missing.
func (s *Service) AnnotateEpisodes(ctx context.Context, o media.Opts, tmdbID int, sd *media.SeasonDetail) {
	d, err := s.media.Detail(ctx, o, "tv", tmdbID)
	if err != nil {
		return
	}
	sr := s.series(ctx, d.TVDBID)
	if sr == nil {
		return
	}
	c := s.instance(ctx, servarr.KindSonarr)
	eps, err := c.Episodes(ctx, sr.ID, sd.Number)
	if err != nil {
		return
	}
	by := map[int]servarr.Episode{}
	for _, e := range eps {
		by[e.EpisodeNumber] = e
	}
	for i := range sd.Episodes {
		e, ok := by[sd.Episodes[i].Number]
		switch {
		case !ok:
			// Sonarr does not know the episode (not aired in its database): leave the state empty
		case e.HasFile:
			sd.Episodes[i].State = "available"
		case e.Monitored && sr.Monitored:
			sd.Episodes[i].State = "monitored"
		default:
			sd.Episodes[i].State = "missing"
		}
	}
}
