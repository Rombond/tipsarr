package api

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/Rombond/tipsarr/backend/internal/notify"
	"github.com/Rombond/tipsarr/backend/internal/store"
	"github.com/danielgtaylor/huma/v2"
)

type webhookView struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	URL              string   `json:"url"`
	Events           []string `json:"events" doc:"Subscribed events; empty means all"`
	Enabled          bool     `json:"enabled"`
	SecretConfigured bool     `json:"secretConfigured" doc:"A signing secret is saved (HMAC-SHA256 in X-Tipsarr-Signature)"`
}

func toWebhookView(w store.Webhook) webhookView {
	ev := []string{}
	for _, e := range strings.Split(w.Events, ",") {
		if e = strings.TrimSpace(e); e != "" && e != "*" {
			ev = append(ev, e)
		}
	}
	return webhookView{ID: w.ID, Name: w.Name, URL: w.URL, Events: ev, Enabled: w.Enabled == 1, SecretConfigured: w.Secret != ""}
}

type webhookBody struct {
	Name    string   `json:"name" minLength:"1" maxLength:"100"`
	URL     string   `json:"url" format:"uri"`
	Events  []string `json:"events,omitempty" doc:"Any of request.created, request.approved, request.declined, request.failed, media.available; empty means all"`
	Enabled bool     `json:"enabled"`
	Secret  *string  `json:"secret,omitempty" doc:"Signing secret (write-only). Empty string clears it; omit to keep it"`
}

func registerWebhooks(api huma.API, d Deps) {
	sec := []map[string][]string{{"session": {}}, {"bearer": {}}}
	adminErrs := []int{http.StatusUnauthorized, http.StatusForbidden}

	check := func(b webhookBody) (string, error) {
		u, err := url.Parse(b.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return "", fail(422, "url_invalid", "url must be http(s)://…")
		}
		known := map[string]bool{}
		for _, e := range notify.AllEvents {
			known[e] = true
		}
		for _, e := range b.Events {
			if !known[e] {
				return "", fail(422, "unknown_event", "unknown event "+e)
			}
		}
		return strings.Join(b.Events, ","), nil
	}
	flag := func(b bool) int {
		if b {
			return 1
		}
		return 0
	}

	huma.Register(api, huma.Operation{
		OperationID: "listWebhooks", Method: http.MethodGet, Path: "/admin/webhooks",
		Summary: "Outgoing webhooks (admin)", Tags: []string{"admin"}, Security: sec, Errors: adminErrs,
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body []webhookView }, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		rows, err := d.Store.ListWebhooks(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]webhookView, 0, len(rows))
		for _, r := range rows {
			out = append(out, toWebhookView(r))
		}
		return &struct{ Body []webhookView }{out}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "createWebhook", Method: http.MethodPost, Path: "/admin/webhooks",
		Summary: "Add an outgoing webhook (admin)", Tags: []string{"admin"}, Security: sec,
		DefaultStatus: http.StatusCreated, Errors: append(adminErrs, http.StatusUnprocessableEntity),
	}, func(ctx context.Context, in *struct{ Body webhookBody }) (*struct{ Body webhookView }, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		events, err := check(in.Body)
		if err != nil {
			return nil, err
		}
		w := &store.Webhook{ID: store.NewID(), Name: in.Body.Name, URL: in.Body.URL, Events: events, Enabled: flag(in.Body.Enabled)}
		if in.Body.Secret != nil {
			w.Secret = *in.Body.Secret
		}
		if err := d.Store.SaveWebhook(ctx, w); err != nil {
			return nil, err
		}
		return &struct{ Body webhookView }{toWebhookView(*w)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "updateWebhook", Method: http.MethodPut, Path: "/admin/webhooks/{id}",
		Summary: "Update an outgoing webhook (admin)", Tags: []string{"admin"}, Security: sec,
		Errors: append(adminErrs, http.StatusNotFound, http.StatusUnprocessableEntity),
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id" maxLength:"32"`
		Body webhookBody
	}) (*struct{ Body webhookView }, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		events, err := check(in.Body)
		if err != nil {
			return nil, err
		}
		w, err := d.Store.GetWebhook(ctx, in.ID)
		if errors.Is(err, store.ErrNotFound) {
			return nil, fail(404, "not_found", "not found")
		}
		if err != nil {
			return nil, err
		}
		w.Name, w.URL, w.Events, w.Enabled = in.Body.Name, in.Body.URL, events, flag(in.Body.Enabled)
		if in.Body.Secret != nil {
			w.Secret = *in.Body.Secret
		}
		if err := d.Store.SaveWebhook(ctx, w); err != nil {
			return nil, err
		}
		return &struct{ Body webhookView }{toWebhookView(*w)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "deleteWebhook", Method: http.MethodDelete, Path: "/admin/webhooks/{id}",
		Summary: "Delete an outgoing webhook (admin)", Tags: []string{"admin"}, Security: sec,
		DefaultStatus: http.StatusNoContent, Errors: adminErrs,
	}, func(ctx context.Context, in *struct {
		ID string `path:"id" maxLength:"32"`
	}) (*struct{}, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		return &struct{}{}, d.Store.DeleteWebhook(ctx, in.ID)
	})

	huma.Register(api, huma.Operation{
		OperationID: "testWebhook", Method: http.MethodPost, Path: "/admin/webhooks/{id}/test",
		Summary: "Send a test event to a webhook (admin)", Tags: []string{"admin"}, Security: sec,
		DefaultStatus: http.StatusNoContent, Errors: append(adminErrs, http.StatusNotFound, http.StatusBadGateway),
	}, func(ctx context.Context, in *struct {
		ID string `path:"id" maxLength:"32"`
	}) (*struct{}, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		w, err := d.Store.GetWebhook(ctx, in.ID)
		if errors.Is(err, store.ErrNotFound) {
			return nil, fail(404, "not_found", "not found")
		}
		if err != nil {
			return nil, err
		}
		if err := d.Notify.Deliver(ctx, *w, "test", map[string]string{"message": "Tipsarr test event"}); err != nil {
			return nil, fail(502, "delivery_failed", "delivery failed: "+err.Error())
		}
		return &struct{}{}, nil
	})
}
