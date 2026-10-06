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
	Statistics struct {
		EpisodeFileCount int `json:"episodeFileCount"`
		EpisodeCount     int `json:"episodeCount"`
	} `json:"statistics"`
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

// AddSeries adds the series with only the given seasons monitored and starts a search.
// Idempotent: an existing series is returned untouched. In dry-run mode the POST is blocked.
func (c *Client) AddSeries(ctx context.Context, tvdbID, qualityProfileID int, rootFolder string, seasons []int) (int, error) {
	if existing, err := c.SeriesByTVDB(ctx, tvdbID); err != nil {
		return 0, err
	} else if existing != nil {
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
