package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// Secrets the admin UI may need to show (GET /admin/settings only says whether they exist).
const (
	secretJellyfinWebhook = "jellyfin_webhook"
	secretAPIKey          = "api_key"
)

type secretInput struct {
	Name string `path:"name" enum:"jellyfin_webhook,api_key"`
}

type secretOutput struct {
	Body struct {
		Value string `json:"value" doc:"The secret; empty when it is not set (the dashboard key is off)"`
	}
}

func registerSecrets(api huma.API, d Deps) {
	sec := []map[string][]string{{"session": {}}, {"bearer": {}}}
	errs := []int{http.StatusUnauthorized, http.StatusForbidden}

	read := func(ctx context.Context, name string) (string, error) {
		if name == secretJellyfinWebhook {
			return webhookSecret(ctx, d)
		}
		return d.Store.GetSetting(ctx, settingAPIKey)
	}

	huma.Register(api, huma.Operation{
		OperationID: "revealSecret", Method: http.MethodPost, Path: "/admin/secrets/{name}/reveal",
		Summary:     "Show a secret (admin role only)",
		Description: "The Jellyfin webhook secret (X-Tipsarr-Token) or the dashboard key. POST so it is never cached nor read with the read-only key.",
		Tags:        []string{"admin"}, Security: sec, Errors: errs,
	}, func(ctx context.Context, in *secretInput) (*secretOutput, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		v, err := read(ctx, in.Name)
		if err != nil {
			return nil, err
		}
		out := &secretOutput{}
		out.Body.Value = v
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "regenerateSecret", Method: http.MethodPost, Path: "/admin/secrets/{name}/regenerate",
		Summary:     "Replace a secret with a new one and show it (admin role only)",
		Description: "The old value stops working at once: update the Jellyfin webhook plugin or the dashboards. For api_key this also turns the key on.",
		Tags:        []string{"admin"}, Security: sec, Errors: errs,
	}, func(ctx context.Context, in *secretInput) (*secretOutput, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		if in.Name == secretJellyfinWebhook {
			if err := d.Store.SetSetting(ctx, settingWebhookSecret, ""); err != nil {
				return nil, err
			}
		} else if err := updateAPIKey(ctx, d, ptr(true), ptr(true)); err != nil {
			return nil, err
		}
		v, err := read(ctx, in.Name)
		if err != nil {
			return nil, err
		}
		out := &secretOutput{}
		out.Body.Value = v
		return out, nil
	})
}
