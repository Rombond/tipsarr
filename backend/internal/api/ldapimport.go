package api

import (
	"context"
	"net/http"

	"github.com/Rombond/tipsarr/backend/internal/auth"
	"github.com/Rombond/tipsarr/backend/internal/avatars"
	"github.com/Rombond/tipsarr/backend/internal/clients/jellyfin"
	"github.com/Rombond/tipsarr/backend/internal/library"
	"github.com/danielgtaylor/huma/v2"
)

type ldapImportBody struct {
	URL       string `json:"ldapUrl"`
	BindDN    string `json:"ldapBindDn"`
	BaseDN    string `json:"ldapBaseDn"`
	Connected bool   `json:"connected" doc:"True when Tipsarr could sign in to the LDAP server with the imported settings"`
	Error     string `json:"error,omitempty" doc:"Why the connection test failed (the settings are saved anyway so they can be corrected)"`
}

// importLDAPFromJellyfin copies the settings of Jellyfin's LDAP Authentication plugin (address,
// bind account, password, base DN) into Tipsarr's LDAP settings. (nil, nil): no such plugin.
func importLDAPFromJellyfin(ctx context.Context, d Deps) (*ldapImportBody, error) {
	jfURL, _ := d.Store.GetSetting(ctx, auth.SettingJellyfinURL)
	key, _ := d.Store.GetSetting(ctx, library.SettingJellyfinAPIKey)
	if jfURL == "" || key == "" {
		return nil, fail(503, "jellyfin_key_missing", "save a Jellyfin API key in Settings first")
	}
	cfg, err := jellyfin.New(jfURL).WithToken(key).LDAPPlugin(ctx)
	if err != nil {
		return nil, fail(502, "jellyfin_unreachable", "cannot read Jellyfin's plugins: "+err.Error())
	}
	if cfg == nil {
		return nil, nil
	}
	out := &ldapImportBody{URL: cfg.URL(), BindDN: cfg.BindUser, BaseDN: cfg.BaseDN}
	for k, v := range map[string]string{
		avatars.SettingLDAPURL: out.URL, avatars.SettingLDAPBindDN: cfg.BindUser,
		avatars.SettingLDAPPassword: cfg.BindPassword, avatars.SettingLDAPBaseDN: cfg.BaseDN,
	} {
		if err := d.Store.SetSetting(ctx, k, v); err != nil {
			return nil, err
		}
	}
	if err := avatars.CheckLDAP(ctx, out.URL, cfg.BindUser, cfg.BindPassword); err != nil {
		out.Error = err.Error()
	} else {
		out.Connected = true
	}
	return out, nil
}

func registerLDAPImport(api huma.API, d Deps) {
	huma.Register(api, huma.Operation{
		OperationID: "importLdapFromJellyfin", Method: http.MethodPost, Path: "/admin/ldap/import-jellyfin",
		Summary:     "Copy the LDAP settings of Jellyfin's LDAP Authentication plugin (admin)",
		Description: "Needs the Jellyfin API key. The password is copied server-side and never sent to the browser.",
		Tags:        []string{"admin"}, Security: []map[string][]string{{"session": {}}, {"bearer": {}}},
		Errors: []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusServiceUnavailable, http.StatusBadGateway},
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body ldapImportBody }, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		res, err := importLDAPFromJellyfin(ctx, d)
		if err != nil {
			return nil, err
		}
		if res == nil {
			return nil, fail(404, "ldap_plugin_not_found", "Jellyfin has no configured LDAP Authentication plugin")
		}
		return &struct{ Body ldapImportBody }{*res}, nil
	})
}
