package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/Rombond/tipsarr/backend/internal/teasers"
	"github.com/danielgtaylor/huma/v2"
)

func registerTeasers(api huma.API, d Deps) {
	sec := []map[string][]string{{"session": {}}, {"bearer": {}}}

	huma.Register(api, huma.Operation{
		OperationID: "teasers", Method: http.MethodGet, Path: "/teasers",
		Summary:     "The next trailers of the Teasers feed",
		Description: "Titles you have not swiped past yet, mixed from your suggestions, trending and upcoming, each with a trailer in your language. Skips titles you watched, blocked, requested or already have in the library. Display only: nothing is requested automatically.",
		Tags:        []string{"teasers"}, Security: sec,
		Errors: []int{http.StatusUnauthorized, http.StatusBadGateway, http.StatusServiceUnavailable},
	}, func(ctx context.Context, in *struct {
		Limit   int    `query:"limit" minimum:"1" maximum:"20" default:"8"`
		Exclude string `query:"exclude" doc:"Comma-separated type:id keys the app already holds, for example movie:603,tv:1399"`
	}) (*struct{ Body *teasers.Feed }, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		exclude := map[string]bool{}
		for _, k := range strings.Split(in.Exclude, ",") {
			if k = strings.TrimSpace(k); k != "" {
				exclude[k] = true
			}
		}
		f, err := d.Teasers.Next(ctx, u, in.Limit, exclude)
		return &struct{ Body *teasers.Feed }{f}, mediaErr(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "teaserSeen", Method: http.MethodPost, Path: "/teasers/seen",
		Summary: "Mark a title as swiped past, so the feed never shows it again", Tags: []string{"teasers"}, Security: sec,
		DefaultStatus: http.StatusNoContent, Errors: []int{http.StatusUnauthorized},
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
		return &struct{}{}, d.Teasers.MarkSeen(ctx, u.ID, in.Body.Type, in.Body.TMDBID)
	})
}
