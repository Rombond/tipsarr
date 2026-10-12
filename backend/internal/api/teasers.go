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
		Description: "Answers 404 (code teasers_disabled) when trailer files are not set up on this server: apps then hide the tab. Titles you have not swiped past yet, mixed from your suggestions, trending and upcoming, each with a trailer in your language. Skips titles you watched, blocked, requested or already have in the library. Display only: nothing is requested automatically.",
		Tags:        []string{"teasers"}, Security: sec,
		Errors: []int{http.StatusUnauthorized, http.StatusNotFound, http.StatusBadGateway, http.StatusServiceUnavailable},
	}, func(ctx context.Context, in *struct {
		Limit   int    `query:"limit" minimum:"1" maximum:"20" default:"8"`
		Exclude string `query:"exclude" doc:"Comma-separated type:id keys the app already holds, for example movie:603,tv:1399"`
	}) (*struct{ Body *teasers.Feed }, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		if !d.Teasers.Enabled(ctx) {
			return nil, errTeasersDisabled()
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

	huma.Register(api, huma.Operation{
		OperationID: "teaserHistory", Method: http.MethodGet, Path: "/teasers/history",
		Summary:     "Titles you swiped past in the Teasers feed, newest first",
		Description: "Open to every signed-in person (their own history). Titles are looked up in your language.",
		Tags:        []string{"teasers"}, Security: sec, Errors: []int{http.StatusUnauthorized},
	}, func(ctx context.Context, in *struct {
		Limit  int `query:"limit" minimum:"1" maximum:"100" default:"40"`
		Offset int `query:"offset" minimum:"0" default:"0"`
	}) (*struct {
		Body struct {
			Items []teasers.HistoryItem `json:"items"`
			Total int                   `json:"total" doc:"How many titles the history holds in all"`
		}
	}, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		items, total, err := d.Teasers.History(ctx, u, in.Limit, in.Offset)
		if err != nil {
			return nil, mediaErr(err)
		}
		out := &struct {
			Body struct {
				Items []teasers.HistoryItem `json:"items"`
				Total int                   `json:"total" doc:"How many titles the history holds in all"`
			}
		}{}
		out.Body.Items, out.Body.Total = items, total
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "teaserForgetAll", Method: http.MethodDelete, Path: "/teasers/seen",
		Summary: "Wipe your whole Teasers history: every title can come back in the feed", Tags: []string{"teasers"}, Security: sec,
		DefaultStatus: http.StatusNoContent, Errors: []int{http.StatusUnauthorized},
	}, func(ctx context.Context, in *struct{}) (*struct{}, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		return &struct{}{}, d.Teasers.ForgetAll(ctx, u.ID)
	})

	huma.Register(api, huma.Operation{
		OperationID: "teaserForget", Method: http.MethodDelete, Path: "/teasers/seen/{type}/{id}",
		Summary: "Forget one title of your Teasers history", Tags: []string{"teasers"}, Security: sec,
		DefaultStatus: http.StatusNoContent, Errors: []int{http.StatusUnauthorized},
	}, func(ctx context.Context, in *struct {
		Type string `path:"type" enum:"movie,tv"`
		ID   int    `path:"id" minimum:"1"`
	}) (*struct{}, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		return &struct{}{}, d.Teasers.Forget(ctx, u.ID, in.Type, in.ID)
	})
}
