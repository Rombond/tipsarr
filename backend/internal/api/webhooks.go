package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strings"

	"github.com/Rombond/tipsarr/backend/internal/netguard"
	"github.com/Rombond/tipsarr/backend/internal/notify"
	"github.com/Rombond/tipsarr/backend/internal/perms"
	"github.com/Rombond/tipsarr/backend/internal/store"
	"github.com/danielgtaylor/huma/v2"
)

type webhookView struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	URL              string          `json:"url" doc:"May contain Liquid ({{ ... }})"`
	Events           []string        `json:"events" doc:"Subscribed events; empty means all"`
	Enabled          bool            `json:"enabled"`
	SecretConfigured bool            `json:"secretConfigured" doc:"A signing secret is saved (HMAC-SHA256 in X-Tipsarr-Signature)"`
	Preset           string          `json:"preset" doc:"Preset the webhook was made from (informative)"`
	Template         string          `json:"template" doc:"Liquid body template; empty = the raw JSON envelope"`
	Headers          []notify.Header `json:"headers"`
	AuthType         string          `json:"authType" enum:"none,bearer,basic"`
	AuthUsername     string          `json:"authUsername"`
	AuthConfigured   bool            `json:"authSecretConfigured" doc:"A bearer token or basic password is saved (write-only)"`
	PerRecipient     bool            `json:"perRecipient" doc:"One delivery per concerned person, with notifyUser filled"`
	BlockedByPolicy  bool            `json:"blockedByPolicy" doc:"The URL points at a private or local address the outbound policy refuses (see /admin/webhooks/egress)"`
}

func toWebhookView(w store.Webhook) webhookView {
	ev := []string{}
	for _, e := range strings.Split(w.Events, ",") {
		if e = strings.TrimSpace(e); e != "" && e != "*" {
			ev = append(ev, e)
		}
	}
	hd := notify.ParseHeaders(w.Headers)
	if hd == nil {
		hd = []notify.Header{}
	}
	auth := w.AuthType
	if auth == "" {
		auth = "none"
	}
	return webhookView{ID: w.ID, Name: w.Name, URL: w.URL, Events: ev, Enabled: w.Enabled == 1, SecretConfigured: w.Secret != "",
		Preset: w.Preset, Template: w.Template, Headers: hd, AuthType: auth, AuthUsername: w.AuthUsername, AuthConfigured: w.AuthSecret != "", PerRecipient: w.PerRecipient == 1}
}

type webhookBody struct {
	Name         string          `json:"name" minLength:"1" maxLength:"100"`
	URL          string          `json:"url" minLength:"1" maxLength:"512" doc:"http(s) URL; may contain Liquid, rendered per delivery"`
	Events       []string        `json:"events,omitempty" doc:"Any of request.created, request.approved, request.auto_approved, request.declined, request.failed, media.available, issue.created, issue.commented, issue.resolved, issue.reopened; empty means all. request.approved also receives request.auto_approved"`
	Enabled      bool            `json:"enabled"`
	Secret       *string         `json:"secret,omitempty" doc:"Signing secret (write-only). Empty string clears it; omit to keep it"`
	Preset       string          `json:"preset,omitempty" maxLength:"32"`
	Template     string          `json:"template,omitempty" maxLength:"20000" doc:"Liquid body template; empty = the raw JSON envelope"`
	Headers      []notify.Header `json:"headers,omitempty" maxItems:"20"`
	AuthType     string          `json:"authType,omitempty" enum:"none,bearer,basic,"`
	AuthUsername string          `json:"authUsername,omitempty" maxLength:"191"`
	AuthSecret   *string         `json:"authSecret,omitempty" maxLength:"512" doc:"Bearer token or basic password (write-only). Empty string clears it; omit to keep it"`
	PerRecipient bool            `json:"perRecipient,omitempty"`
}

// apply copies the form onto a stored webhook.
func (b webhookBody) apply(w *store.Webhook, events string) {
	w.Name, w.URL, w.Events, w.Preset, w.Template = b.Name, strings.TrimSpace(b.URL), events, b.Preset, b.Template
	w.Enabled, w.PerRecipient = flag(b.Enabled), flag(b.PerRecipient)
	w.AuthType, w.AuthUsername = b.AuthType, b.AuthUsername
	if w.AuthType == "none" {
		w.AuthType = ""
	}
	hd := []notify.Header{}
	for _, h := range b.Headers {
		if h.Name = strings.TrimSpace(h.Name); h.Name != "" {
			hd = append(hd, h)
		}
	}
	raw, _ := json.Marshal(hd)
	w.Headers = string(raw)
	if b.Secret != nil {
		w.Secret = *b.Secret
	}
	if b.AuthSecret != nil {
		w.AuthSecret = *b.AuthSecret
	}
}

