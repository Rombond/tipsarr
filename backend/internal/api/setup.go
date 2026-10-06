package api

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Rombond/tipsarr/backend/internal/auth"
	"github.com/Rombond/tipsarr/backend/internal/clients/jellyfin"
	"github.com/danielgtaylor/huma/v2"
)

type setupStatusOutput struct {
	Body struct {
		Configured    bool `json:"configured" doc:"True once a Jellyfin URL is saved"`
		TokenRequired bool `json:"tokenRequired" doc:"The setup call needs the token printed in the server log (TIPSARR_SETUP_TOKEN=true)"`
	}
}

type setupInput struct {
	Body struct {
		SetupToken  string `json:"setupToken,omitempty" doc:"One-time token printed in the server log at startup (required on a fresh install)"`
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
		out.Body.TokenRequired = url == "" && d.SetupToken != ""
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "setup", Method: http.MethodPost, Path: "/setup",
		Summary:     "First-run setup: point Tipsarr at Jellyfin",
		Description: "Only allowed until a Jellyfin URL is saved, and, when TIPSARR_SETUP_TOKEN=true, only with the setup token printed in the server log. Afterwards the first Jellyfin administrator to log in becomes the Tipsarr admin.",
		Tags:        []string{"setup"}, Errors: []int{http.StatusUnauthorized, http.StatusConflict, http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *setupInput) (*setupOutput, error) {
		current, err := d.Store.GetSetting(ctx, auth.SettingJellyfinURL)
		if err != nil {
			return nil, err
		}
		if current != "" {
			return nil, fail(409, "setup_done", "setup already completed")
		}
		// Without the token anyone who reaches the port first could point Tipsarr at their own
		// "Jellyfin" and become admin. The token is generated at startup and only printed to the log.
		if d.SetupToken != "" && subtle.ConstantTimeCompare([]byte(in.Body.SetupToken), []byte(d.SetupToken)) != 1 {
			return nil, fail(401, "setup_token", "setup token required: see the Tipsarr server log")
		}
		url := strings.TrimRight(in.Body.JellyfinURL, "/")
		info, err := jellyfin.New(url).PublicInfo(ctx)
		if err != nil {
			slog.Warn("setup: cannot reach Jellyfin", "err", err)
			return nil, fail(422, "setup_unreachable", "cannot reach a Jellyfin server at that URL")
		}
		// claim atomically: of two concurrent setups only one wins
		if ok, err := d.Store.SetSettingIfAbsent(ctx, auth.SettingJellyfinURL, url); err != nil {
			return nil, err
		} else if !ok {
			return nil, fail(409, "setup_done", "setup already completed")
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
