package requests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/clients/servarr"
	"github.com/Rombond/tipsarr/backend/internal/media"
	"github.com/Rombond/tipsarr/backend/internal/store"
)

// How long the season source waits for Sonarr and how long it keeps what it learned.
const (
	seriesTimeout   = 4 * time.Second // tracked series (fast in Sonarr)
	lookupTimeout   = 6 * time.Second // metadata lookup of an untracked show
	lookupTTL       = 24 * time.Hour  // untracked show: its season list rarely moves
	trackedTTL      = 24 * time.Hour  // tracked show: only the offline fallback reads it (live first)
	firstAiredTTL   = 6 * time.Hour   // first air date of each Sonarr season
	firstAiredQuick = 3 * time.Second // budget of the episode list when nothing is cached
)

// ErrSeasonNumbering: a request made in TMDB numbering cannot be translated to Sonarr's seasons.
var ErrSeasonNumbering = errors.New("TMDB and Sonarr number this show's seasons differently: ask for every season, or pick the season in Sonarr")

// ErrNumberingChanged: the person picked seasons in a numbering the server no longer uses (a stale page).
var ErrNumberingChanged = errors.New("the seasons of this show changed: reload and pick them again")

// SeasonSource is the TVDB season list read from the default Sonarr instance; wire it with media.Service.SetSeasonSource.
func (s *Service) SeasonSource() media.SeasonSource { return sonarrSeasons{s} }

type sonarrSeasons struct{ s *Service }

type cachedSeasons struct {
	Seasons  []media.TVDBSeason `json:"seasons"`
	InSonarr bool               `json:"inSonarr"`
}

func (src sonarrSeasons) TVDBSeasons(ctx context.Context, tvdbID int) ([]media.TVDBSeason, bool, bool) {
	s := src.s
	if tvdbID <= 0 {
		return nil, false, false
	}
	inst, err := s.store.DefaultServarr(ctx, servarr.KindSonarr)
	if err != nil {
		return nil, false, false
	}
	c := s.newServarr(inst)
	key := fmt.Sprintf("sonarr:seasons:%s:%d", inst.ID, tvdbID)

	sctx, cancel := context.WithTimeout(ctx, seriesTimeout)
	sr, err := c.SeriesByTVDB(sctx, tvdbID)
	cancel()
	if err != nil {
		return s.staleSeasons(ctx, key, err)
	}
	if sr != nil {
		out := make([]media.TVDBSeason, 0, len(sr.Seasons))
		for _, sn := range sr.Seasons {
			out = append(out, media.TVDBSeason{Number: sn.SeasonNumber, EpisodeCount: sn.Statistics.TotalEpisodeCount, Monitored: sr.Monitored && sn.Monitored})
		}
		first := s.firstAired(ctx, c, inst.ID, sr.ID)
		for i := range out {
			out[i].FirstAired = first[out[i].Number]
		}
		sortTVDB(out)
		s.putSeasons(ctx, key, cachedSeasons{Seasons: out, InSonarr: true}, trackedTTL)
		return out, true, len(out) > 0
	}

	// Sonarr does not track it: its metadata lookup still knows the TVDB season list (without sizes).
	if body, ok, _ := s.store.GetCache(ctx, key); ok {
		var cs cachedSeasons
		if json.Unmarshal(body, &cs) == nil && !cs.InSonarr && len(cs.Seasons) > 0 {
			return cs.Seasons, false, true
		}
	}
	lctx, cancel := context.WithTimeout(ctx, lookupTimeout)
	lk, err := c.LookupSeries(lctx, tvdbID)
	cancel()
	if err != nil {
		return s.staleSeasons(ctx, key, err)
	}
	if lk == nil || len(lk.Seasons) == 0 {
		return nil, false, false
	}
	out := make([]media.TVDBSeason, 0, len(lk.Seasons))
	for _, sn := range lk.Seasons {
		out = append(out, media.TVDBSeason{Number: sn.SeasonNumber, EpisodeCount: sn.Statistics.TotalEpisodeCount})
	}
	sortTVDB(out)
	s.putSeasons(ctx, key, cachedSeasons{Seasons: out}, lookupTTL)
	return out, false, true
}

