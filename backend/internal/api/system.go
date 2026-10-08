package api

import (
	"context"
	"github.com/Rombond/tipsarr/backend/internal/media"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type healthOutput struct {
	Body struct {
		OK bool `json:"ok"`
	}
}

// statusFeatures tells an app which optional parts this server has.
type statusFeatures struct {
	Push bool `json:"push" doc:"Push notifications are set up on this server"`
}

type statusOutput struct {
	Body struct {
		Version string `json:"version"`
		DB      string `json:"db" doc:"Database engine: sqlite, postgres or mysql"`
		DryRun  bool   `json:"dryRun" doc:"When true nothing is ever sent to Radarr/Sonarr"`
		// Whether non-admins may pick the root folder when requesting (the profile is always theirs to pick).
		UserFolderChoice bool `json:"userFolderChoice"`
		// The app-wide default language (a TMDB tag such as fr-FR); empty means the browser decides.
		DefaultLanguage string `json:"defaultLanguage"`
		// APIVersion counts breaking changes of /api/v1: it only grows when an app must be updated.
		APIVersion int `json:"apiVersion"`
		// The oldest app version still allowed to talk to this server ("" = any); apps older than this ask the user to update.
		MinAppVersion string         `json:"minAppVersion"`
		Features      statusFeatures `json:"features"`
	}
}

func registerSystem(api huma.API, d Deps) {
	huma.Register(api, huma.Operation{
		OperationID: "health", Method: http.MethodGet, Path: "/health",
		Summary: "Liveness probe", Tags: []string{"system"},
	}, func(ctx context.Context, _ *struct{}) (*healthOutput, error) {
		out := &healthOutput{}
		out.Body.OK = true
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "status", Method: http.MethodGet, Path: "/status",
		Summary: "Version and runtime info", Tags: []string{"system"},
	}, func(ctx context.Context, _ *struct{}) (*statusOutput, error) {
		out := &statusOutput{}
		out.Body.Version = Version
		out.Body.DB = d.Store.Dialect
		out.Body.DryRun = d.DryRun
		out.Body.UserFolderChoice = d.Requests.UsersMayChooseFolder(ctx)
		out.Body.DefaultLanguage, _ = d.Store.GetSetting(ctx, media.SettingDefaultLanguage)
		out.Body.APIVersion = APIVersion
		out.Body.MinAppVersion, _ = d.Store.GetSetting(ctx, SettingMinAppVersion)
		return out, nil
	})
}
