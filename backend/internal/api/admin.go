package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/Rombond/tipsarr/backend/internal/auth"
	"github.com/Rombond/tipsarr/backend/internal/avatars"
	"github.com/Rombond/tipsarr/backend/internal/boxoffice"
	"github.com/Rombond/tipsarr/backend/internal/library"
	"github.com/Rombond/tipsarr/backend/internal/media"
	"github.com/Rombond/tipsarr/backend/internal/push"
	"github.com/Rombond/tipsarr/backend/internal/requests"
	"github.com/danielgtaylor/huma/v2"
)

type settingsBody struct {
	JellyfinURL              string `json:"jellyfinUrl"`
	JellyfinPublicURL        string `json:"jellyfinPublicUrl" doc:"Address browsers use to open Jellyfin (Play buttons); empty = same as jellyfinUrl"`
	TMDBConfigured           bool   `json:"tmdbConfigured" doc:"Secrets are write-only; this only says whether a key is saved"`
	JellyfinAPIKeyConfigured bool   `json:"jellyfinApiKeyConfigured" doc:"Needed for library and history sync"`
	BoxOfficeRegions         string `json:"boxofficeRegions" doc:"Comma-separated box-office region codes, e.g. US,GB,FR"`
	WebhookPath              string `json:"webhookPath" doc:"Path (with secret token) for the Jellyfin webhook plugin to call"`
	DryRun                   bool   `json:"dryRun" doc:"When true nothing is ever sent to Radarr/Sonarr"`
	OIDCIssuer               string `json:"oidcIssuer" doc:"OpenID Connect provider (Authelia), e.g. https://auth.example.org"`
	OIDCClientID             string `json:"oidcClientId"`
	OIDCClientSecretSet      bool   `json:"oidcClientSecretConfigured"`
	OIDCAdminGroup           string `json:"oidcAdminGroup" doc:"Members of this group become admins (optional)"`
	OIDCGroupsClaim          string `json:"oidcGroupsClaim" doc:"Claim that lists the groups, default groups"`
	LDAPURL                  string `json:"ldapUrl" doc:"LLDAP/LDAP address for profile pictures, e.g. ldap://lldap:3890"`
	LDAPBindDN               string `json:"ldapBindDn"`
	LDAPPasswordSet          bool   `json:"ldapBindPasswordConfigured"`
	LDAPBaseDN               string `json:"ldapBaseDn" doc:"Where users live, e.g. ou=people,dc=example,dc=com"`
	UserFolderChoice         bool   `json:"userFolderChoice" doc:"Non-admins may choose the root folder when requesting (default false); the quality profile is always their choice"`
	DefaultLanguage          string `json:"defaultLanguage" doc:"App-wide default language, e.g. fr-FR; empty = the browser decides. A person's own language wins"`
	PushEnabled              bool   `json:"pushEnabled" doc:"Mobile push notifications through the relay (off by default)"`
	PushRelayURL             string `json:"pushRelayUrl" doc:"Address of the push relay, e.g. https://push.example.org"`
	PushRelayKeyConfigured   bool   `json:"pushRelayKeyConfigured" doc:"The relay key is write-only; this only says whether one is saved"`
	ServarrAutoImport        bool   `json:"servarrAutoImport" doc:"Mirror what Radarr/Sonarr monitor but have not downloaded as approved requests (reads only)"`
}

type settingsOutput struct{ Body settingsBody }