// staleSeasons is the fallback while Sonarr cannot be reached: the last list it gave, however old.
func (s *Service) staleSeasons(ctx context.Context, key string, cause error) ([]media.TVDBSeason, bool, bool) {
	slog.Debug("sonarr seasons", "key", key, "err", cause)
	body, ok, err := s.store.GetCacheStale(ctx, key)
	if err != nil || !ok {
		return nil, false, false
	}
	var cs cachedSeasons
	if json.Unmarshal(body, &cs) != nil || len(cs.Seasons) == 0 {
		return nil, false, false
	}
	return cs.Seasons, cs.InSonarr, true
}

func (s *Service) putSeasons(ctx context.Context, key string, cs cachedSeasons, ttl time.Duration) {
	if body, err := json.Marshal(cs); err == nil {
		_ = s.store.PutCache(ctx, key, body, ttl)
	}
}

// firstAired gives the first air date of each season of a tracked series (from its episode list, cached).
// Best effort: an empty map only means no TMDB season can be matched by date.
func (s *Service) firstAired(ctx context.Context, c *servarr.Client, instID string, seriesID int) map[int]string {
	key := fmt.Sprintf("sonarr:firstaired:%s:%d", instID, seriesID)
	out := map[int]string{}
	if body, ok, _ := s.store.GetCache(ctx, key); ok && json.Unmarshal(body, &out) == nil {
		return out
	}
	ectx, cancel := context.WithTimeout(ctx, firstAiredQuick)
	eps, err := c.SeriesEpisodes(ectx, seriesID)
	cancel()
	if err != nil {
		if body, ok, _ := s.store.GetCacheStale(ctx, key); ok {
			_ = json.Unmarshal(body, &out)
		}
		return out
	}
	out = map[int]string{}
	for _, e := range eps {
		if e.AirDate == "" {
			continue
		}
		if cur, ok := out[e.SeasonNumber]; !ok || e.AirDate < cur {
			out[e.SeasonNumber] = e.AirDate
		}
	}
	if body, err := json.Marshal(out); err == nil {
		_ = s.store.PutCache(ctx, key, body, firstAiredTTL)
	}
	return out
}

func sortTVDB(list []media.TVDBSeason) {
	sort.Slice(list, func(i, j int) bool { return list[i].Number < list[j].Number })
}

// ---- season detail ----------------------------------------------------------------------------

// SeasonDetail lists the episodes of season n of a show, n in the numbering of the show's Detail (TVDB when
// Sonarr can tell). It takes TMDB's episodes when a TMDB season matches, Sonarr's otherwise, and marks what
// Sonarr already has. A season nobody can list yet comes back with no episodes.
func (s *Service) SeasonDetail(ctx context.Context, o media.Opts, tmdbID, n int) (*media.SeasonDetail, error) {
	o.Seasons = true
	d, err := s.media.Detail(ctx, o, "tv", tmdbID)
	if err != nil {
		return nil, err
	}
	if d.SeasonNumbering != media.NumberingTVDB {
		sd, err := s.media.Season(ctx, o, tmdbID, n)
		if err != nil {
			return nil, err
		}
		if !d.SeasonNumberingDiffers {
			s.markEpisodes(ctx, d.TVDBID, sd)
		}
		return sd, nil
	}
	var sn *media.Season
	for i := range d.Seasons {
		if d.Seasons[i].Number == n {
			sn = &d.Seasons[i]
		}
	}
	if sn == nil {
		return nil, media.ErrNotFound
	}
	if sn.TMDBNumber > 0 || (n == 0 && !d.SeasonNumberingDiffers) {
		tn := sn.TMDBNumber
		if sd, err := s.media.Season(ctx, o, tmdbID, tn); err == nil {
			sd.Number = n
			s.markEpisodes(ctx, d.TVDBID, sd)
			return sd, nil
		} else if !errors.Is(err, media.ErrNotFound) {
			return nil, err
		}
	}
	sd := &media.SeasonDetail{Number: n, Name: sn.Name, AirDate: sn.AirDate, PosterPath: sn.PosterPath, Episodes: []media.Episode{}}
	sr, c := s.liveSeries(ctx, d.TVDBID)
	if sr == nil {
		return sd, nil
	}
	ectx, cancel := context.WithTimeout(ctx, seriesTimeout)
	eps, err := c.Episodes(ectx, sr.ID, n)
	cancel()
	if err != nil {
		return sd, nil
	}
	sort.Slice(eps, func(i, j int) bool { return eps[i].EpisodeNumber < eps[j].EpisodeNumber })
	for _, e := range eps {
		if e.SeasonNumber != n {
			continue
		}
		sd.Episodes = append(sd.Episodes, media.Episode{Number: e.EpisodeNumber, Name: e.Title, Overview: e.Overview, AirDate: e.AirDate, Runtime: e.Runtime, State: episodeState(e, true, sr.Monitored)})
	}
	return sd, nil
}

