package api

import (
	"context"
	"net/http"
	"sort"

	"github.com/Rombond/tipsarr/backend/internal/perms"
	"github.com/danielgtaylor/huma/v2"
)

// maxHistorySources is how many accounts one person can borrow history from.
const maxHistorySources = 5

type historySource struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type historySourcesBody struct {
	Sources []historySource `json:"sources" doc:"Accounts whose watch history this person's recommendations also use, with their own. Titles they watched are hidden from the recommendations too"`
}

func registerHistorySources(api huma.API, d Deps) {
	sec := []map[string][]string{{"session": {}}, {"bearer": {}}}
	adminErrs := []int{http.StatusUnauthorized, http.StatusForbidden}

	read := func(ctx context.Context, userID string) (*struct{ Body historySourcesBody }, error) {
		ids, err := d.Store.HistorySources(ctx, userID)
		if err != nil {
			return nil, err
		}
		names, err := d.Store.UserNames(ctx, ids)
		if err != nil {
			return nil, err
		}
		out := &struct{ Body historySourcesBody }{historySourcesBody{Sources: []historySource{}}}
		for _, id := range ids {
			out.Body.Sources = append(out.Body.Sources, historySource{ID: id, Name: names[id]})
		}
		return out, nil
	}

	huma.Register(api, huma.Operation{
		OperationID: "myHistorySources", Method: http.MethodGet, Path: "/me/history-sources",
		Summary:     "Whose watch history your recommendations also use",
		Description: "Set by an admin (a shared TV account, say). Read-only here: it explains why titles that someone else watched appear in Discover.",
		Tags:        []string{"me"}, Security: sec, Errors: []int{http.StatusUnauthorized},
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body historySourcesBody }, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		return read(ctx, u.ID)
	})

	huma.Register(api, huma.Operation{
		OperationID: "adminHistorySources", Method: http.MethodGet, Path: "/admin/users/{id}/history-sources",
		Summary: "Whose watch history a person's recommendations also use (needs the manage-users right)", Tags: []string{"admin"}, Security: sec,
		Errors: append(adminErrs, http.StatusNotFound),
	}, func(ctx context.Context, in *struct {
		ID string `path:"id" maxLength:"64"`
	}) (*struct{ Body historySourcesBody }, error) {
		if _, err := requireCan(ctx, perms.ManageUsers); err != nil {
			return nil, err
		}
		if _, err := d.Store.GetUser(ctx, in.ID); err != nil {
			return nil, fail(404, "not_found", "not found")
		}
		return read(ctx, in.ID)
	})

	huma.Register(api, huma.Operation{
		OperationID: "setHistorySources", Method: http.MethodPut, Path: "/admin/users/{id}/history-sources",
		Summary:     "Set whose watch history a person's recommendations also use (needs the manage-users right)",
		Description: "Replaces the list. An empty list goes back to the person's own history. The sources must be Tipsarr users (import the shared account from Jellyfin first); a person cannot be their own source; at most 5. Only one level is followed: a source's own sources are not used. The person's recommendations are rebuilt in the background.",
		Tags:        []string{"admin"}, Security: sec,
		Errors: append(adminErrs, http.StatusNotFound, http.StatusUnprocessableEntity),
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id" maxLength:"64"`
		Body struct {
			Sources []string `json:"sources" maxItems:"5" doc:"Jellyfin user ids"`
		}
	}) (*struct{ Body historySourcesBody }, error) {
		if _, err := requireCan(ctx, perms.ManageUsers); err != nil {
			return nil, err
		}
		if _, err := d.Store.GetUser(ctx, in.ID); err != nil {
			return nil, fail(404, "not_found", "not found")
		}
		seen := map[string]bool{}
		var ids []string
		for _, id := range in.Body.Sources {
			if id == in.ID {
				return nil, fail(422, "history_source_self", "a person cannot use their own history as a source")
			}
			if seen[id] {
				continue
			}
			seen[id] = true
			if _, err := d.Store.GetUser(ctx, id); err != nil {
				return nil, fail(422, "history_source_unknown", "that account is not a Tipsarr user: import it from Jellyfin first")
			}
			ids = append(ids, id)
		}
		if len(ids) > maxHistorySources {
			return nil, fail(422, "history_source_too_many", "at most 5 sources")
		}
		sort.Strings(ids)
		if err := d.Store.SetHistorySources(ctx, in.ID, ids); err != nil {
			return nil, err
		}
		d.Suggestions.QueueRefresh(in.ID) // the version changed: rebuild now rather than at the next visit
		return read(ctx, in.ID)
	})
}
