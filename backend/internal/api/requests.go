package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/Rombond/tipsarr/backend/internal/clients/servarr"
	"github.com/Rombond/tipsarr/backend/internal/media"
	"github.com/Rombond/tipsarr/backend/internal/requests"
	"github.com/Rombond/tipsarr/backend/internal/store"
	"github.com/danielgtaylor/huma/v2"
)

func reqErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, store.ErrNotFound), errors.Is(err, media.ErrNotFound):
		return fail(404, "not_found", "not found")
	case errors.Is(err, media.ErrNotConfigured):
		return fail(503, "tmdb_not_configured", "TMDB API key is not configured")
	case errors.Is(err, requests.ErrAlreadyAvailable):
		return fail(409, "already_available", err.Error())
	case errors.Is(err, requests.ErrDuplicate):
		return fail(409, "duplicate_request", err.Error())
	case errors.Is(err, requests.ErrBadState):
		return fail(409, "bad_state", err.Error())
	case errors.Is(err, requests.ErrNoInstance):
		return fail(409, "no_instance", err.Error())
	case errors.Is(err, requests.ErrForbidden):
		return fail(403, "forbidden", err.Error())
	case errors.Is(err, requests.ErrServarrFailed):
		return fail(502, "servarr_error", err.Error())
	case errors.Is(err, requests.ErrInvalid):
		return fail(422, "invalid_request", err.Error())
	}
	return err
}

type requestOutput struct{ Body *requests.View }

type requestList struct {
	Total int             `json:"total"`
	Items []requests.View `json:"items"`
}