// liveSeries reads the series Sonarr tracks for a TVDB id; nil when there is none or Sonarr cannot tell.
func (s *Service) liveSeries(ctx context.Context, tvdbID int) (*servarr.Series, *servarr.Client) {
	c := s.instance(ctx, servarr.KindSonarr)
	if c == nil || tvdbID == 0 {
		return nil, nil
	}
	sctx, cancel := context.WithTimeout(ctx, seriesTimeout)
	defer cancel()
	sr, err := c.SeriesByTVDB(sctx, tvdbID)
	if err != nil || sr == nil {
		return nil, nil
	}
	return sr, c
}

// markEpisodes marks each episode of a (Sonarr-numbered) season as available, monitored or missing.
func (s *Service) markEpisodes(ctx context.Context, tvdbID int, sd *media.SeasonDetail) {
	sr, c := s.liveSeries(ctx, tvdbID)
	if sr == nil {
		return
	}
	ectx, cancel := context.WithTimeout(ctx, seriesTimeout)
	eps, err := c.Episodes(ectx, sr.ID, sd.Number)
	cancel()
	if err != nil {
		return
	}
	by := map[int]servarr.Episode{}
	for _, e := range eps {
		by[e.EpisodeNumber] = e
	}
	for i := range sd.Episodes {
		e, ok := by[sd.Episodes[i].Number]
		sd.Episodes[i].State = episodeState(e, ok, sr.Monitored)
	}
}

// episodeState is what Sonarr says about one episode; empty when Sonarr does not know it (not aired in its database).
func episodeState(e servarr.Episode, known, seriesMonitored bool) string {
	switch {
	case !known:
		return ""
	case e.HasFile:
		return "available"
	case e.Monitored && seriesMonitored:
		return "monitored"
	default:
		return "missing"
	}
}

// ---- numbering guard --------------------------------------------------------------------------

// sonarrSeasonsFor turns the seasons of a request into Sonarr's numbering. 'tvdb' rows pass as they are; 'tmdb'
// rows (old ones, or made while Sonarr was unreachable) pass when both numberings agree, or become every Sonarr
// season when they asked for every TMDB season. Anything else is ErrSeasonNumbering: never send wrong seasons.
func (s *Service) sonarrSeasonsFor(ctx context.Context, r *store.Request, requested []int) ([]int, error) {
	if r.SeasonNumbering != media.NumberingTMDB || len(requested) == 0 {
		return requested, nil
	}
	d, err := s.media.Detail(ctx, media.Opts{Seasons: true}, "tv", int(r.TMDBID))
	if err != nil {
		return nil, err
	}
	if d.SeasonNumbering != media.NumberingTVDB {
		return requested, nil // Sonarr still cannot tell: same guess the request was made with
	}
	if !d.SeasonNumberingDiffers {
		return requested, nil
	}
	tmdb, err := s.media.Detail(ctx, media.Opts{}, "tv", int(r.TMDBID)) // without the source: TMDB's own seasons
	if err != nil {
		return nil, err
	}
	want := map[int]bool{}
	for _, n := range requested {
		want[n] = true
	}
	for _, sn := range tmdb.Seasons {
		if sn.Number > 0 && sn.EpisodeCount > 0 && !want[sn.Number] {
			return nil, ErrSeasonNumbering
		}
	}
	var all []int
	for _, sn := range d.Seasons {
		if sn.Number > 0 {
			all = append(all, sn.Number)
		}
	}
	if len(all) == 0 {
		return nil, ErrSeasonNumbering
	}
	return all, nil
}