type updateSettingsInput struct {
	Body struct {
		TMDBKey           *string `json:"tmdbApiKey,omitempty" doc:"Set or replace the TMDB key (empty string clears it)"`
		BoxOfficeRegions  *string `json:"boxofficeRegions,omitempty" doc:"Comma-separated region codes; empty resets to US"`
		JellyfinPublicURL *string `json:"jellyfinPublicUrl,omitempty" doc:"Empty string clears it"`
		JellyfinAPIKey    *string `json:"jellyfinApiKey,omitempty" doc:"Jellyfin API key (Dashboard > API Keys); empty string clears it"`
		ServarrAutoImport *bool   `json:"servarrAutoImport,omitempty"`
		PushEnabled       *bool   `json:"pushEnabled,omitempty" doc:"Needs the relay address and key; turning it on with either missing is refused"`
		PushRelayURL      *string `json:"pushRelayUrl,omitempty" doc:"Empty string clears it (and turns push off)"`
		PushRelayKey      *string `json:"pushRelayKey,omitempty" doc:"Write-only; empty string clears it (and turns push off)"`
		UserFolderChoice  *bool   `json:"userFolderChoice,omitempty"`
		DefaultLanguage   *string `json:"defaultLanguage,omitempty" doc:"e.g. fr-FR; empty clears it"`
		LDAPURL           *string `json:"ldapUrl,omitempty" doc:"Empty string turns LDAP pictures off; the connection is tested when saving"`
		LDAPBindDN        *string `json:"ldapBindDn,omitempty"`
		LDAPPassword      *string `json:"ldapBindPassword,omitempty" doc:"Write-only"`
		LDAPBaseDN        *string `json:"ldapBaseDn,omitempty"`
		OIDCIssuer        *string `json:"oidcIssuer,omitempty" doc:"Empty string turns single sign-on off. The provider is contacted when saving"`
		OIDCClientID      *string `json:"oidcClientId,omitempty"`
		OIDCClientSecret  *string `json:"oidcClientSecret,omitempty" doc:"Write-only; empty string clears it"`
		OIDCAdminGroup    *string `json:"oidcAdminGroup,omitempty"`
		OIDCGroupsClaim   *string `json:"oidcGroupsClaim,omitempty"`
	}
}

