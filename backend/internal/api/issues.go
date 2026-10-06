package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/Rombond/tipsarr/backend/internal/issues"
	"github.com/Rombond/tipsarr/backend/internal/media"
	"github.com/Rombond/tipsarr/backend/internal/store"
	"github.com/danielgtaylor/huma/v2"
)

func issueErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, store.ErrNotFound), errors.Is(err, media.ErrNotFound):
		return fail(404, "not_found", "not found")
	case errors.Is(err, media.ErrNotConfigured):
		return fail(503, "tmdb_not_configured", "TMDB API key is not configured")
	case errors.Is(err, issues.ErrForbidden):
		return fail(403, "forbidden", err.Error())
	case errors.Is(err, issues.ErrInvalid):
		return fail(422, "invalid_request", err.Error())
	}
	return err
}

type issueCounts struct {
	Open     int `json:"open"`
	Resolved int `json:"resolved"`
}

func registerIssues(api huma.API, d Deps) {
	sec := []map[string][]string{{"session": {}}}
	errs := []int{http.StatusUnauthorized, http.StatusNotFound}

	huma.Register(api, huma.Operation{
		OperationID: "createIssue", Method: http.MethodPost, Path: "/issues",
		Summary:     "Report a problem with a movie or show",
		Description: "Admins are notified through the webhooks (issue.created) and the live event stream.",
		Tags:        []string{"issues"}, Security: sec, DefaultStatus: http.StatusCreated,
		Errors: []int{http.StatusUnauthorized, http.StatusNotFound, http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *struct {
		Body struct {
			Type    string `json:"type" enum:"movie,tv"`
			TMDBID  int    `json:"tmdbId" minimum:"1"`
			Kind    string `json:"kind" enum:"video,audio,subtitles,other"`
			Season  int    `json:"season,omitempty" minimum:"0" doc:"TV only, 0 = whole show"`
			Episode int    `json:"episode,omitempty" minimum:"0" doc:"TV only, 0 = whole season"`
			Message string `json:"message" minLength:"1" maxLength:"2000"`
		}
	}) (*struct{ Body *issues.Thread }, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		b := in.Body
		v, err := d.Issues.Create(ctx, u, issues.CreateParams{Type: b.Type, TMDBID: b.TMDBID, Kind: b.Kind, Season: b.Season, Episode: b.Episode, Message: b.Message})
		return &struct{ Body *issues.Thread }{v}, issueErr(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "listIssues", Method: http.MethodGet, Path: "/issues",
		Summary: "List issues (admins see everyone's, users their own)", Tags: []string{"issues"}, Security: sec,
		Errors: []int{http.StatusUnauthorized},
	}, func(ctx context.Context, in *struct {
		Filter string `query:"filter" enum:"all,open,resolved" default:"all"`
		Type   string `query:"type" maxLength:"5" doc:"With tmdbId: only the issues of one title"`
		TMDBID int    `query:"tmdbId" minimum:"0"`
		Take   int    `query:"take" minimum:"1" maximum:"100" default:"20"`
		Skip   int    `query:"skip" minimum:"0" default:"0"`
	}) (*struct {
		Body struct {
			Total int                `json:"total"`
			Items []issues.IssueView `json:"items"`
		}
	}, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		p := issues.ListParams{Filter: in.Filter, Take: in.Take, Skip: in.Skip}
		if in.TMDBID > 0 && in.Type != "" {
			p.MediaType, p.TMDBID = in.Type, in.TMDBID
		}
		res, err := d.Issues.List(ctx, u, p)
		if err != nil {
			return nil, err
		}
		out := &struct {
			Body struct {
				Total int                `json:"total"`
				Items []issues.IssueView `json:"items"`
			}
		}{}
		out.Body.Total, out.Body.Items = res.Total, res.Items
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "issueCounts", Method: http.MethodGet, Path: "/issues/counts",
		Summary: "Open and resolved issue totals", Tags: []string{"issues"}, Security: sec, Errors: []int{http.StatusUnauthorized},
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body issueCounts }, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		o, r, err := d.Issues.Counts(ctx, u)
		return &struct{ Body issueCounts }{issueCounts{o, r}}, err
	})

	type idIn struct {
		ID string `path:"id" maxLength:"32"`
	}

	huma.Register(api, huma.Operation{
		OperationID: "getIssue", Method: http.MethodGet, Path: "/issues/{id}",
		Summary: "One issue with its comments", Tags: []string{"issues"}, Security: sec, Errors: errs,
	}, func(ctx context.Context, in *idIn) (*struct{ Body *issues.Thread }, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		v, err := d.Issues.Get(ctx, u, in.ID)
		return &struct{ Body *issues.Thread }{v}, issueErr(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "commentIssue", Method: http.MethodPost, Path: "/issues/{id}/comments",
		Summary: "Add a comment (reporter or admin)", Tags: []string{"issues"}, Security: sec,
		Errors: append(errs, http.StatusUnprocessableEntity),
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id" maxLength:"32"`
		Body struct {
			Message string `json:"message" minLength:"1" maxLength:"2000"`
		}
	}) (*struct{ Body *issues.Thread }, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		v, err := d.Issues.Comment(ctx, u, in.ID, in.Body.Message)
		return &struct{ Body *issues.Thread }{v}, issueErr(err)
	})

	for op, resolved := range map[string]bool{"resolve": true, "reopen": false} {
		op, resolved := op, resolved
		huma.Register(api, huma.Operation{
			OperationID: op + "Issue", Method: http.MethodPost, Path: "/issues/{id}/" + op,
			Summary: strings.ToUpper(op[:1]) + op[1:] + " an issue (reporter or admin)", Tags: []string{"issues"}, Security: sec, Errors: errs,
		}, func(ctx context.Context, in *idIn) (*struct{ Body *issues.Thread }, error) {
			u, err := requireUser(ctx)
			if err != nil {
				return nil, err
			}
			v, err := d.Issues.SetResolved(ctx, u, in.ID, resolved)
			return &struct{ Body *issues.Thread }{v}, issueErr(err)
		})
	}

	huma.Register(api, huma.Operation{
		OperationID: "deleteIssue", Method: http.MethodDelete, Path: "/issues/{id}",
		Summary: "Delete an issue (admin)", Tags: []string{"issues"}, Security: sec, DefaultStatus: http.StatusNoContent,
		Errors: append(errs, http.StatusForbidden),
	}, func(ctx context.Context, in *idIn) (*struct{}, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		return &struct{}{}, issueErr(d.Issues.Delete(ctx, u, in.ID))
	})
}
