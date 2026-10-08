package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/Rombond/tipsarr/backend/internal/marks"
	"github.com/Rombond/tipsarr/backend/internal/media"
	"github.com/danielgtaylor/huma/v2"
)

func registerMarks(api huma.API, d Deps) {
	sec := []map[string][]string{{"session": {}}, {"bearer": {}}}

	mapErr := func(err error) error {
		if errors.Is(err, marks.ErrInvalid) {
			return fail(422, "invalid_request", err.Error())
		}
		return mediaErr(err)
	}

	for _, k := range []struct{ kind, path, label string }{
		{"watchlist", "/watchlist", "watchlist"},
		{"blocklist", "/blocklist", "blocklist (titles hidden from my suggestions)"},
	} {
		kind, path, label := k.kind, k.path, k.label
		huma.Register(api, huma.Operation{
			OperationID: "list_" + kind, Method: http.MethodGet, Path: path,
			Summary: "My " + label, Tags: []string{kind}, Security: sec, Errors: []int{http.StatusUnauthorized},
		}, func(ctx context.Context, _ *struct{}) (*struct{ Body []media.Item }, error) {
			u, err := requireUser(ctx)
			if err != nil {
				return nil, err
			}
			items, err := d.Marks.List(ctx, u, kind)
			return &struct{ Body []media.Item }{items}, err
		})

		huma.Register(api, huma.Operation{
			OperationID: "add_" + kind, Method: http.MethodPost, Path: path,
			Summary: "Add a title to my " + label, Tags: []string{kind}, Security: sec, DefaultStatus: http.StatusNoContent,
			Errors: []int{http.StatusUnauthorized, http.StatusNotFound, http.StatusUnprocessableEntity},
		}, func(ctx context.Context, in *struct {
			Body struct {
				Type   string `json:"type" enum:"movie,tv"`
				TMDBID int    `json:"tmdbId" minimum:"1"`
			}
		}) (*struct{}, error) {
			u, err := requireUser(ctx)
			if err != nil {
				return nil, err
			}
			return &struct{}{}, mapErr(d.Marks.Add(ctx, u, kind, in.Body.Type, in.Body.TMDBID))
		})

		huma.Register(api, huma.Operation{
			OperationID: "remove_" + kind, Method: http.MethodDelete, Path: path + "/{type}/{id}",
			Summary: "Remove a title from my " + label, Tags: []string{kind}, Security: sec, DefaultStatus: http.StatusNoContent,
			Errors: []int{http.StatusUnauthorized},
		}, func(ctx context.Context, in *struct {
			Type string `path:"type" enum:"movie,tv"`
			ID   int    `path:"id" minimum:"1"`
		}) (*struct{}, error) {
			u, err := requireUser(ctx)
			if err != nil {
				return nil, err
			}
			return &struct{}{}, mapErr(d.Marks.Remove(ctx, u, kind, in.Type, in.ID))
		})
	}

	huma.Register(api, huma.Operation{
		OperationID: "mediaFlags", Method: http.MethodGet, Path: "/media/{type}/{id}/flags",
		Summary: "Is this title on my watchlist / blocklist?", Tags: []string{"media"}, Security: sec, Errors: []int{http.StatusUnauthorized},
	}, func(ctx context.Context, in *struct {
		Type string `path:"type" enum:"movie,tv"`
		ID   int    `path:"id" minimum:"1"`
	}) (*struct{ Body marks.Flags }, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		f, err := d.Marks.Flags(ctx, u, in.Type, in.ID)
		return &struct{ Body marks.Flags }{f}, err
	})
}