func registerAdmin(api huma.API, d Deps) {
	sec := []map[string][]string{{"session": {}}, {"bearer": {}}}

	read := func(ctx context.Context) (settingsBody, error) {
		url, err := d.Store.GetSetting(ctx, auth.SettingJellyfinURL)
		if err != nil {
			return settingsBody{}, err
		}
		key, err := d.Store.GetSetting(ctx, auth.SettingTMDBKey)
		if err != nil {
			return settingsBody{}, err
		}
		jfKey, err := d.Store.GetSetting(ctx, library.SettingJellyfinAPIKey)
		if err != nil {
			return settingsBody{}, err
		}
		secret, err := webhookSecret(ctx, d)
		if err != nil {
			return settingsBody{}, err
		}
		regions := strings.Join(d.BoxOffice.Regions(ctx), ",")
		publicURL, _ := d.Store.GetSetting(ctx, media.SettingJellyfinPublicURL)
		autoImport, _ := d.Store.GetSetting(ctx, requests.SettingAutoImport)
		oc, _ := d.Auth.OIDCConfig(ctx)
		pushOn, _ := d.Store.GetSetting(ctx, push.SettingEnabled)
		pushURL, _ := d.Store.GetSetting(ctx, push.SettingURL)
		pushKey, _ := d.Store.GetSetting(ctx, push.SettingKey)
		return settingsBody{
			PushEnabled: pushOn == "true", PushRelayURL: pushURL, PushRelayKeyConfigured: pushKey != "",
			JellyfinURL: url, JellyfinPublicURL: publicURL, TMDBConfigured: key != "", DryRun: d.DryRun,
			JellyfinAPIKeyConfigured: jfKey != "", BoxOfficeRegions: regions, WebhookPath: "/api/v1/hooks/jellyfin?token=" + secret, ServarrAutoImport: autoImport != "false", UserFolderChoice: d.Requests.UsersMayChooseFolder(ctx), DefaultLanguage: ldapGet(ctx, d, media.SettingDefaultLanguage),
			LDAPURL: ldapGet(ctx, d, avatars.SettingLDAPURL), LDAPBindDN: ldapGet(ctx, d, avatars.SettingLDAPBindDN), LDAPBaseDN: ldapGet(ctx, d, avatars.SettingLDAPBaseDN), LDAPPasswordSet: ldapGet(ctx, d, avatars.SettingLDAPPassword) != "",
			OIDCIssuer: oc.Issuer, OIDCClientID: oc.ClientID, OIDCClientSecretSet: oc.ClientSecret != "", OIDCAdminGroup: oc.AdminGroup, OIDCGroupsClaim: oc.GroupsClaim,
		}, nil
	}

	huma.Register(api, huma.Operation{
		OperationID: "getSettings", Method: http.MethodGet, Path: "/admin/settings",
		Summary: "Read settings (admin)", Tags: []string{"admin"}, Security: sec,
		Errors: []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, _ *struct{}) (*settingsOutput, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		b, err := read(ctx)
		return &settingsOutput{Body: b}, err
	})

	huma.Register(api, huma.Operation{
		OperationID: "updateSettings", Method: http.MethodPut, Path: "/admin/settings",
		Summary: "Update settings (admin)", Tags: []string{"admin"}, Security: sec,
		Errors: []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, in *updateSettingsInput) (*settingsOutput, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		if in.Body.TMDBKey != nil {
			if err := d.Store.SetSetting(ctx, auth.SettingTMDBKey, strings.TrimSpace(*in.Body.TMDBKey)); err != nil {
				return nil, err
			}
		}
		if in.Body.JellyfinPublicURL != nil {
			v := strings.TrimRight(strings.TrimSpace(*in.Body.JellyfinPublicURL), "/")
			if v != "" && !strings.HasPrefix(v, "http://") && !strings.HasPrefix(v, "https://") {
				return nil, fail(422, "url_invalid", "jellyfinPublicUrl must start with http:// or https://")
			}
			if err := d.Store.SetSetting(ctx, media.SettingJellyfinPublicURL, v); err != nil {
				return nil, err
			}
		}
		if in.Body.BoxOfficeRegions != nil {
			regions := strings.Join(boxoffice.ParseRegions(*in.Body.BoxOfficeRegions), ",")
			if err := d.Store.SetSetting(ctx, boxoffice.SettingRegions, regions); err != nil {
				return nil, err
			}
		}
		if in.Body.JellyfinAPIKey != nil {
			if err := d.Store.SetSetting(ctx, library.SettingJellyfinAPIKey, strings.TrimSpace(*in.Body.JellyfinAPIKey)); err != nil {
				return nil, err
			}
			// Jellyfin is now linked: prefill the LDAP settings from its LDAP plugin, if there is one
			// and LDAP is not configured yet (best effort, never blocks saving the key).
			if strings.TrimSpace(*in.Body.JellyfinAPIKey) != "" && in.Body.LDAPURL == nil && ldapGet(ctx, d, avatars.SettingLDAPURL) == "" {
				if _, err := importLDAPFromJellyfin(ctx, d); err != nil {
					slog.Info("LDAP prefill from Jellyfin skipped", "err", err)
				}
			}
		}
		for key, v := range map[string]*string{
			auth.SettingOIDCIssuer: in.Body.OIDCIssuer, auth.SettingOIDCClientID: in.Body.OIDCClientID,
			auth.SettingOIDCClientSecret: in.Body.OIDCClientSecret, auth.SettingOIDCAdminGroup: in.Body.OIDCAdminGroup,
			auth.SettingOIDCGroupsClaim: in.Body.OIDCGroupsClaim,
		} {
			if v == nil {
				continue
			}
			val := strings.TrimSpace(*v)
			if key == auth.SettingOIDCIssuer && val != "" {
				if u, err := url.Parse(val); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
					return nil, fail(422, "url_invalid", "oidcIssuer must start with http:// or https://")
				}
				if err := d.Auth.CheckOIDC(ctx, val); err != nil {
					return nil, fail(422, "oidc_unreachable", "cannot read the provider's configuration: "+err.Error())
				}
			}
			if err := d.Store.SetSetting(ctx, key, val); err != nil {
				return nil, err
			}
		}
		if in.Body.LDAPURL != nil || in.Body.LDAPBindDN != nil || in.Body.LDAPPassword != nil || in.Body.LDAPBaseDN != nil {
			vals := map[string]*string{avatars.SettingLDAPURL: in.Body.LDAPURL, avatars.SettingLDAPBindDN: in.Body.LDAPBindDN, avatars.SettingLDAPPassword: in.Body.LDAPPassword, avatars.SettingLDAPBaseDN: in.Body.LDAPBaseDN}
			next := map[string]string{}
			for k := range vals {
				next[k] = ldapGet(ctx, d, k)
				if vals[k] != nil {
					next[k] = strings.TrimSpace(*vals[k])
				}
			}
			if next[avatars.SettingLDAPURL] != "" {
				if err := avatars.CheckLDAP(ctx, next[avatars.SettingLDAPURL], next[avatars.SettingLDAPBindDN], next[avatars.SettingLDAPPassword]); err != nil {
					return nil, fail(422, "ldap_unreachable", "cannot sign in to LDAP: "+err.Error())
				}
			}
			for k, v := range next {
				if err := d.Store.SetSetting(ctx, k, v); err != nil {
					return nil, err
				}
			}
		}
		if in.Body.DefaultLanguage != nil {
			v := strings.TrimSpace(*in.Body.DefaultLanguage)
			if v != "" && !languageRe.MatchString(v) {
				return nil, fail(422, "bad_language", "language must look like fr or fr-FR")
			}
			if err := d.Store.SetSetting(ctx, media.SettingDefaultLanguage, v); err != nil {
				return nil, err
			}
		}
		if in.Body.UserFolderChoice != nil {
			v := "false"
			if *in.Body.UserFolderChoice {
				v = "true"
			}
			if err := d.Store.SetSetting(ctx, requests.SettingUserFolder, v); err != nil {
				return nil, err
			}
		}
		if in.Body.ServarrAutoImport != nil {
			v := "true"
			if !*in.Body.ServarrAutoImport {
				v = "false"
			}
			if err := d.Store.SetSetting(ctx, requests.SettingAutoImport, v); err != nil {
				return nil, err
			}
		}
		if err := updatePush(ctx, d, in.Body.PushEnabled, in.Body.PushRelayURL, in.Body.PushRelayKey); err != nil {
			return nil, err
		}
		b, err := read(ctx)
		return &settingsOutput{Body: b}, err
	})
}

