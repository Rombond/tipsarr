package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/perms"
	"github.com/Rombond/tipsarr/backend/internal/trailers"
	"github.com/danielgtaylor/huma/v2"
)

func errTeasersDisabled() error {
	return fail(404, "teasers_disabled", "trailer files are not set up on this server")
}

// writeCodedError answers a raw (non-huma) handler with the same JSON shape as fail(): the code sits in errors[0].value.
func writeCodedError(w http.ResponseWriter, status int, code, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"title": http.StatusText(status), "status": status, "detail": detail,
		"errors": []map[string]any{{"message": detail, "location": "code", "value": code}},
	})
}

// teaserVideoHandler serves a finished trailer file with Range support (apps stream it with a native player).
// The key is validated against the YouTube key pattern, so no path can leave the shared folder.
func teaserVideoHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if userFrom(r.Context()) == nil {
			writeCodedError(w, http.StatusUnauthorized, "login_required", "login required")
			return
		}
		if !d.Trailers.Enabled(r.Context()) {
			writeCodedError(w, http.StatusNotFound, "teasers_disabled", "trailer files are not set up on this server")
			return
		}
		key := chiParam(r, "key")
		row, ok := d.Trailers.Ready(r.Context(), key)
		if !ok {
			writeCodedError(w, http.StatusNotFound, "trailer_not_ready", "this trailer file is not ready")
			return
		}
		f, err := os.Open(d.Trailers.Path(key))
		if err != nil {
			writeCodedError(w, http.StatusNotFound, "trailer_not_ready", "this trailer file is not ready")
			return
		}
		defer f.Close()
		fi, err := f.Stat()
		if err != nil || !fi.Mode().IsRegular() {
			writeCodedError(w, http.StatusNotFound, "trailer_not_ready", "this trailer file is not ready")
			return
		}
		h := w.Header()
		h.Set("Content-Type", "video/mp4")
		h.Set("Cache-Control", "private, max-age=86400")
		h.Set("ETag", fmt.Sprintf(`"%s-%d-%d"`, key, fi.Size(), fi.ModTime().UnixNano()))
		h.Set("Accept-Ranges", "bytes")
		d.Trailers.Touch(r.Context(), row)
		http.ServeContent(w, r, key+".mp4", fi.ModTime(), f)
	}
}

type teaserFilesOverview struct{ Body *trailers.Overview }

func registerTeaserFiles(api huma.API, d Deps) {
	sec := []map[string][]string{{"session": {}}, {"bearer": {}}}

	huma.Register(api, huma.Operation{
		OperationID: "teaserFilesOverview", Method: http.MethodGet, Path: "/admin/teasers/files",
		Summary:     "Trailer files: settings state, resolver health and file counts (admin)",
		Description: "The Teasers feed plays trailers from files fetched by the optional trailer-resolver sidecar. The resolver is asked for its health now.",
		Tags:        []string{"admin"}, Security: sec,
		Errors: []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, _ *struct{}) (*teaserFilesOverview, error) {
		if _, err := requireCan(ctx, perms.ManageSettings); err != nil {
			return nil, err
		}
		o, err := d.Trailers.Overview(ctx)
		return &teaserFilesOverview{Body: o}, err
	})

	huma.Register(api, huma.Operation{
		OperationID: "teaserFilesTest", Method: http.MethodPost, Path: "/admin/teasers/files/test",
		Summary:     "Resolve one video through the resolver and report time and size (admin)",
		Description: "Uses the saved resolver address and key even while the feature is off. Waits up to 90 seconds; if the video is not ready by then the state is queued or running and the test can be repeated.",
		Tags:        []string{"admin"}, Security: sec,
		Errors: []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusUnprocessableEntity, http.StatusBadGateway},
	}, func(ctx context.Context, in *struct {
		Body struct {
			Key string `json:"key,omitempty" doc:"YouTube video key; default is a short public video"`
		}
	}) (*struct{ Body *trailers.TestResult }, error) {
		if _, err := requireCan(ctx, perms.ManageSettings); err != nil {
			return nil, err
		}
		res, err := d.Trailers.Test(ctx, strings.TrimSpace(in.Body.Key), 90*time.Second)
		if err != nil {
			return nil, trailerErr(err)
		}
		return &struct{ Body *trailers.TestResult }{res}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "teaserFilesClear", Method: http.MethodDelete, Path: "/admin/teasers/files",
		Summary: "Empty the resolver's cache and forget every file (admin)",
		Tags:    []string{"admin"}, Security: sec,
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusUnprocessableEntity, http.StatusBadGateway},
	}, func(ctx context.Context, _ *struct{}) (*struct{}, error) {
		if _, err := requireCan(ctx, perms.ManageSettings); err != nil {
			return nil, err
		}
		if err := d.Trailers.Clear(ctx); err != nil {
			return nil, trailerErr(err)
		}
		return &struct{}{}, nil
	})
}

// trailerErr maps the trailers package errors to API errors.
func trailerErr(err error) error {
	switch {
	case errors.Is(err, trailers.ErrBadKey):
		return fail(422, "trailer_key_invalid", "that is not a YouTube video key")
	case errors.Is(err, trailers.ErrDisabled):
		return fail(422, "trailer_resolver_not_configured", "save the resolver address and key first")
	case errors.Is(err, trailers.ErrUnauthorized):
		return fail(502, "trailer_resolver_refused", err.Error())
	case errors.Is(err, trailers.ErrUnreachable):
		return fail(502, "trailer_resolver_unreachable", err.Error())
	}
	return fail(502, "trailer_resolver_error", err.Error())
}

// updateTeaserFiles saves the trailer-file settings. They cannot be on without an address and a key.
func updateTeaserFiles(ctx context.Context, d Deps, enabled *bool, resolverURL, resolverKey *string, maxHeight *int) error {
	set := func(k, v string) error { return d.Store.SetSetting(ctx, k, v) }
	if resolverURL != nil {
		v := strings.TrimRight(strings.TrimSpace(*resolverURL), "/")
		if v != "" {
			if err := trailers.ValidURL(v); err != nil {
				return fail(422, "invalid_trailer_resolver_url", err.Error())
			}
		}
		if err := set(trailers.SettingURL, v); err != nil {
			return err
		}
		if v == "" {
			enabled = ptr(false)
		}
	}
	if resolverKey != nil {
		v := strings.TrimSpace(*resolverKey)
		if err := set(trailers.SettingKey, v); err != nil {
			return err
		}
		if v == "" {
			enabled = ptr(false)
		}
	}
	if maxHeight != nil {
		ok := false
		for _, h := range trailers.MaxHeights {
			ok = ok || h == *maxHeight
		}
		if !ok {
			return fail(422, "invalid_trailer_height", fmt.Sprintf("the maximum height must be one of %v", trailers.MaxHeights))
		}
		if err := set(trailers.SettingMaxHeight, strconv.Itoa(*maxHeight)); err != nil {
			return err
		}
	}
	if enabled != nil {
		if *enabled {
			u, _ := d.Store.GetSetting(ctx, trailers.SettingURL)
			k, _ := d.Store.GetSetting(ctx, trailers.SettingKey)
			if u == "" || k == "" {
				return fail(422, "trailer_resolver_not_configured", "set the resolver address and key before turning trailer files on")
			}
		}
		v := "false"
		if *enabled {
			v = "true"
		}
		if err := set(trailers.SettingEnabled, v); err != nil {
			return err
		}
	}
	d.Trailers.ResetHealth()
	return nil
}
