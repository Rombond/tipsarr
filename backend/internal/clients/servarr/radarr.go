package servarr

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

type Movie struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	TMDBID    int    `json:"tmdbId"`
	HasFile   bool   `json:"hasFile"`
	Monitored bool   `json:"monitored"`
	Year      int    `json:"year"`
}

// MovieByTMDB returns the movie if Radarr already has it.
func (c *Client) MovieByTMDB(ctx context.Context, tmdbID int) (*Movie, error) {
	var out []Movie
	if err := c.get(ctx, fmt.Sprintf("/movie?tmdbId=%d", tmdbID), &out); err != nil {
		return nil, err
	}
	for i := range out {
		if out[i].TMDBID == tmdbID {
			return &out[i], nil
		}
	}
	return nil, nil
}

func (c *Client) Movie(ctx context.Context, id int) (*Movie, error) {
	var out Movie
	return &out, c.get(ctx, fmt.Sprintf("/movie/%d", id), &out)
}

// AddMovie adds the movie to Radarr and starts a search. It is idempotent: if Radarr already
// has the movie, nothing is written and the existing id is returned. In dry-run mode the
// reads happen but the POST is blocked and ErrDryRun is returned.
func (c *Client) AddMovie(ctx context.Context, tmdbID, qualityProfileID int, rootFolder string) (int, error) {
	if existing, err := c.MovieByTMDB(ctx, tmdbID); err != nil {
		return 0, err
	} else if existing != nil {
		return existing.ID, nil
	}
	var lookup map[string]any
	if err := c.get(ctx, fmt.Sprintf("/movie/lookup/tmdb?tmdbId=%d", tmdbID), &lookup); err != nil {
		return 0, fmt.Errorf("lookup movie: %w", err)
	}
	lookup["qualityProfileId"] = qualityProfileID
	lookup["rootFolderPath"] = rootFolder
	lookup["monitored"] = true
	lookup["minimumAvailability"] = "released"
	lookup["addOptions"] = map[string]any{"searchForMovie": true}

	var created Movie
	if err := c.send(ctx, http.MethodPost, "/movie", lookup, &created); err != nil {
		if errors.Is(err, ErrDryRun) {
			return 0, err
		}
		return 0, fmt.Errorf("add movie: %w", err)
	}
	return created.ID, nil
}

// Movies lists every movie Radarr tracks.
func (c *Client) Movies(ctx context.Context) ([]Movie, error) {
	var out []Movie
	return out, c.get(ctx, "/movie", &out)
}

// Rating is one score Radarr knows about a movie. Value is out of 10 for IMDb and TMDB, out of 100
// for Metacritic and Rotten Tomatoes.
type Rating struct {
	Votes int     `json:"votes"`
	Value float64 `json:"value"`
}

type MovieRatings struct {
	IMDbID         string
	IMDb           *Rating
	Metacritic     *Rating
	RottenTomatoes *Rating
}

// RatingsByTMDB asks Radarr (a read) for what its metadata service knows about a movie: IMDb,
// Metacritic and Rotten Tomatoes scores. A score Radarr does not have is nil.
func (c *Client) RatingsByTMDB(ctx context.Context, tmdbID int) (*MovieRatings, error) {
	var out struct {
		IMDbID  string `json:"imdbId"`
		Ratings struct {
			IMDb           *Rating `json:"imdb"`
			Metacritic     *Rating `json:"metacritic"`
			RottenTomatoes *Rating `json:"rottenTomatoes"`
		} `json:"ratings"`
	}
	if err := c.get(ctx, fmt.Sprintf("/movie/lookup/tmdb?tmdbId=%d", tmdbID), &out); err != nil {
		return nil, err
	}
	keep := func(r *Rating) *Rating {
		if r == nil || r.Value <= 0 {
			return nil
		}
		return r
	}
	return &MovieRatings{IMDbID: out.IMDbID, IMDb: keep(out.Ratings.IMDb), Metacritic: keep(out.Ratings.Metacritic), RottenTomatoes: keep(out.Ratings.RottenTomatoes)}, nil
}