func flag(b bool) int {
	if b {
		return 1
	}
	return 0
}

type webhookPreview struct {
	URL       string          `json:"url"`
	Headers   []notify.Header `json:"headers" doc:"Secrets are masked"`
	Body      string          `json:"body"`
	ValidJSON bool            `json:"validJson"`
	Error     string          `json:"error,omitempty" doc:"Template or URL problem; nothing else is set then"`
	Delivered *bool           `json:"delivered,omitempty" doc:"Test only: whether the receiver accepted it"`
	Status    string          `json:"deliveryError,omitempty" doc:"Test only: why the delivery failed"`
}

// sampleEvent is the event used by the test button and the preview: a made-up request by the caller.
func sampleEvent(name string, me *store.User) notify.Event {
	if name == "" {
		name = notify.Test
	}
	ev := notify.Event{Name: name, RequestedBy: me.ID, Users: []string{me.ID},
		Media:   &notify.Media{TMDBID: 438631, IMDBID: "tt1160419", Type: "movie", Title: "Dune", Year: "2021", Status: "pending", PosterPath: "/d5NXSklXo0qyIYkgV94XAgMIckC.jpg"},
		Request: map[string]any{"id": "sample", "type": "movie", "tmdbId": 438631, "title": "Dune", "status": "pending", "requestedBy": map[string]any{"id": me.ID, "name": me.Name}},
		Data:    map[string]any{"message": "Tipsarr test event"}}
	if strings.HasPrefix(name, "issue.") {
		ev.IssueID = "sample"
		ev.Issue = map[string]any{"id": "sample", "type": "movie", "tmdbId": 438631, "title": "Dune", "kind": "audio", "status": "open"}
		ev.Comment = &notify.Comment{Message: "The audio is \"out of sync\"\nafter 20 minutes.", UserID: me.ID, Name: me.Name}
	}
	return ev
}

func preview(m *notify.Message) webhookPreview {
	out := webhookPreview{URL: m.URL, Body: string(m.Body), ValidJSON: json.Valid(m.Body), Headers: []notify.Header{}}
	for k, vs := range m.Headers {
		v := strings.Join(vs, ", ")
		if k == "Authorization" || k == "X-Tipsarr-Signature" {
			v = "••••"
		}
		out.Headers = append(out.Headers, notify.Header{Name: k, Value: v})
	}
	sort.Slice(out.Headers, func(i, j int) bool { return out.Headers[i].Name < out.Headers[j].Name })
	return out
}

func renderError(err error) string {
	var te *notify.TemplateError
	if errors.As(err, &te) || errors.Is(err, notify.ErrServarrTarget) {
		return err.Error()
	}
	return ""
}

type webhookRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type webhookEgress struct {
	AllowedHosts []string     `json:"allowedHosts" doc:"Private targets webhooks may reach: host, host:port or CIDR"`
	Blocked      []webhookRef `json:"blocked" doc:"Saved webhooks whose URL the policy refuses now"`
}

