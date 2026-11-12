package media

import (
	"context"
	"sort"
	"strconv"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/i18n"
)

// Season numbering of a Detail: TV seasons follow TVDB (through Sonarr) whenever Sonarr can tell, TMDB otherwise.
const (
	NumberingTVDB = "tvdb"
	NumberingTMDB = "tmdb"
)

// TVDBSeason is one season as Sonarr (TVDB) numbers it.
type TVDBSeason struct {
	Number       int
	EpisodeCount int // 0 = unknown (Sonarr lookup of a show it does not track)
	FirstAired   string
	Monitored    bool
}

// SeasonSource gives the TVDB season list of a show (implemented over Sonarr, injected at startup).
type SeasonSource interface {
	// ok=false => Sonarr unavailable/not configured/unknown show: caller keeps TMDB seasons.
	TVDBSeasons(ctx context.Context, tvdbID int) (seasons []TVDBSeason, inSonarr bool, ok bool)
}

// SetSeasonSource wires the TVDB season source; nil keeps TMDB seasons.
func (s *Service) SetSeasonSource(src SeasonSource) { s.seasons = src }

// NumberingDiffers says whether TMDB and TVDB split the show's regular seasons differently. A single
// extra season past the other side's last one (a newly announced season) is not a difference.
func NumberingDiffers(tmdb []Season, tvdb []TVDBSeason) bool {
	tm := map[int]int{}
	for _, s := range tmdb {
		if s.Number > 0 && s.EpisodeCount > 0 {
			tm[s.Number] = s.EpisodeCount
		}
	}
	tv := map[int]int{}
	for _, s := range tvdb {
		if s.Number > 0 {
			tv[s.Number] = s.EpisodeCount
		}
	}
	if len(tm) == 0 || len(tv) == 0 {
		return false
	}
	if extraDiffers(tm, tv) || extraDiffers(tv, tm) {
		return true
	}
	for n, c := range tm {
		v, ok := tv[n]
		if !ok || v == 0 {
			continue
		}
		d := c - v
		if d < 0 {
			d = -d
		}
		if d > max(2, c/10) {
			return true
		}
	}
	return false
}

// extraDiffers: a has seasons b lacks, more than one, or one that is not past b's highest season.
func extraDiffers(a, b map[int]int) bool {
	hi := 0
	for n := range b {
		hi = max(hi, n)
	}
	var extra []int
	for n := range a {
		if _, ok := b[n]; !ok {
			extra = append(extra, n)
		}
	}
	return len(extra) > 1 || (len(extra) == 1 && extra[0] <= hi)
}

// MergeSeasons returns the TVDB seasons enriched with their TMDB counterpart (name, poster, air date,
// count when TVDB does not know it). name gives the label of a season without counterpart; never empty.
func MergeSeasons(tmdb []Season, tvdb []TVDBSeason, name func(n int) string) (out []Season, differs bool) {
	differs = NumberingDiffers(tmdb, tvdb)
	byNum := map[int]Season{}
	for _, s := range tmdb {
		byNum[s.Number] = s
	}
	sorted := append([]TVDBSeason(nil), tvdb...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Number < sorted[j].Number })
	out = make([]Season, 0, len(sorted))
	for _, tv := range sorted {
		var cp *Season
		if t, ok := byNum[tv.Number]; ok && !differs {
			cp = &t
		} else if tv.EpisodeCount > 0 && tv.FirstAired != "" {
			for _, t := range tmdb {
				if t.Number > 0 && t.AirDate == tv.FirstAired && t.EpisodeCount == tv.EpisodeCount {
					t := t
					cp = &t
					break
				}
			}
		}
		sn := Season{Number: tv.Number, EpisodeCount: tv.EpisodeCount, Monitored: tv.Monitored}
		if cp != nil {
			sn.TMDBNumber, sn.Name, sn.PosterPath, sn.AirDate = cp.Number, cp.Name, cp.PosterPath, cp.AirDate
			if sn.EpisodeCount == 0 {
				sn.EpisodeCount = cp.EpisodeCount
			}
		} else {
			sn.AirDate = tv.FirstAired
		}
		if sn.Name == "" {
			sn.Name = name(tv.Number)
		}
		sn.CountUnknown = sn.EpisodeCount == 0
		out = append(out, sn)
	}
	return out, differs
}

// SeasonName is the localized label of a season without a TMDB name ("Season 3", "Specials").
func SeasonName(lang string) func(n int) string {
	return func(n int) string {
		code, params := "srv.season", map[string]any{"n": n}
		if n == 0 {
			code, params = "srv.season.specials", nil
		}
		if t, ok := i18n.T(lang, code, params); ok && t != "" {
			return t
		}
		if n == 0 {
			return "Specials"
		}
		return "Season " + strconv.Itoa(n)
	}
}

// applySeasonSource switches a TV detail to TVDB numbering when the source can tell.
func (s *Service) applySeasonSource(ctx context.Context, o Opts, d *Detail) {
	d.SeasonNumbering = NumberingTMDB
	if s.seasons == nil || d.TVDBID == 0 {
		return
	}
	tvdb, _, ok := s.seasons.TVDBSeasons(ctx, d.TVDBID)
	if !ok || len(tvdb) == 0 {
		return
	}
	d.Seasons, d.SeasonNumberingDiffers = MergeSeasons(d.Seasons, tvdb, SeasonName(o.lang()))
	d.SeasonNumbering = NumberingTVDB
	d.NumberOfSeasons = 0
	for _, sn := range d.Seasons {
		if sn.Number > 0 {
			d.NumberOfSeasons++
		}
	}
}

// ApplyLibrary (re)computes library availability of a TV detail from its seasons; safe to call again
// after the seasons changed. A season of unknown size counts as complete once the library holds any of it.
func (s *Service) ApplyLibrary(ctx context.Context, d *Detail) {
	have, err := s.store.LibraryIDs(ctx, "tv", []int{d.TMDBID})
	if err != nil || !have[d.TMDBID] {
		return
	}
	lib, err := s.store.LibrarySeasons(ctx, d.TMDBID)
	if err != nil {
		return
	}
	d.Availability = libraryAvailability(d.Seasons, lib)
}

func libraryAvailability(seasons []Season, lib map[int]int) string {
	complete, anyEpisode := true, false
	for i, sn := range seasons {
		seasons[i].LibraryEpisodes = lib[sn.Number]
		if sn.Number == 0 || (sn.EpisodeCount == 0 && !sn.CountUnknown) {
			continue // specials and announced-but-empty seasons don't count
		}
		if lib[sn.Number] > 0 {
			anyEpisode = true
		}
		if sn.CountUnknown {
			if lib[sn.Number] == 0 && aired(sn.AirDate) {
				complete = false // a season not aired yet (or undated) does not make the show partial
			}
		} else if lib[sn.Number] < sn.EpisodeCount {
			complete = false
		}
	}
	if complete && anyEpisode {
		return AvailabilityAvailable
	}
	return AvailabilityPartial
}

func aired(date string) bool {
	return date != "" && date <= time.Now().UTC().Format("2006-01-02")
}
