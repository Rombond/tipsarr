package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/Rombond/tipsarr/backend/internal/suggestions"
	"github.com/danielgtaylor/huma/v2"
)

func registerSuggestions(api huma.API, d Deps) {
	sec := []map[string][]string{{"session": {}}, {"bearer": {}}}

	huma.Register(api, huma.Operation{
		OperationID: "suggestions", Method: http.MethodGet, Path: "/suggestions",
		Summary:     "Your suggestion rows: \"Recommended for you\" and \"Because you watched …\"",
		Description: "Generated the first time it is called, then recomputed only when your watch history changed. Display only: nothing is ever requested automatically.",
		Tags:        []string{"suggestions"}, Security: sec,
		Errors: []int{http.StatusUnauthorized, http.StatusBadGateway, http.StatusServiceUnavailable},
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body *suggestions.Result }, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		res, err := d.Suggestions.Get(ctx, u)
		return &struct{ Body *suggestions.Result }{res}, mediaErr(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "refreshSuggestions", Method: http.MethodPost, Path: "/suggestions/refresh",
		Summary: "Recompute my suggestions in the background (once a minute at most)", Tags: []string{"suggestions"}, Security: sec,
		DefaultStatus: http.StatusAccepted, Errors: []int{http.StatusUnauthorized, http.StatusTooManyRequests},
	}, func(ctx context.Context, _ *struct{}) (*struct{}, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		if err := d.Suggestions.Force(ctx, u.ID); errors.Is(err, suggestions.ErrTooSoon) {
			return nil, fail(429, "refresh_too_soon", err.Error())
		} else if err != nil {
			return nil, err
		}
		return &struct{}{}, nil
	})
}
