package api

import (
	"context"
	"errors"
	"github.com/Rombond/tipsarr/backend/internal/ratings"
	"log/slog"
	"net/http"

	"github.com/Rombond/tipsarr/backend/internal/media"
	"github.com/danielgtaylor/huma/v2"
)

type PageParam struct {
	Page int `query:"page" minimum:"1" maximum:"500" default:"1" doc:"Page number"`
}

type listOutput struct{ Body *media.List }

func prefs(ctx context.Context) (media.Opts, error) {
	u, err := requireUser(ctx)
	if err != nil {
		return media.Opts{}, err
	}
	return media.Opts{Language: u.Language, Region: u.Region}, nil
}

// mediaErr maps service errors onto HTTP errors.
func mediaErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, media.ErrNotConfigured):
		return fail(503, "tmdb_not_configured", "TMDB API key is not configured (admin: Settings)")
	case errors.Is(err, media.ErrNotFound):
		return fail(404, "not_found", "not found")
	default:
		slog.Warn("TMDB request failed", "err", err)
		return fail(502, "tmdb_failed", "TMDB request failed")
	}
}

func registerDiscover(api huma.API, d Deps) {
	sec := []map[string][]string{{"session": {}}}
	errs := []int{http.StatusUnauthorized, http.StatusBadGateway, http.StatusServiceUnavailable}

	huma.Register(api, huma.Operation{
		OperationID: "discoverTrending", Method: http.MethodGet, Path: "/discover/trending",
		Summary: "Trending movies and shows this week", Tags: []string{"discover"}, Security: sec, Errors: errs,
	}, func(ctx context.Context, in *PageParam) (*listOutput, error) {
		o, err := prefs(ctx)
		if err != nil {
			return nil, err
		}
		l, err := d.Media.Trending(ctx, o, in.Page)
		return &listOutput{Body: l}, mediaErr(err)
	})

	type discoverIn struct {
		PageParam
		Genre   int `query:"genre" minimum:"0" maximum:"1000000" doc:"TMDB genre id"`
		Keyword int `query:"keyword" minimum:"0" maximum:"100000000" doc:"TMDB keyword (tag) id"`
	}
	for t, path := range map[string]string{"movie": "/discover/movies", "tv": "/discover/tv"} {
		t, path := t, path
		huma.Register(api, huma.Operation{
			OperationID: "discover_" + t, Method: http.MethodGet, Path: path,
			Summary: "Popular " + t + " titles, optionally by genre", Tags: []string{"discover"}, Security: sec, Errors: errs,
		}, func(ctx context.Context, in *discoverIn) (*listOutput, error) {
			o, err := prefs(ctx)
			if err != nil {
				return nil, err
			}
			l, err := d.Media.Discover(ctx, o, t, in.Genre, in.Keyword, in.Page)
			return &listOutput{Body: l}, mediaErr(err)
		})
	}

	huma.Register(api, huma.Operation{
		OperationID: "discoverUpcoming", Method: http.MethodGet, Path: "/discover/upcoming",
		Summary: "Upcoming movies", Tags: []string{"discover"}, Security: sec, Errors: errs,
	}, func(ctx context.Context, in *PageParam) (*listOutput, error) {
		o, err := prefs(ctx)
		if err != nil {
			return nil, err
		}
		l, err := d.Media.Upcoming(ctx, o, in.Page)
		return &listOutput{Body: l}, mediaErr(err)
	})

	type typeIn struct {
		Type string `path:"type" enum:"movie,tv"`
	}
	huma.Register(api, huma.Operation{
		OperationID: "discoverGenres", Method: http.MethodGet, Path: "/discover/genres/{type}",
		Summary: "Genre list", Tags: []string{"discover"}, Security: sec, Errors: errs,
	}, func(ctx context.Context, in *typeIn) (*struct{ Body []media.Genre }, error) {
		o, err := prefs(ctx)
		if err != nil {
			return nil, err
		}
		g, err := d.Media.Genres(ctx, o, in.Type)
		return &struct{ Body []media.Genre }{g}, mediaErr(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "keyword", Method: http.MethodGet, Path: "/discover/keywords/{id}",
		Summary: "One tag (keyword) by id", Tags: []string{"discover"}, Security: sec, Errors: append(errs, http.StatusNotFound),
	}, func(ctx context.Context, in *struct {
		ID int `path:"id" minimum:"1"`
	}) (*struct{ Body *media.Keyword }, error) {
		k, err := d.Media.Keyword(ctx, in.ID)
		return &struct{ Body *media.Keyword }{k}, mediaErr(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "search", Method: http.MethodGet, Path: "/search",
		Summary: "Search movies, shows and people", Tags: []string{"discover"}, Security: sec, Errors: errs,
	}, func(ctx context.Context, in *struct {
		PageParam
		Query string `query:"q" required:"true" minLength:"1" maxLength:"200"`
		Tags  bool   `query:"tags" doc:"Also look for titles carrying a matching tag (slower; for the results page)"`
	}) (*struct{ Body *media.SearchResult }, error) {
		o, err := prefs(ctx)
		if err != nil {
			return nil, err
		}
		r, err := d.Media.Search(ctx, o, in.Query, in.Page, in.Tags)
		return &struct{ Body *media.SearchResult }{r}, mediaErr(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "movieRatings", Method: http.MethodGet, Path: "/media/movie/{id}/ratings",
		Summary:     "IMDb, Metacritic and Rotten Tomatoes scores of a movie, from Radarr",
		Description: "Empty when no Radarr is configured or Radarr has no score. Shows have none.",
		Tags:        []string{"media"}, Security: sec, Errors: errs,
	}, func(ctx context.Context, in *struct {
		ID int `path:"id" minimum:"1"`
	}) (*struct{ Body ratings.MovieScores }, error) {
		if _, err := requireUser(ctx); err != nil {
			return nil, err
		}
		return &struct{ Body ratings.MovieScores }{d.Ratings.Movie(ctx, in.ID)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "mediaDetail", Method: http.MethodGet, Path: "/media/{type}/{id}",
		Summary: "Movie or show details with cast, recommendations and similar titles", Tags: []string{"media"},
		Security: sec, Errors: append(errs, http.StatusNotFound),
	}, func(ctx context.Context, in *struct {
		Type string `path:"type" enum:"movie,tv"`
		ID   int    `path:"id" minimum:"1"`
	}) (*struct{ Body *media.Detail }, error) {
		o, err := prefs(ctx)
		if err != nil {
			return nil, err
		}
		r, err := d.Media.Detail(ctx, o, in.Type, in.ID)
		return &struct{ Body *media.Detail }{r}, mediaErr(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "tvSeason", Method: http.MethodGet, Path: "/media/tv/{id}/seasons/{season}",
		Summary: "Episodes of a season", Tags: []string{"media"}, Security: sec, Errors: append(errs, http.StatusNotFound),
	}, func(ctx context.Context, in *struct {
		ID     int `path:"id" minimum:"1"`
		Season int `path:"season" minimum:"0"`
	}) (*struct{ Body *media.SeasonDetail }, error) {
		o, err := prefs(ctx)
		if err != nil {
			return nil, err
		}
		r, err := d.Media.Season(ctx, o, in.ID, in.Season)
		return &struct{ Body *media.SeasonDetail }{r}, mediaErr(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "person", Method: http.MethodGet, Path: "/person/{id}",
		Summary: "Person with credits", Tags: []string{"media"}, Security: sec, Errors: append(errs, http.StatusNotFound),
	}, func(ctx context.Context, in *struct {
		ID int `path:"id" minimum:"1"`
	}) (*struct{ Body *media.PersonDetail }, error) {
		o, err := prefs(ctx)
		if err != nil {
			return nil, err
		}
		r, err := d.Media.PersonDetail(ctx, o, in.ID)
		return &struct{ Body *media.PersonDetail }{r}, mediaErr(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "collection", Method: http.MethodGet, Path: "/collection/{id}",
		Summary: "Movie collection", Tags: []string{"media"}, Security: sec, Errors: append(errs, http.StatusNotFound),
	}, func(ctx context.Context, in *struct {
		ID int `path:"id" minimum:"1"`
	}) (*struct{ Body *media.CollectionDetail }, error) {
		o, err := prefs(ctx)
		if err != nil {
			return nil, err
		}
		r, err := d.Media.Collection(ctx, o, in.ID)
		return &struct{ Body *media.CollectionDetail }{r}, mediaErr(err)
	})
}