func registerWebhooks(api huma.API, d Deps) {
	sec := []map[string][]string{{"session": {}}, {"bearer": {}}}
	adminErrs := []int{http.StatusUnauthorized, http.StatusForbidden}

	check := func(b webhookBody) (string, error) {
		raw := strings.TrimSpace(b.URL)
		if strings.Contains(raw, "{{") || strings.Contains(raw, "{%") {
			if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
				return "", fail(422, "url_invalid", "url must start with http(s)://")
			}
			if err := notify.CheckTemplate("url", raw); err != nil {
				return "", fail(422, "template_invalid", err.Error())
			}
		} else if u, err := url.Parse(raw); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return "", fail(422, "url_invalid", "url must be http(s)://…")
		}
		if err := notify.CheckTemplate("body", b.Template); err != nil {
			return "", fail(422, "template_invalid", err.Error())
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

	huma.Register(api, huma.Operation{
		OperationID: "listWebhooks", Method: http.MethodGet, Path: "/admin/webhooks",
		Summary: "Outgoing webhooks (admin)", Tags: []string{"admin"}, Security: sec, Errors: adminErrs,
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body []webhookView }, error) {
		if _, err := requireCan(ctx, perms.ManageSettings); err != nil {
			return nil, err
		}
		rows, err := d.Store.ListWebhooks(ctx)
		if err != nil {
			return nil, err
		}
		blocked := map[string]bool{}
		for _, b := range d.Notify.BlockedWebhooks(ctx) {
			blocked[b.ID] = true
		}
		out := make([]webhookView, 0, len(rows))
		for _, r := range rows {
			v := toWebhookView(r)
			v.BlockedByPolicy = blocked[r.ID]
			out = append(out, v)
		}
		return &struct{ Body []webhookView }{out}, nil
	})

	egressOut := func(ctx context.Context) (*struct{ Body webhookEgress }, error) {
		v, err := d.Store.GetSetting(ctx, notify.SettingAllowedHosts)
		if err != nil {
			return nil, err
		}
		out := webhookEgress{AllowedHosts: notify.SplitAllowList(v), Blocked: []webhookRef{}}
		if out.AllowedHosts == nil {
			out.AllowedHosts = []string{}
		}
		for _, b := range d.Notify.BlockedWebhooks(ctx) {
			out.Blocked = append(out.Blocked, webhookRef{ID: b.ID, Name: b.Name})
		}
		return &struct{ Body webhookEgress }{out}, nil
	}
	huma.Register(api, huma.Operation{
		OperationID: "getWebhookEgress", Method: http.MethodGet, Path: "/admin/webhooks/egress",
		Summary:     "Outbound policy of webhooks (admin)",
		Description: "Webhooks may only reach public addresses, plus the private ones listed here. Loopback and link-local addresses are always refused.",
		Tags:        []string{"admin"}, Security: sec, Errors: adminErrs,
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body webhookEgress }, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		return egressOut(ctx)
	})
	huma.Register(api, huma.Operation{
		OperationID: "updateWebhookEgress", Method: http.MethodPut, Path: "/admin/webhooks/egress",
		Summary: "Set the private hosts webhooks may reach (admin)", Tags: []string{"admin"}, Security: sec,
		Errors: append(adminErrs, http.StatusUnprocessableEntity),
	}, func(ctx context.Context, in *struct {
		Body struct {
			AllowedHosts []string `json:"allowedHosts" maxItems:"50" doc:"host, host:port or CIDR, e.g. ntfy.lan, 192.168.1.20:8123, 10.0.0.0/24"`
		}
	}) (*struct{ Body webhookEgress }, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		var keep []string
		for _, h := range in.Body.AllowedHosts {
			if h = strings.TrimSpace(h); h == "" {
				continue
			}
			if !netguard.ValidAllowEntry(h) {
				return nil, fail(422, "bad_allowed_host", "not a host, host:port or CIDR: "+h)
			}
			keep = append(keep, h)
		}
		if err := d.Store.SetSetting(ctx, notify.SettingAllowedHosts, strings.Join(keep, "\n")); err != nil {
			return nil, err
		}
		return egressOut(ctx)
	})

	huma.Register(api, huma.Operation{
		OperationID: "listWebhookPresets", Method: http.MethodGet, Path: "/admin/webhooks/presets",
		Summary: "Ready-made webhook templates and the events (admin)", Tags: []string{"admin"}, Security: sec, Errors: adminErrs,
	}, func(ctx context.Context, _ *struct{}) (*struct {
		Body struct {
			Presets []notify.Preset `json:"presets"`
			Events  []string        `json:"events" doc:"Events a webhook may subscribe to"`
		}
	}, error) {
		if _, err := requireCan(ctx, perms.ManageSettings); err != nil {
			return nil, err
		}
		out := &struct {
			Body struct {
				Presets []notify.Preset `json:"presets"`
				Events  []string        `json:"events" doc:"Events a webhook may subscribe to"`
			}
		}{}
		out.Body.Presets, out.Body.Events = notify.Presets, notify.AllEvents
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "createWebhook", Method: http.MethodPost, Path: "/admin/webhooks",
		Summary: "Add an outgoing webhook (admin)", Tags: []string{"admin"}, Security: sec,
		DefaultStatus: http.StatusCreated, Errors: append(adminErrs, http.StatusUnprocessableEntity),
	}, func(ctx context.Context, in *struct{ Body webhookBody }) (*struct{ Body webhookView }, error) {
		if _, err := requireCan(ctx, perms.ManageSettings); err != nil {
			return nil, err
		}
		events, err := check(in.Body)
		if err != nil {
			return nil, err
		}
		w := &store.Webhook{ID: store.NewID()}
		in.Body.apply(w, events)
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
		if _, err := requireCan(ctx, perms.ManageSettings); err != nil {
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
		in.Body.apply(w, events)
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
		if _, err := requireCan(ctx, perms.ManageSettings); err != nil {
			return nil, err
		}
		return &struct{}{}, d.Store.DeleteWebhook(ctx, in.ID)
	})

	huma.Register(api, huma.Operation{
		OperationID: "listWebhookVariables", Method: http.MethodGet, Path: "/admin/webhooks/variables",
		Summary:     "Template variables of an event, with sample values (admin)",
		Description: "Built from the same variables as real deliveries, with the sample data of the preview. empty marks what the event does not fill; perRecipient what only one delivery per recipient fills.",
		Tags:        []string{"admin"}, Security: sec, Errors: append(adminErrs, http.StatusUnprocessableEntity),
	}, func(ctx context.Context, in *struct {
		Event string `query:"event" maxLength:"40" doc:"Event (default test)"`
		Lang  string `query:"lang" maxLength:"16" doc:"Language of the descriptions (default: the caller's)"`
	}) (*struct{ Body []notify.Variable }, error) {
		me, err := requireCan(ctx, perms.ManageSettings)
		if err != nil {
			return nil, err
		}
		if in.Event != "" && in.Event != notify.Test && !slices.Contains(notify.AllEvents, in.Event) {
			return nil, fail(422, "unknown_event", "unknown event "+in.Event)
		}
		lang := in.Lang
		if lang == "" {
			lang = me.Language
		}
		all := d.Notify.Vars(ctx, sampleEvent(notify.IssueCreated, me), me)
		vars := d.Notify.Vars(ctx, sampleEvent(in.Event, me), me) // notifyUser filled, flagged perRecipient
		return &struct{ Body []notify.Variable }{notify.Variables(all, vars, lang)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "previewWebhook", Method: http.MethodPost, Path: "/admin/webhooks/preview",
		Summary:     "Render a webhook form with sample values, without sending (admin)",
		Description: "Renders the URL, headers and body for a made-up event. With id, saved secrets are used for what the form leaves out.",
		Tags:        []string{"admin"}, Security: sec, Errors: append(adminErrs, http.StatusUnprocessableEntity),
	}, func(ctx context.Context, in *struct {
		ID    string `query:"id" maxLength:"32" doc:"Saved webhook whose secrets fill what the form omits"`
		Event string `query:"event" maxLength:"40" doc:"Event to simulate (default test)"`
		Body  webhookBody
	}) (*struct{ Body webhookPreview }, error) {
		me, err := requireCan(ctx, perms.ManageSettings)
		if err != nil {
			return nil, err
		}
		w := &store.Webhook{}
		if in.ID != "" {
			if saved, err := d.Store.GetWebhook(ctx, in.ID); err == nil {
				w = saved
			}
		}
		in.Body.apply(w, "")
		if err := notify.CheckTemplate("body", w.Template); err != nil {
			return &struct{ Body webhookPreview }{webhookPreview{Error: err.Error(), Headers: []notify.Header{}}}, nil
		}
		m, err := d.Notify.Render(ctx, *w, sampleEvent(in.Event, me), me)
		if err != nil {
			if msg := renderError(err); msg != "" {
				return &struct{ Body webhookPreview }{webhookPreview{Error: msg, Headers: []notify.Header{}}}, nil
			}
			return nil, err
		}
		return &struct{ Body webhookPreview }{preview(m)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "testWebhook", Method: http.MethodPost, Path: "/admin/webhooks/{id}/test",
		Summary:     "Send a test event to a webhook (admin)",
		Description: "204 when delivered. 422 template_error when the template does not render (the message says where), 502 when the receiver fails.",
		Tags:        []string{"admin"}, Security: sec,
		DefaultStatus: http.StatusNoContent, Errors: append(adminErrs, http.StatusNotFound, http.StatusUnprocessableEntity, http.StatusBadGateway),
	}, func(ctx context.Context, in *struct {
		ID string `path:"id" maxLength:"32"`
	}) (*struct{}, error) {
		me, err := requireCan(ctx, perms.ManageSettings)
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
		var to *store.User
		if w.PerRecipient == 1 {
			to = me
		}
		m, err := d.Notify.Render(ctx, *w, sampleEvent(notify.Test, me), to)
		if errors.Is(err, notify.ErrServarrTarget) {
			return nil, fail(502, "delivery_failed", "delivery failed: "+notify.FailureText(err))
		}
		if err != nil {
			if msg := renderError(err); msg != "" {
				return nil, fail(422, "template_error", msg)
			}
			return nil, err
		}
		if err := d.Notify.Post(ctx, m); err != nil {
			slog.Info("webhook test failed", "webhook", w.Name, "err", err) // details stay in the log
			return nil, fail(502, "delivery_failed", "delivery failed: "+notify.FailureText(err))
		}
		return &struct{}{}, nil
	})
}
