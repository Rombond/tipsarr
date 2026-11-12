package servarr

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

type Series struct {
	ID         int    `json:"id"`
	Title      string `json:"title"`
	TVDBID     int    `json:"tvdbId"`
	TMDBID     int    `json:"tmdbId"`
	TitleSlug  string `json:"titleSlug"`
	Monitored  bool   `json:"monitored"`
	Year       int    `json:"year"`
	Statistics struct {
		EpisodeFileCount int `json:"episodeFileCount"`
		EpisodeCount     int `json:"episodeCount"`
	} `json:"statistics"`
	Seasons []SeriesSeason `json:"seasons"`
}

// SeriesSeason is one season of a Sonarr series (TVDB numbering). Lookups of untracked shows carry no statistics.
type SeriesSeason struct {
	SeasonNumber int  `json:"seasonNumber"`
	Monitored    bool `json:"monitored"`
	Statistics   struct {
		EpisodeFileCount  int `json:"episodeFileCount"`
		EpisodeCount      int `json:"episodeCount"`
		TotalEpisodeCount int `json:"totalEpisodeCount"` // whole season, monitored or not (episodeCount is 0 when unmonitored)
	} `json:"statistics"`
}

// MissingSeasons lists the monitored seasons that still lack episode files.
func (s Series) MissingSeasons() []int {
	var out []int
	for _, sn := range s.Seasons {
		if sn.SeasonNumber > 0 && sn.Monitored && sn.Statistics.EpisodeCount > 0 && sn.Statistics.EpisodeFileCount < sn.Statistics.EpisodeCount {
			out = append(out, sn.SeasonNumber)
		}
	}
	return out
}

// Episode is the part of a Sonarr episode the app needs.
type Episode struct {
	ID                    int    `json:"id"`
	SeasonNumber          int    `json:"seasonNumber"`
	EpisodeNumber         int    `json:"episodeNumber"`
	HasFile               bool   `json:"hasFile"`
	Monitored             bool   `json:"monitored"`
	Title                 string `json:"title"`
	Overview              string `json:"overview"`
	AirDate               string `json:"airDate"` // YYYY-MM-DD, empty when unknown
	Runtime               int    `json:"runtime"` // minutes
	AbsoluteEpisodeNumber int    `json:"absoluteEpisodeNumber"`
}

// SeriesEpisodes lists every episode of a series, all seasons.
func (c *Client) SeriesEpisodes(ctx context.Context, seriesID int) ([]Episode, error) {
	var out []Episode
	return out, c.get(ctx, fmt.Sprintf("/episode?seriesId=%d", seriesID), &out)
}

// LookupSeries asks Sonarr's metadata source about a show, tracked or not. ID is 0 when Sonarr does not track it,
// and seasons carry no statistics. nil, nil when Sonarr does not know the show.
func (c *Client) LookupSeries(ctx context.Context, tvdbID int) (*Series, error) {
	var raw []struct {
		Series
		ID *int `json:"id"` // null for shows Sonarr does not track
	}
	if err := c.get(ctx, fmt.Sprintf("/series/lookup?term=tvdb:%d", tvdbID), &raw); err != nil {
		return nil, err
	}
	for _, r := range raw {
		if r.TVDBID == tvdbID {
			out := r.Series
			if r.ID != nil {
				out.ID = *r.ID
			}
			return &out, nil
		}
	}
	return nil, nil
}

// MonitorSeasons turns monitoring on for the given seasons of a tracked series (and the series itself), then
// searches each of them. It never unmonitors anything. In dry-run mode both writes are blocked.
func (c *Client) MonitorSeasons(ctx context.Context, seriesID int, seasons []int) error {
	var series map[string]any
	if err := c.get(ctx, fmt.Sprintf("/series/%d", seriesID), &series); err != nil {
		return fmt.Errorf("read series: %w", err)
	}
	want := map[int]bool{}
	for _, n := range seasons {
		want[n] = true
	}
	if list, ok := series["seasons"].([]any); ok {
		for _, s := range list {
			if m, ok := s.(map[string]any); ok {
				n, _ := m["seasonNumber"].(float64)
				if want[int(n)] {
					m["monitored"] = true
				}
			}
		}
	}
	series["monitored"] = true
	if err := c.send(ctx, http.MethodPut, fmt.Sprintf("/series/%d", seriesID), series, nil); err != nil {
		if errors.Is(err, ErrDryRun) {
			return err
		}
		return fmt.Errorf("monitor seasons: %w", err)
	}
	for _, n := range seasons {
		cmd := map[string]any{"name": "SeasonSearch", "seriesId": seriesID, "seasonNumber": n}
		if err := c.send(ctx, http.MethodPost, "/command", cmd, nil); err != nil {
			if errors.Is(err, ErrDryRun) {
				return err
			}
			return fmt.Errorf("search season %d: %w", n, err)
		}
	}
	return nil
}

