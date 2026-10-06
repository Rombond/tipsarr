package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type healthOutput struct {
	Body struct {
		OK bool `json:"ok"`
	}
}

type statusOutput struct {
	Body struct {
		Version string `json:"version"`
		DB      string `json:"db" doc:"Database engine: sqlite, postgres or mysql"`
		DryRun  bool   `json:"dryRun" doc:"When true nothing is ever sent to Radarr/Sonarr"`
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
		return out, nil
	})
}
