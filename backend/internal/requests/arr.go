package requests

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/Rombond/tipsarr/backend/internal/clients/servarr"
	"github.com/Rombond/tipsarr/backend/internal/media"
)

// ErrNotInServarr is returned when Radarr/Sonarr does not track the title.
var ErrNotInServarr = errors.New("Radarr/Sonarr does not track this title")

// ArrLink is where a title lives in Radarr/Sonarr, as an address the person's browser can open.
type ArrLink struct {
	Kind string `json:"kind" enum:"radarr,sonarr"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// arrTarget finds the instance and the id Radarr (movie id) or Sonarr (series id) gives the title.
func (s *Service) arrTarget(ctx context.Context, o media.Opts, typ string, tmdbID int) (kind, name, base, slug string, id int, c *servarr.Client, err error) {
	kind = servarr.KindRadarr
	if typ == "tv" {
		kind = servarr.KindSonarr
	}
	inst, ierr := s.store.DefaultServarr(ctx, kind)
	if ierr != nil {
		return "", "", "", "", 0, nil, ErrNoInstance
	}
	c = s.newServarr(inst)
	name, base = inst.Name, strings.TrimRight(inst.URL, "/")
	if inst.PublicURL != "" {
		base = strings.TrimRight(inst.PublicURL, "/")
	}
	if typ == "movie" {
		m, merr := c.MovieByTMDB(ctx, tmdbID)
		if merr != nil {
			return "", "", "", "", 0, nil, merr
		}
		if m == nil {
			return "", "", "", "", 0, nil, ErrNotInServarr
		}
		return kind, name, base, strconv.Itoa(tmdbID), m.ID, c, nil
	}
	d, derr := s.media.Detail(ctx, media.Opts{Language: o.Language, Region: o.Region, Seasons: true}, "tv", tmdbID)
	if derr != nil {
		return "", "", "", "", 0, nil, derr
	}
	if d.TVDBID == 0 {
		return "", "", "", "", 0, nil, ErrNotInServarr
	}
	sr, serr := c.SeriesByTVDB(ctx, d.TVDBID)
	if serr != nil {
		return "", "", "", "", 0, nil, serr
	}
	if sr == nil {
		return "", "", "", "", 0, nil, ErrNotInServarr
	}
	return kind, name, base, sr.TitleSlug, sr.ID, c, nil
}

// ArrLink returns the address of the title's page in Radarr/Sonarr. It only reads.
func (s *Service) ArrLink(ctx context.Context, o media.Opts, typ string, tmdbID int) (*ArrLink, error) {
	kind, name, base, slug, _, _, err := s.arrTarget(ctx, o, typ, tmdbID)
	if err != nil {
		return nil, err
	}
	path := "/movie/"
	if kind == servarr.KindSonarr {
		path = "/series/"
	}
	return &ArrLink{Kind: kind, Name: name, URL: base + path + slug}, nil
}

// ArrSearch starts an automatic search in Radarr/Sonarr for the title (season/episode 0 = everything).
// In dry-run mode the write is blocked and servarr.ErrDryRun is returned.
func (s *Service) ArrSearch(ctx context.Context, o media.Opts, typ string, tmdbID, season, episode int) error {
	_, _, _, _, id, c, err := s.arrTarget(ctx, o, typ, tmdbID)
	if err != nil {
		return err
	}
	if typ == "movie" {
		return c.SearchMovie(ctx, id)
	}
	return c.SearchSeries(ctx, id, season, episode)
}
