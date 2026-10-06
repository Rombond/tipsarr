package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/Rombond/tipsarr/backend/internal/auth"
	"github.com/Rombond/tipsarr/backend/internal/clients/jellyfin"
	"github.com/danielgtaylor/huma/v2"
)

type setupStatusOutput struct {
	Body struct {
		Configured bool `json:"configured" doc:"True once a Jellyfin URL is saved"`
	}
}

type setupInput struct {
	Body struct {
		JellyfinURL string `json:"jellyfinUrl" format:"uri" doc:"Base URL of the Jellyfin server"`
		TMDBKey     string `json:"tmdbApiKey,omitempty" doc:"TMDB API key or read token (can be added later)"`
	}
}

type setupOutput struct {
	Body struct {
		ServerName string `json:"serverName"`
	}
}

func registerSetup(api huma.API, d Deps) {
	huma.Register(api, huma.Operation{
		OperationID: "setupStatus", Method: http.MethodGet, Path: "/setup/status",
		Summary: "Has first-run setup been completed?", Tags: []string{"setup"},
	}, func(ctx context.Context, _ *struct{}) (*setupStatusOutput, error) {
		url, err := d.Store.GetSetting(ctx, auth.SettingJellyfinURL)
		if err != nil {
			return nil, err
		}
		out := &setupStatusOutput{}
		out.Body.Configured = url != ""
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "setup", Method: http.MethodPost, Path: "/setup",
		Summary:     "First-run setup: point Tipsarr at Jellyfin",
		Description: "Only allowed until a Jellyfin URL is saved. Afterwards the first Jellyfin administrator to log in becomes the Tipsarr admin.",
		Tags:        []string{"setup"}, Errors: []int{http.StatusConflict, http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *setupInput) (*setupOutput, error) {
		current, err := d.Store.GetSetting(ctx, auth.SettingJellyfinURL)
		if err != nil {
			return nil, err
		}
		if current != "" {
			return nil, huma.Error409Conflict("setup already completed")
		}
		url := strings.TrimRight(in.Body.JellyfinURL, "/")
		info, err := jellyfin.New(url).PublicInfo(ctx)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("cannot reach Jellyfin: " + err.Error())
		}
		if err := d.Store.SetSetting(ctx, auth.SettingJellyfinURL, url); err != nil {
			return nil, err
		}
		if in.Body.TMDBKey != "" {
			if err := d.Store.SetSetting(ctx, auth.SettingTMDBKey, in.Body.TMDBKey); err != nil {
				return nil, err
			}
		}
		out := &setupOutput{}
		out.Body.ServerName = info.ServerName
		return out, nil
	})
}
