package api

import (
	"context"
	"github.com/Rombond/tipsarr/backend/internal/clients/github"
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
	Push       bool `json:"push" doc:"Push notifications are set up on this server"`
	LiveFollow bool `json:"liveFollow,omitempty" doc:"Requests can be followed from the apps (Live Activity / ongoing notification). Absent on older servers"`
	Teasers    bool `json:"teasers" doc:"The Teasers feed is available: trailer files are on and the resolver answers. Apps hide the Teasers tab otherwise"`
}

type statusOutput struct {
	Body struct {
		Version       string `json:"version"`
		LatestVersion string `json:"latestVersion" doc:"Newest release tag published on GitHub (checked every 6 hours); empty while unknown, in the demo and for a dev build"`
		DB            string `json:"db" doc:"Database engine: sqlite, postgres or mysql"`
		DryRun        bool   `json:"dryRun" doc:"When true nothing is ever sent to Radarr/Sonarr"`
		Demo          bool   `json:"demo" doc:"This server is a throwaway public demo sandbox: one-click sign-in, actions that reach other services are refused (demo_disabled)"`
		// The app-wide default language (a TMDB tag such as fr-FR); empty means the browser decides.
		DefaultLanguage string `json:"defaultLanguage"`
		// APIVersion counts breaking changes of /api/v1: it only grows when an app must be updated.
		APIVersion int `json:"apiVersion"`
		// The oldest app version still allowed to talk to this server ("" = any); apps older than this ask the user to update.
		MinAppVersion string         `json:"minAppVersion"`
		Features      statusFeatures `json:"features"`
	}
}

var latestRelease = github.NewLatest(Version)

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
		if !d.Demo && Version != "dev" { // a local build never shows an update link, so it does not ask GitHub
			out.Body.LatestVersion = latestRelease.Get()
		}
		out.Body.DB = d.Store.Dialect
		out.Body.DryRun = d.DryRun
		out.Body.Demo = d.Demo
		out.Body.DefaultLanguage, _ = d.Store.GetSetting(ctx, media.SettingDefaultLanguage)
		out.Body.APIVersion = APIVersion
		out.Body.MinAppVersion, _ = d.Store.GetSetting(ctx, SettingMinAppVersion)
		out.Body.Features.Push = d.Push != nil && d.Push.Configured(ctx)
		out.Body.Features.LiveFollow = !d.Demo && d.Follow != nil && out.Body.Features.Push
		out.Body.Features.Teasers = d.Trailers.Available(ctx)
		return out, nil
	})
}
