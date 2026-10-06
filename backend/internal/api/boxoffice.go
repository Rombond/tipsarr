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
	huma.Register(api, huma.Operation{
		OperationID: "boxOffice", Method: http.MethodGet, Path: "/boxoffice",
		Summary:     "Weekend box-office chart (top 10) with availability and Radarr status",
		Description: "Global, not personalised: one chart per region and week. Defaults to your region (if configured) and the latest stored week. Display only: nothing is added automatically.",
		Tags:        []string{"boxoffice"}, Security: []map[string][]string{{"session": {}}},
		Errors: []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, in *struct {
		Region string `query:"region" maxLength:"3" doc:"Region code, e.g. US, GB, FR"`
		Week   string `query:"week" maxLength:"12" doc:"Week key, e.g. 2026W40"`
	}) (*struct{ Body *boxoffice.Chart }, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		c, err := d.BoxOffice.Chart(ctx, u.Region, in.Region, in.Week)
		if errors.Is(err, store.ErrNotFound) {
			return nil, huma.Error404NotFound("no chart stored for that week")
		}
		return &struct{ Body *boxoffice.Chart }{c}, err
	})
}
