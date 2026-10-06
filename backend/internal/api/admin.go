package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/Rombond/tipsarr/backend/internal/auth"
	"github.com/Rombond/tipsarr/backend/internal/library"
	"github.com/danielgtaylor/huma/v2"
)

type settingsBody struct {
	JellyfinURL              string `json:"jellyfinUrl"`
	TMDBConfigured           bool   `json:"tmdbConfigured" doc:"Secrets are write-only; this only says whether a key is saved"`
	JellyfinAPIKeyConfigured bool   `json:"jellyfinApiKeyConfigured" doc:"Needed for library and history sync"`
	WebhookPath              string `json:"webhookPath" doc:"Path (with secret token) for the Jellyfin webhook plugin to call"`
	DryRun                   bool   `json:"dryRun" doc:"When true nothing is ever sent to Radarr/Sonarr"`
}

type settingsOutput struct{ Body settingsBody }

type updateSettingsInput struct {
	Body struct {
		TMDBKey        *string `json:"tmdbApiKey,omitempty" doc:"Set or replace the TMDB key (empty string clears it)"`
		JellyfinAPIKey *string `json:"jellyfinApiKey,omitempty" doc:"Jellyfin API key (Dashboard > API Keys); empty string clears it"`
	}
}

func registerAdmin(api huma.API, d Deps) {
	sec := []map[string][]string{{"session": {}}}

	read := func(ctx context.Context) (settingsBody, error) {
		url, err := d.Store.GetSetting(ctx, auth.SettingJellyfinURL)
		if err != nil {
			return settingsBody{}, err
		}
		key, err := d.Store.GetSetting(ctx, auth.SettingTMDBKey)
		if err != nil {
			return settingsBody{}, err
		}
		jfKey, err := d.Store.GetSetting(ctx, library.SettingJellyfinAPIKey)
		if err != nil {
			return settingsBody{}, err
		}
		secret, err := webhookSecret(ctx, d)
		if err != nil {
			return settingsBody{}, err
		}
		return settingsBody{
			JellyfinURL: url, TMDBConfigured: key != "", DryRun: d.DryRun,
			JellyfinAPIKeyConfigured: jfKey != "", WebhookPath: "/api/v1/hooks/jellyfin?token=" + secret,
		}, nil
	}

	huma.Register(api, huma.Operation{
		OperationID: "getSettings", Method: http.MethodGet, Path: "/admin/settings",
		Summary: "Read settings (admin)", Tags: []string{"admin"}, Security: sec,
		Errors: []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, _ *struct{}) (*settingsOutput, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		b, err := read(ctx)
		return &settingsOutput{Body: b}, err
	})

	huma.Register(api, huma.Operation{
		OperationID: "updateSettings", Method: http.MethodPut, Path: "/admin/settings",
		Summary: "Update settings (admin)", Tags: []string{"admin"}, Security: sec,
		Errors: []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, in *updateSettingsInput) (*settingsOutput, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		if in.Body.TMDBKey != nil {
			if err := d.Store.SetSetting(ctx, auth.SettingTMDBKey, strings.TrimSpace(*in.Body.TMDBKey)); err != nil {
				return nil, err
			}
		}
		if in.Body.JellyfinAPIKey != nil {
			if err := d.Store.SetSetting(ctx, library.SettingJellyfinAPIKey, strings.TrimSpace(*in.Body.JellyfinAPIKey)); err != nil {
				return nil, err
			}
		}
		b, err := read(ctx)
		return &settingsOutput{Body: b}, err
	})
}

const settingWebhookSecret = "hooks.jellyfin_secret"

// webhookSecret returns the shared secret for the Jellyfin webhook, generating it on first use.
func webhookSecret(ctx context.Context, d Deps) (string, error) {
	secret, err := d.Store.GetSetting(ctx, settingWebhookSecret)
	if err != nil || secret != "" {
		return secret, err
	}
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	secret = hex.EncodeToString(raw)
	return secret, d.Store.SetSetting(ctx, settingWebhookSecret, secret)
}
