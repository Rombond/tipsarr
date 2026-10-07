package api

import (
	"context"
	"net/http"

	"github.com/Rombond/tipsarr/backend/internal/stats"
	"github.com/danielgtaylor/huma/v2"
)

func registerStats(api huma.API, d Deps) {
	huma.Register(api, huma.Operation{
		OperationID: "getStats", Method: http.MethodGet, Path: "/stats",
		Summary: "Watching and request statistics",
		Description: "Exact when Jellyfin's Playback Reporting plugin is installed (its plays are copied into Tipsarr), " +
			"otherwise estimated from the watch history. People see their own numbers; `user` and `all` are admin only.",
		Tags: []string{"stats"}, Security: []map[string][]string{{"session": {}}},
		Errors: []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, in *struct {
		User   string `query:"user" maxLength:"64" doc:"A user id, or all (admins). Default: yourself"`
		Period string `query:"period" enum:"30d,12m,all" default:"all"`
	}) (*struct{ Body stats.StatsReport }, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		target := u.ID
		switch {
		case in.User == "" || in.User == u.ID:
		case u.Role != "admin":
			return nil, fail(403, "admin_only", "admin only")
		case in.User == "all":
			target = ""
		default:
			target = in.User
		}
		res, err := d.Stats.Compute(ctx, target, in.Period, u.Role == "admin")
		if err != nil {
			return nil, err
		}
		return &struct{ Body stats.StatsReport }{*res}, nil
	})
}
