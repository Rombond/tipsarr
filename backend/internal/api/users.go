package api

import (
	"context"
	"net/http"
	"regexp"

	"github.com/Rombond/tipsarr/backend/internal/store"
	"github.com/danielgtaylor/huma/v2"
)

var (
	regionRe   = regexp.MustCompile(`^([A-Z]{2})?$`)
	languageRe = regexp.MustCompile(`^([a-z]{2}(-[A-Z]{2})?)?$`)
)

type prefsBody struct {
	Region   *string `json:"region,omitempty" doc:"ISO 3166 country, e.g. FR (empty = none). Used for release dates and the box-office region"`
	Language *string `json:"language,omitempty" doc:"TMDB language, e.g. fr-FR or fr (empty = English)"`
	// RatingSource is the score shown on posters. Only movies have IMDb, Metacritic and Rotten Tomatoes scores (through Radarr); anything else shows TMDB.
	RatingSource *string `json:"ratingSource,omitempty" enum:"tmdb,imdb,metacritic,rottenTomatoes," doc:"Score shown on posters (empty = tmdb)"`
}

func (b prefsBody) check() error {
	if b.Region != nil && !regionRe.MatchString(*b.Region) {
		return fail(422, "bad_region", "region must be a 2-letter country code like FR")
	}
	if b.Language != nil && !languageRe.MatchString(*b.Language) {
		return fail(422, "bad_language", "language must look like fr or fr-FR")
	}
	if b.RatingSource != nil && !validRatingSource(*b.RatingSource) {
		return fail(422, "bad_rating_source", "ratingSource must be tmdb, imdb, metacritic or rottenTomatoes")
	}
	return nil
}

func validRatingSource(v string) bool {
	switch v {
	case "", "tmdb", "imdb", "metacritic", "rottenTomatoes":
		return true
	}
	return false
}

func (b prefsBody) apply(u *store.User) {
	if b.Region != nil {
		u.Region = *b.Region
	}
	if b.Language != nil {
		u.Language = *b.Language
	}
	if b.RatingSource != nil {
		u.RatingSource = *b.RatingSource
	}
}

func registerUsers(api huma.API, d Deps) {
	sec := []map[string][]string{{"session": {}}, {"bearer": {}}}
	adminErrs := []int{http.StatusUnauthorized, http.StatusForbidden}

	huma.Register(api, huma.Operation{
		OperationID: "updateMe", Method: http.MethodPatch, Path: "/me",
		Summary: "Change my region and language", Tags: []string{"auth"}, Security: sec,
		Errors: []int{http.StatusUnauthorized, http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *struct{ Body prefsBody }) (*meOutput, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		if err := in.Body.check(); err != nil {
			return nil, err
		}
		// work on the stored row: the request's user may carry a language hint that must not be saved
		stored, err := d.Store.GetUser(ctx, u.ID)
		if err != nil {
			return nil, err
		}
		u = stored
		before := u.Language
		in.Body.apply(u)
		if err := d.Store.UpdatePrefs(ctx, u); err != nil {
			return nil, err
		}
		if u.Language != before && d.Suggestions != nil {
			d.Suggestions.QueueRefresh(u.ID) // stored suggestion rows hold titles in the old language
		}
		return &meOutput{Body: u}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "listUsers", Method: http.MethodGet, Path: "/admin/users",
		Summary: "All Tipsarr users (admin)", Tags: []string{"admin"}, Security: sec, Errors: adminErrs,
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body []store.User }, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		rows, err := d.Store.ListUsers(ctx)
		if rows == nil {
			rows = []store.User{}
		}
		return &struct{ Body []store.User }{rows}, err
	})

	huma.Register(api, huma.Operation{
		OperationID: "updateUser", Method: http.MethodPatch, Path: "/admin/users/{id}",
		Summary: "Change a user's role, region or language (admin)", Tags: []string{"admin"}, Security: sec,
		Errors: append(adminErrs, http.StatusNotFound, http.StatusConflict, http.StatusUnprocessableEntity),
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id" maxLength:"64"`
		Body struct {
			Role     *string `json:"role,omitempty" enum:"admin,user"`
			Region   *string `json:"region,omitempty" doc:"2-letter country code, empty to clear"`
			Language *string `json:"language,omitempty" doc:"e.g. fr or fr-FR, empty to clear"`
		}
	}) (*meOutput, error) {
		me, err := requireAdmin(ctx)
		if err != nil {
			return nil, err
		}
		prefs := prefsBody{Region: in.Body.Region, Language: in.Body.Language}
		if err := prefs.check(); err != nil {
			return nil, err
		}
		u, err := d.Store.GetUser(ctx, in.ID)
		if err != nil {
			return nil, fail(404, "not_found", "not found")
		}
		roleChanged := false
		if in.Body.Role != nil && *in.Body.Role != u.Role {
			if u.ID == me.ID {
				return nil, fail(409, "own_role", "you cannot change your own role")
			}
			if u.Role == store.RoleAdmin {
				if n, err := d.Store.CountAdmins(ctx); err != nil {
					return nil, err
				} else if n <= 1 {
					return nil, fail(409, "last_admin", "at least one admin is required")
				}
			}
			u.Role = *in.Body.Role
			roleChanged = true
		}
		prefs.apply(u)
		if err := d.Store.UpdateUser(ctx, u); err != nil {
			return nil, err
		}
		if roleChanged { // a demoted (or promoted) user signs in again with their new rights
			if err := d.Store.DeleteUserSessions(ctx, u.ID); err != nil {
				return nil, err
			}
		}
		return &meOutput{Body: u}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "importUsers", Method: http.MethodPost, Path: "/admin/users/import",
		Summary:     "Import users from Jellyfin (admin)",
		Description: "Adds Jellyfin users who have not logged in to Tipsarr yet (needs the Jellyfin API key). Existing users keep their role.",
		Tags:        []string{"admin"}, Security: sec, Errors: append(adminErrs, http.StatusServiceUnavailable),
	}, func(ctx context.Context, _ *struct{}) (*struct {
		Body struct {
			Created int `json:"created"`
			Total   int `json:"total"`
		}
	}, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		created, total, err := d.Library.ImportUsers(ctx)
		if err != nil {
			return nil, fail(503, "jellyfin_unavailable", err.Error())
		}
		out := &struct {
			Body struct {
				Created int `json:"created"`
				Total   int `json:"total"`
			}
		}{}
		out.Body.Created, out.Body.Total = created, total
		return out, nil
	})
}