// updatePush saves the relay settings. Push cannot be on without an address and a key.
func updatePush(ctx context.Context, d Deps, enabled *bool, relayURL, relayKey *string) error {
	set := func(k, v string) error { return d.Store.SetSetting(ctx, k, v) }
	if relayURL != nil {
		v := strings.TrimRight(strings.TrimSpace(*relayURL), "/")
		if v != "" {
			if err := push.ValidURL(v); err != nil {
				return fail(422, "invalid_push_url", err.Error())
			}
		}
		if err := set(push.SettingURL, v); err != nil {
			return err
		}
		if v == "" {
			enabled = ptr(false)
		}
	}
	if relayKey != nil {
		v := strings.TrimSpace(*relayKey)
		if err := set(push.SettingKey, v); err != nil {
			return err
		}
		if v == "" {
			enabled = ptr(false)
		}
	}
	if enabled == nil {
		return nil
	}
	if *enabled {
		u, _ := d.Store.GetSetting(ctx, push.SettingURL)
		k, _ := d.Store.GetSetting(ctx, push.SettingKey)
		if u == "" || k == "" {
			return fail(422, "push_not_configured", "set the relay address and key before turning push on")
		}
	}
	v := "false"
	if *enabled {
		v = "true"
	}
	return set(push.SettingEnabled, v)
}

func ptr[T any](v T) *T { return &v }

func ldapGet(ctx context.Context, d Deps, key string) string {
	v, _ := d.Store.GetSetting(ctx, key)
	return v
}

const settingWebhookSecret = "hooks.jellyfin_secret"

// webhookSecret returns the shared secret for the Jellyfin webhook, generating it on first use.
func webhookSecret(ctx context.Context, d Deps) (string, error) {
	secret, err := d.Store.GetSetting(ctx, settingWebhookSecret)
	if err != nil || secret != "" {
		return secret, err
	}
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	secret = hex.EncodeToString(raw)
	return secret, d.Store.SetSetting(ctx, settingWebhookSecret, secret)
}
