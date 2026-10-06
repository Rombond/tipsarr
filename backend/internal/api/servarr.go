package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Rombond/tipsarr/backend/internal/clients/servarr"
	"github.com/Rombond/tipsarr/backend/internal/store"
	"github.com/danielgtaylor/huma/v2"
)

type instanceView struct {
	ID               string            `json:"id"`
	Kind             string            `json:"kind" enum:"radarr,sonarr"`
	Name             string            `json:"name"`
	URL              string            `json:"url"`
	APIKeyConfigured bool              `json:"apiKeyConfigured"`
	QualityProfileID int               `json:"qualityProfileId"`
	RootFolder       string            `json:"rootFolder"`
	IsDefault        bool              `json:"isDefault"`
	GenreRoots       map[string]string `json:"genreRoots" doc:"Radarr only: TMDB genre id -> root folder; the first matching genre of a movie wins"`
}

func toInstanceView(in store.ServarrInstance) instanceView {
	roots := map[string]string{}
	_ = json.Unmarshal([]byte(in.GenreRoots), &roots)
	return instanceView{ID: in.ID, Kind: in.Kind, Name: in.Name, URL: in.URL, APIKeyConfigured: in.APIKey != "",
		QualityProfileID: in.QualityProfileID, RootFolder: in.RootFolder, IsDefault: in.IsDefault == 1, GenreRoots: roots}
}

type instanceBody struct {
	Kind             string            `json:"kind" enum:"radarr,sonarr"`
	Name             string            `json:"name" minLength:"1" maxLength:"100"`
	URL              string            `json:"url" format:"uri"`
	APIKey           string            `json:"apiKey,omitempty" doc:"Required when creating; omit on update to keep the saved key"`
	QualityProfileID int               `json:"qualityProfileId" minimum:"1"`
	RootFolder       string            `json:"rootFolder" minLength:"1"`
	IsDefault        bool              `json:"isDefault"`
	GenreRoots       map[string]string `json:"genreRoots,omitempty" doc:"Radarr only. Omit to keep the saved map, send {} to clear it"`
}

type probeBody struct {
	Kind   string `json:"kind" enum:"radarr,sonarr"`
	URL    string `json:"url" format:"uri"`
	APIKey string `json:"apiKey,omitempty" doc:"Leave empty to use the saved key of the instance given by id"`
	ID     string `json:"id,omitempty"`
}

type probeResult struct {
	AppName     string                   `json:"appName"`
	Version     string                   `json:"version"`
	Profiles    []servarr.QualityProfile `json:"profiles"`
	RootFolders []servarr.RootFolder     `json:"rootFolders"`
}

