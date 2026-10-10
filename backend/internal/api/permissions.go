package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/Rombond/tipsarr/backend/internal/perms"
	"github.com/Rombond/tipsarr/backend/internal/requests"
	"github.com/Rombond/tipsarr/backend/internal/store"
	"github.com/danielgtaylor/huma/v2"
)

// permissionDefaults is what new people start with: a permission mask and the request quotas of people without one
// of their own. A limit of 0 is "no limit".
type permissionDefaults struct {
	Permissions     int64 `json:"permissions" doc:"Mask new people start with (see flags)"`
	MovieQuotaLimit int   `json:"movieQuotaLimit" minimum:"0" maximum:"100000" doc:"Movie requests allowed per period, 0 = no limit"`
	MovieQuotaDays  int   `json:"movieQuotaDays" minimum:"0" maximum:"3650"`
	TVQuotaLimit    int   `json:"tvQuotaLimit" minimum:"0" maximum:"100000" doc:"Seasons of shows allowed per period, 0 = no limit"`
	TVQuotaDays     int   `json:"tvQuotaDays" minimum:"0" maximum:"3650"`
}

func loadDefaults(ctx context.Context, st *store.Store) permissionDefaults {
	return permissionDefaults{
		Permissions:     int64(st.DefaultPermissions(ctx)),
		MovieQuotaLimit: st.SettingInt(ctx, store.SettingQuotaMovieLimit, 0), MovieQuotaDays: st.SettingInt(ctx, store.SettingQuotaMovieDays, 0),
		TVQuotaLimit: st.SettingInt(ctx, store.SettingQuotaTVLimit, 0), TVQuotaDays: st.SettingInt(ctx, store.SettingQuotaTVDays, 0),
	}
}

func registerPermissions(api huma.API, d Deps) {
	sec := []map[string][]string{{"session": {}}, {"bearer": {}}}
	adminErrs := []int{http.StatusUnauthorized, http.StatusForbidden}

	huma.Register(api, huma.Operation{
		OperationID: "getPermissions", Method: http.MethodGet, Path: "/admin/permissions",
		Summary:     "The permission flags and the defaults for new people (needs the manage-users right)",
		Description: "Each person has a mask: the sum of the values of the flags they have. Admins can do everything whatever their mask says. Flags marked reserved exist so the numbers match Seerr but no feature uses them yet: do not show them.",
		Tags:        []string{"admin"}, Security: sec, Errors: adminErrs,
	}, func(ctx context.Context, _ *struct{}) (*struct {
		Body struct {
			Flags    []perms.Flag       `json:"flags"`
			Defaults permissionDefaults `json:"defaults"`
		}
	}, error) {
		if _, err := requireCan(ctx, perms.ManageUsers); err != nil {
			return nil, err
		}
		out := &struct {
			Body struct {
				Flags    []perms.Flag       `json:"flags"`
				Defaults permissionDefaults `json:"defaults"`
			}
		}{}
		out.Body.Flags = perms.Catalog
		out.Body.Defaults = loadDefaults(ctx, d.Store)
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "setPermissionDefaults", Method: http.MethodPut, Path: "/admin/permissions/defaults",
		Summary:     "Set the permissions and quotas new people start with (admin)",
		Description: "Only changes people who sign in or are imported from now on.",
		Tags:        []string{"admin"}, Security: sec, Errors: append(adminErrs, http.StatusUnprocessableEntity),
	}, func(ctx context.Context, in *struct{ Body permissionDefaults }) (*struct{ Body permissionDefaults }, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		if err := d.Store.SetDefaultPermissions(ctx, perms.Mask(in.Body.Permissions)); err != nil {
			return nil, err
		}
		for key, v := range map[string]int{
			store.SettingQuotaMovieLimit: in.Body.MovieQuotaLimit, store.SettingQuotaMovieDays: in.Body.MovieQuotaDays,
			store.SettingQuotaTVLimit: in.Body.TVQuotaLimit, store.SettingQuotaTVDays: in.Body.TVQuotaDays,
		} {
			if err := d.Store.SetSetting(ctx, key, strconv.Itoa(v)); err != nil {
				return nil, err
			}
		}
		return &struct{ Body permissionDefaults }{loadDefaults(ctx, d.Store)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "myQuota", Method: http.MethodGet, Path: "/me/quota",
		Summary:     "How many requests the signed-in person may still make",
		Description: "Movies count one per request, shows one per season, over the last N days. Admins and people who manage requests have no limit.",
		Tags:        []string{"requests"}, Security: sec, Errors: []int{http.StatusUnauthorized},
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body *requests.Quota }, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		q, err := d.Requests.Quota(ctx, u)
		return &struct{ Body *requests.Quota }{q}, err
	})
}
