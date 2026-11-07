package api

import (
	"log/slog"
	"strconv"

	"github.com/Rombond/tipsarr/backend/internal/store"
	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5/middleware"
)

// fail builds an API error that carries a stable machine-readable code next to the English
// message, so clients can show the message in the user's language (the frontend maps `code`
// to its own text and falls back to `detail`).
func fail(status int, code, detail string) error {
	return huma.NewError(status, detail, &huma.ErrorDetail{Message: detail, Location: "code", Value: code})
}

// hideInternalErrors is a huma transformer: a 5xx answer to anyone but an admin keeps its status and
// machine-readable code, but its message becomes generic with the request id, and every other error
// detail (often a raw err.Error() with internal addresses) is dropped. The full error is logged.
func hideInternalErrors(ctx huma.Context, _ string, v any) (any, error) {
	em, ok := v.(*huma.ErrorModel)
	if !ok || em.Status < 500 {
		return v, nil
	}
	if u := userFrom(ctx.Context()); u != nil && u.Role == store.RoleAdmin {
		return v, nil
	}
	reqID := middleware.GetReqID(ctx.Context())
	args := []any{"status", em.Status, "path", ctx.URL().Path, "request_id", reqID, "detail", em.Detail}
	for _, d := range em.Errors {
		if d != nil && d.Location != "code" {
			args = append(args, "error", d.Message)
		}
	}
	slog.Warn("request failed", args...)
	out := &huma.ErrorModel{Type: em.Type, Title: em.Title, Status: em.Status, Instance: em.Instance,
		Detail: "internal error (request id " + reqID + ")"}
	if reqID == "" {
		out.Detail = "internal error (status " + strconv.Itoa(em.Status) + ")"
	}
	for _, d := range em.Errors {
		if d != nil && d.Location == "code" {
			out.Errors = append(out.Errors, &huma.ErrorDetail{Location: "code", Value: d.Value, Message: out.Detail})
		}
	}
	return out, nil
}
