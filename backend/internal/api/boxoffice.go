package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/Rombond/tipsarr/backend/internal/boxoffice"
	"github.com/Rombond/tipsarr/backend/internal/store"
	"github.com/danielgtaylor/huma/v2"
)

func registerBoxOffice(api huma.API, d Deps) {
	adminSec := []map[string][]string{{"session": {}}}
	huma.Register(api, huma.Operation{
		OperationID: "setBoxOfficeAlias", Method: http.MethodPut, Path: "/admin/boxoffice/alias",
		Summary:     "Pin a box-office title to a TMDB movie (admin)",
		Description: "Fixes titles the automatic search matched wrongly or not at all. Applies to every stored week and to future charts.",
		Tags:        []string{"admin"}, Security: adminSec, DefaultStatus: http.StatusNoContent,
		Errors: []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *struct {
		Body struct {
			Title  string `json:"title" minLength:"1" maxLength:"255"`
			TMDBID int    `json:"tmdbId" minimum:"1"`
		}
	}) (*struct{}, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		err := d.BoxOffice.SetAlias(ctx, in.Body.Title, in.Body.TMDBID)
		if errors.Is(err, boxoffice.ErrBadTitle) {
			return nil, fail(422, "bad_title", err.Error())
		}
		return &struct{}{}, mediaErr(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "deleteBoxOfficeAlias", Method: http.MethodDelete, Path: "/admin/boxoffice/alias",
		Summary: "Remove a manual match and match automatically again (admin)", Tags: []string{"admin"}, Security: adminSec,
		DefaultStatus: http.StatusNoContent, Errors: []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, in *struct {
		Title string `query:"title" required:"true" maxLength:"255"`
	}) (*struct{}, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		return &struct{}{}, mediaErr(d.BoxOffice.RemoveAlias(ctx, in.Title))
	})

	huma.Register(api, huma.Operation{
		OperationID: "boxOffice", Method: http.MethodGet, Path: "/boxoffice",
		Summary:     "Box-office chart (top 10): weekend or full week with availability and Radarr status",
		Description: "Global, not personalised: one chart per region and week. Defaults to your region (if configured) and the latest stored week. Display only: nothing is added automatically.",
		Tags:        []string{"boxoffice"}, Security: []map[string][]string{{"session": {}}},
		Errors: []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, in *struct {
		Region string `query:"region" maxLength:"3" doc:"Region code, e.g. US, GB, FR"`
		Week   string `query:"week" maxLength:"12" doc:"Week key, e.g. 2026W40 (a trailing w means the full week)"`
		Weekly bool   `query:"weekly" doc:"List full Monday-Sunday weeks instead of weekends"`
	}) (*struct{ Body *boxoffice.Chart }, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		c, err := d.BoxOffice.Chart(ctx, u.Region, in.Region, in.Week, in.Weekly)
		if errors.Is(err, store.ErrNotFound) {
			return nil, fail(404, "chart_not_found", "no chart stored for that week")
		}
		return &struct{ Body *boxoffice.Chart }{c}, err
	})
}