func registerRequests(api huma.API, d Deps) {
	sec := []map[string][]string{{"session": {}}}
	errs := []int{http.StatusUnauthorized, http.StatusNotFound}

	huma.Register(api, huma.Operation{
		OperationID: "listRequests", Method: http.MethodGet, Path: "/requests",
		Summary: "List requests (admins see everyone's, users their own)", Tags: []string{"requests"}, Security: sec,
		Errors: []int{http.StatusUnauthorized},
	}, func(ctx context.Context, in *struct {
		Filter string `query:"filter" enum:"all,mine,pending,approved,available,declined,failed,unwatched" default:"all"`
		User   string `query:"user" maxLength:"64" doc:"Admins only: only this user's requests"`
		Take   int    `query:"take" minimum:"1" maximum:"100" default:"20"`
		Skip   int    `query:"skip" minimum:"0" default:"0"`
	}) (*struct{ Body requestList }, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		res, err := d.Requests.List(ctx, u, requests.ListParams{Filter: in.Filter, User: in.User, Take: in.Take, Skip: in.Skip})
		if err != nil {
			return nil, err
		}
		return &struct{ Body requestList }{requestList{Total: res.Total, Items: res.Items}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "requestCounts", Method: http.MethodGet, Path: "/requests/counts",
		Summary: "Requests per status", Tags: []string{"requests"}, Security: sec, Errors: []int{http.StatusUnauthorized},
	}, func(ctx context.Context, _ *struct{}) (*struct {
		Body struct {
			Pending   int `json:"pending"`
			Approved  int `json:"approved"`
			Available int `json:"available"`
			Declined  int `json:"declined"`
			Failed    int `json:"failed"`
		}
	}, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		c, err := d.Requests.Counts(ctx, u)
		if err != nil {
			return nil, err
		}
		out := &struct {
			Body struct {
				Pending   int `json:"pending"`
				Approved  int `json:"approved"`
				Available int `json:"available"`
				Declined  int `json:"declined"`
				Failed    int `json:"failed"`
			}
		}{}
		out.Body.Pending, out.Body.Approved, out.Body.Available = c[store.StatusPending], c[store.StatusApproved], c[store.StatusAvailable]
		out.Body.Declined, out.Body.Failed = c[store.StatusDeclined], c[store.StatusFailed]
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "createRequest", Method: http.MethodPost, Path: "/requests",
		Summary:     "Request a movie or show",
		Description: "Never automatic: this is always a user action. Admin requests are approved immediately. In dry-run mode an approved request is recorded but nothing is sent to Radarr/Sonarr.",
		Tags:        []string{"requests"}, Security: sec, DefaultStatus: http.StatusCreated,
		Errors: []int{http.StatusUnauthorized, http.StatusNotFound, http.StatusConflict, http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *struct {
		Body struct {
			Type             string `json:"type" enum:"movie,tv"`
			TMDBID           int    `json:"tmdbId" minimum:"1"`
			Seasons          []int  `json:"seasons,omitempty" doc:"TV only; empty means every season"`
			QualityProfileID *int   `json:"qualityProfileId,omitempty" doc:"Radarr/Sonarr quality profile to use (omit for the default); see /requests/options"`
			RootFolder       string `json:"rootFolder,omitempty" doc:"Root folder to use (omit for the default)"`
		}
	}) (*requestOutput, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		p := requests.CreateParams{Type: in.Body.Type, TMDBID: in.Body.TMDBID, Seasons: in.Body.Seasons}
		if in.Body.QualityProfileID != nil || in.Body.RootFolder != "" {
			p.Overrides = &requests.Overrides{ProfileID: in.Body.QualityProfileID, RootFolder: in.Body.RootFolder}
		}
		v, err := d.Requests.Create(ctx, u, p)
		return &requestOutput{Body: v}, reqErr(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "requestOptions", Method: http.MethodGet, Path: "/requests/options",
		Summary:     "Quality profiles (and, when allowed, root folders) you can pick when requesting",
		Description: "Read-only calls to the default Radarr (movie) or Sonarr (tv) instance.",
		Tags:        []string{"requests"}, Security: sec,
		Errors: []int{http.StatusUnauthorized, http.StatusConflict, http.StatusBadGateway},
	}, func(ctx context.Context, in *struct {
		Type string `query:"type" enum:"movie,tv" required:"true"`
	}) (*struct{ Body *requests.Options }, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		o, err := d.Requests.Options(ctx, in.Type)
		if err != nil && !errors.Is(err, requests.ErrNoInstance) {
			return nil, fail(502, "servarr_error", err.Error())
		}
		if o != nil && u.Role != store.RoleAdmin && !d.Requests.UsersMayChooseFolder(ctx) {
			o.RootFolders, o.RootFolder = []servarr.RootFolder{}, "" // everyone picks the quality profile; the folder is the admin's call
		}
		return &struct{ Body *requests.Options }{o}, reqErr(err)
	})

	type idIn struct {
		ID string `path:"id" maxLength:"32"`
	}

	huma.Register(api, huma.Operation{
		OperationID: "getRequest", Method: http.MethodGet, Path: "/requests/{id}",
		Summary: "One request", Tags: []string{"requests"}, Security: sec, Errors: errs,
	}, func(ctx context.Context, in *idIn) (*requestOutput, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		v, err := d.Requests.Get(ctx, u, in.ID)
		return &requestOutput{Body: v}, reqErr(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "requestProgress", Method: http.MethodGet, Path: "/requests/{id}/progress",
		Summary: "Live stage and download progress", Tags: []string{"requests"}, Security: sec, Errors: errs,
	}, func(ctx context.Context, in *idIn) (*requestOutput, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		v, err := d.Requests.Get(ctx, u, in.ID)
		return &requestOutput{Body: v}, reqErr(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "approveRequest", Method: http.MethodPost, Path: "/requests/{id}/approve",
		Summary: "Approve and send to Radarr/Sonarr (admin; also retries a failed request)", Tags: []string{"requests"}, Security: sec,
		Errors: []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id" maxLength:"32"`
		Body struct {
			QualityProfileID *int   `json:"qualityProfileId,omitempty"`
			RootFolder       string `json:"rootFolder,omitempty"`
		}
	}) (*requestOutput, error) {
		u, err := requireAdmin(ctx)
		if err != nil {
			return nil, err
		}
		v, err := d.Requests.Approve(ctx, u, in.ID, &requests.Overrides{ProfileID: in.Body.QualityProfileID, RootFolder: in.Body.RootFolder})
		return &requestOutput{Body: v}, reqErr(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "updateRequestOptions", Method: http.MethodPatch, Path: "/requests/{id}",
		Summary: "Change the quality profile / root folder of a pending or failed request (requester or admin)", Tags: []string{"requests"}, Security: sec,
		Errors: []int{http.StatusUnauthorized, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id" maxLength:"32"`
		Body struct {
			QualityProfileID *int   `json:"qualityProfileId,omitempty" doc:"Omit for the instance default"`
			RootFolder       string `json:"rootFolder,omitempty" doc:"Empty for the instance default"`
		}
	}) (*requestOutput, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		v, err := d.Requests.UpdateOptions(ctx, u, in.ID, requests.Overrides{ProfileID: in.Body.QualityProfileID, RootFolder: in.Body.RootFolder})
		return &requestOutput{Body: v}, reqErr(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "retryRequest", Method: http.MethodPost, Path: "/requests/{id}/retry",
		Summary: "Retry a failed request (admin)", Tags: []string{"requests"}, Security: sec,
		Errors: []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, in *idIn) (*requestOutput, error) {
		u, err := requireAdmin(ctx)
		if err != nil {
			return nil, err
		}
		v, err := d.Requests.Approve(ctx, u, in.ID, nil)
		return &requestOutput{Body: v}, reqErr(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "declineRequest", Method: http.MethodPost, Path: "/requests/{id}/decline",
		Summary: "Decline a request (admin)", Tags: []string{"requests"}, Security: sec,
		Errors: []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id" maxLength:"32"`
		Body struct {
			Reason string `json:"reason,omitempty" maxLength:"500"`
		}
	}) (*requestOutput, error) {
		u, err := requireAdmin(ctx)
		if err != nil {
			return nil, err
		}
		v, err := d.Requests.Decline(ctx, u, in.ID, in.Body.Reason)
		return &requestOutput{Body: v}, reqErr(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "deleteRequest", Method: http.MethodDelete, Path: "/requests/{id}",
		Summary: "Delete a request (admin: any; owner: unfinished ones). A request still waiting in Radarr/Sonarr is removed there too (files are kept)", Tags: []string{"requests"}, Security: sec,
		DefaultStatus: http.StatusNoContent, Errors: append(errs, http.StatusForbidden, http.StatusBadGateway),
	}, func(ctx context.Context, in *idIn) (*struct{}, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		return &struct{}{}, reqErr(d.Requests.Delete(ctx, u, in.ID))
	})
}