func registerServarr(api huma.API, d Deps) {
	sec := []map[string][]string{{"session": {}}}
	adminErrs := []int{http.StatusUnauthorized, http.StatusForbidden}

	huma.Register(api, huma.Operation{
		OperationID: "listServarr", Method: http.MethodGet, Path: "/admin/servarr",
		Summary: "Radarr/Sonarr instances (admin)", Tags: []string{"admin"}, Security: sec, Errors: adminErrs,
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body []instanceView }, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		rows, err := d.Store.ListServarr(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]instanceView, 0, len(rows))
		for _, r := range rows {
			out = append(out, toInstanceView(r))
		}
		return &struct{ Body []instanceView }{out}, nil
	})

	validate := func(b instanceBody) error {
		u, err := url.Parse(b.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return huma.Error422UnprocessableEntity("url must be http(s)://host[:port]")
		}
		return nil
	}
	genreJSON := func(m map[string]string) (string, error) {
		for k, v := range m {
			if _, err := strconv.Atoi(k); err != nil || strings.TrimSpace(v) == "" {
				return "", huma.Error422UnprocessableEntity("genreRoots must map TMDB genre ids to root folders")
			}
		}
		b, _ := json.Marshal(m)
		return string(b), nil
	}

	huma.Register(api, huma.Operation{
		OperationID: "createServarr", Method: http.MethodPost, Path: "/admin/servarr",
		Summary: "Add a Radarr/Sonarr instance (admin)", Tags: []string{"admin"}, Security: sec,
		DefaultStatus: http.StatusCreated, Errors: append(adminErrs, http.StatusUnprocessableEntity),
	}, func(ctx context.Context, in *struct{ Body instanceBody }) (*struct{ Body instanceView }, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		if err := validate(in.Body); err != nil {
			return nil, err
		}
		if strings.TrimSpace(in.Body.APIKey) == "" {
			return nil, huma.Error422UnprocessableEntity("apiKey is required")
		}
		inst := &store.ServarrInstance{
			ID: store.NewID(), Kind: in.Body.Kind, Name: in.Body.Name, URL: strings.TrimRight(in.Body.URL, "/"),
			APIKey: strings.TrimSpace(in.Body.APIKey), QualityProfileID: in.Body.QualityProfileID, RootFolder: in.Body.RootFolder,
		}
		if in.Body.IsDefault {
			inst.IsDefault = 1
		}
		if in.Body.GenreRoots != nil {
			g, err := genreJSON(in.Body.GenreRoots)
			if err != nil {
				return nil, err
			}
			inst.GenreRoots = g
		}
		if err := d.Store.SaveServarr(ctx, inst); err != nil {
			return nil, err
		}
		saved, err := d.Store.GetServarr(ctx, inst.ID)
		if err != nil {
			return nil, err
		}
		return &struct{ Body instanceView }{toInstanceView(*saved)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "updateServarr", Method: http.MethodPut, Path: "/admin/servarr/{id}",
		Summary: "Update a Radarr/Sonarr instance (admin)", Tags: []string{"admin"}, Security: sec,
		Errors: append(adminErrs, http.StatusNotFound, http.StatusUnprocessableEntity),
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id" maxLength:"32"`
		Body instanceBody
	}) (*struct{ Body instanceView }, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		if err := validate(in.Body); err != nil {
			return nil, err
		}
		cur, err := d.Store.GetServarr(ctx, in.ID)
		if errors.Is(err, store.ErrNotFound) {
			return nil, huma.Error404NotFound("not found")
		}
		if err != nil {
			return nil, err
		}
		cur.Kind, cur.Name, cur.URL = in.Body.Kind, in.Body.Name, strings.TrimRight(in.Body.URL, "/")
		cur.QualityProfileID, cur.RootFolder = in.Body.QualityProfileID, in.Body.RootFolder
		if k := strings.TrimSpace(in.Body.APIKey); k != "" {
			cur.APIKey = k
		}
		if in.Body.IsDefault {
			cur.IsDefault = 1
		}
		if in.Body.GenreRoots != nil {
			g, err := genreJSON(in.Body.GenreRoots)
			if err != nil {
				return nil, err
			}
			cur.GenreRoots = g
		}
		if err := d.Store.SaveServarr(ctx, cur); err != nil {
			return nil, err
		}
		return &struct{ Body instanceView }{toInstanceView(*cur)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "deleteServarr", Method: http.MethodDelete, Path: "/admin/servarr/{id}",
		Summary: "Remove a Radarr/Sonarr instance (admin)", Tags: []string{"admin"}, Security: sec,
		DefaultStatus: http.StatusNoContent, Errors: adminErrs,
	}, func(ctx context.Context, in *struct {
		ID string `path:"id" maxLength:"32"`
	}) (*struct{}, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		return &struct{}{}, d.Store.DeleteServarr(ctx, in.ID)
	})

	huma.Register(api, huma.Operation{
		OperationID: "probeServarr", Method: http.MethodPost, Path: "/admin/servarr/probe",
		Summary:     "Test a Radarr/Sonarr connection and list its quality profiles and root folders (admin)",
		Description: "Read-only: uses a dry-run client, so it can never change anything in Radarr/Sonarr.",
		Tags:        []string{"admin"}, Security: sec, Errors: append(adminErrs, http.StatusUnprocessableEntity),
	}, func(ctx context.Context, in *struct{ Body probeBody }) (*struct{ Body probeResult }, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		key := strings.TrimSpace(in.Body.APIKey)
		if key == "" && in.Body.ID != "" {
			if cur, err := d.Store.GetServarr(ctx, in.Body.ID); err == nil {
				key = cur.APIKey
			}
		}
		if key == "" {
			return nil, huma.Error422UnprocessableEntity("apiKey is required")
		}
		c := servarr.New(in.Body.Kind, in.Body.URL, key, true) // reads only
		st, err := c.Status(ctx)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("cannot reach " + in.Body.Kind + ": " + err.Error())
		}
		profiles, err := c.QualityProfiles(ctx)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity(err.Error())
		}
		folders, err := c.RootFolders(ctx)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity(err.Error())
		}
		return &struct{ Body probeResult }{probeResult{AppName: st.AppName, Version: st.Version, Profiles: profiles, RootFolders: folders}}, nil
	})
}
