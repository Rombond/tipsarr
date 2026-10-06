package api

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"

	"github.com/Rombond/tipsarr/backend/internal/jobs"
	"github.com/Rombond/tipsarr/backend/internal/library"
	"github.com/danielgtaylor/huma/v2"
)

type syncStatusBody struct {
	CanSync bool          `json:"canSync" doc:"True when a Jellyfin API key is saved"`
	Movies  int64         `json:"movies" doc:"Movies currently known in the Jellyfin library"`
	Shows   int64         `json:"shows" doc:"Shows currently known in the Jellyfin library"`
	Jobs    []jobs.Status `json:"jobs"`
}

type webhookInput struct {
	Token string `query:"token" required:"true"`
	Body  struct {
		NotificationType string `json:"NotificationType,omitempty" doc:"Jellyfin webhook plugin event, e.g. PlaybackStop or ItemAdded"`
		UserID           string `json:"UserId,omitempty"`
		ItemType         string `json:"ItemType,omitempty"`
	}
}

func registerSync(api huma.API, d Deps) {
	sec := []map[string][]string{{"session": {}}}
	adminErrs := []int{http.StatusUnauthorized, http.StatusForbidden}

	huma.Register(api, huma.Operation{
		OperationID: "syncStatus", Method: http.MethodGet, Path: "/admin/sync",
		Summary: "Library / history sync status (admin)", Tags: []string{"admin"}, Security: sec, Errors: adminErrs,
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body syncStatusBody }, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		movies, shows, err := d.Store.LibraryCounts(ctx)
		if err != nil {
			return nil, err
		}
		st, err := d.Jobs.Statuses(ctx)
		if err != nil {
			return nil, err
		}
		return &struct{ Body syncStatusBody }{syncStatusBody{CanSync: d.Library.Configured(ctx), Movies: movies, Shows: shows, Jobs: st}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "runSyncJob", Method: http.MethodPost, Path: "/admin/sync/{job}",
		Summary: "Run a sync job now, in the background (admin)", Tags: []string{"admin"}, Security: sec,
		DefaultStatus: http.StatusAccepted,
		Errors:        append(adminErrs, http.StatusConflict, http.StatusServiceUnavailable),
	}, func(ctx context.Context, in *struct {
		Job string `path:"job" enum:"library-sync,history-sync,boxoffice-refresh,servarr-import"`
	}) (*struct{}, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		if in.Job != "boxoffice-refresh" && in.Job != "servarr-import" && !d.Library.Configured(ctx) {
			return nil, fail(503, "jellyfin_key_missing", "save a Jellyfin API key in Settings first")
		}
		switch err := d.Jobs.RunNow(ctx, in.Job); {
		case errors.Is(err, jobs.ErrAlreadyRunning):
			return nil, fail(409, "job_running", "job is already running")
		case err != nil:
			return nil, err
		}
		return &struct{}{}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "jellyfinWebhook", Method: http.MethodPost, Path: "/hooks/jellyfin",
		Summary:     "Receives Jellyfin webhook events (shared-secret token)",
		Description: "Configure the Jellyfin Webhook plugin with a Generic destination pointing here and a template like {\"NotificationType\":\"{{NotificationType}}\",\"UserId\":\"{{UserId}}\",\"ItemType\":\"{{ItemType}}\"}. Library events refresh the library; everything else refreshes that user's watch history (debounced).",
		Tags:        []string{"hooks"}, DefaultStatus: http.StatusNoContent,
		Errors: []int{http.StatusUnauthorized},
	}, func(ctx context.Context, in *webhookInput) (*struct{}, error) {
		secret, err := webhookSecret(ctx, d)
		if err != nil {
			return nil, err
		}
		if subtle.ConstantTimeCompare([]byte(in.Token), []byte(secret)) != 1 {
			return nil, fail(401, "invalid_token", "invalid token")
		}
		switch in.Body.NotificationType {
		case "ItemAdded", "ItemDeleted":
			d.Library.QueueLibrary()
		default:
			d.Library.QueueUserHistory(library.NormalizeID(in.Body.UserID))
		}
		return &struct{}{}, nil
	})
}