// Episodes lists the episodes of one season of a series.
func (c *Client) Episodes(ctx context.Context, seriesID, season int) ([]Episode, error) {
	var out []Episode
	return out, c.get(ctx, fmt.Sprintf("/episode?seriesId=%d&seasonNumber=%d", seriesID, season), &out)
}

// AllSeries lists every series Sonarr tracks.
func (c *Client) AllSeries(ctx context.Context) ([]Series, error) {
	var out []Series
	return out, c.get(ctx, "/series", &out)
}

// Complete is true when every monitored, aired episode has a file.
func (s Series) Complete() bool {
	return s.Statistics.EpisodeCount > 0 && s.Statistics.EpisodeFileCount >= s.Statistics.EpisodeCount
}

func (c *Client) SeriesByTVDB(ctx context.Context, tvdbID int) (*Series, error) {
	var out []Series
	if err := c.get(ctx, fmt.Sprintf("/series?tvdbId=%d", tvdbID), &out); err != nil {
		return nil, err
	}
	for i := range out {
		if out[i].TVDBID == tvdbID {
			return &out[i], nil
		}
	}
	return nil, nil
}

func (c *Client) Series(ctx context.Context, id int) (*Series, error) {
	var out Series
	return &out, c.get(ctx, fmt.Sprintf("/series/%d", id), &out)
}

// AddSeries adds the series with only the given seasons monitored and starts a search; a non-empty
// seriesType ("anime") overrides the type Sonarr suggested.
// An existing series gets the given seasons monitored and searched (MonitorSeasons, nothing unmonitored).
// In dry-run mode every write is blocked.
func (c *Client) AddSeries(ctx context.Context, tvdbID, qualityProfileID int, rootFolder string, seasons []int, seriesType string) (int, error) {
	if existing, err := c.SeriesByTVDB(ctx, tvdbID); err != nil {
		return 0, err
	} else if existing != nil {
		if len(seasons) == 0 {
			return existing.ID, nil
		}
		if err := c.MonitorSeasons(ctx, existing.ID, seasons); err != nil {
			return existing.ID, err
		}
		return existing.ID, nil
	}
	var results []map[string]any
	if err := c.get(ctx, fmt.Sprintf("/series/lookup?term=tvdb:%d", tvdbID), &results); err != nil {
		return 0, fmt.Errorf("lookup series: %w", err)
	}
	if len(results) == 0 {
		return 0, fmt.Errorf("lookup series: tvdb %d not found in Sonarr", tvdbID)
	}
	series := results[0]

	want := map[int]bool{}
	for _, n := range seasons {
		want[n] = true
	}
	if list, ok := series["seasons"].([]any); ok {
		for _, s := range list {
			if m, ok := s.(map[string]any); ok {
				n, _ := m["seasonNumber"].(float64)
				m["monitored"] = want[int(n)]
			}
		}
	}
	series["qualityProfileId"] = qualityProfileID
	series["rootFolderPath"] = rootFolder
	series["monitored"] = true
	series["seasonFolder"] = true
	if seriesType != "" { // standard | daily | anime; empty keeps what Sonarr's lookup suggested
		series["seriesType"] = seriesType
	}
	series["addOptions"] = map[string]any{"ignoreEpisodesWithFiles": true, "searchForMissingEpisodes": true}
	// Sonarr v3 still needs a language profile; v4 has none (404 here means v4).
	var langs []struct {
		ID int `json:"id"`
	}
	if err := c.get(ctx, "/languageprofile", &langs); err == nil && len(langs) > 0 {
		series["languageProfileId"] = langs[0].ID
	}

	var created Series
	if err := c.send(ctx, http.MethodPost, "/series", series, &created); err != nil {
		if errors.Is(err, ErrDryRun) {
			return 0, err
		}
		return 0, fmt.Errorf("add series: %w", err)
	}
	return created.ID, nil
}
