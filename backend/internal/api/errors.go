package api

import "github.com/danielgtaylor/huma/v2"

// fail builds an API error that carries a stable machine-readable code next to the English
// message, so clients can show the message in the user's language (the frontend maps `code`
// to its own text and falls back to `detail`).
func fail(status int, code, detail string) error {
	return huma.NewError(status, detail, &huma.ErrorDetail{Message: detail, Location: "code", Value: code})
}
