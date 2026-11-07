package api

import (
	"github.com/Rombond/tipsarr/backend/internal/demo"
	neturl "net/url"

	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/Rombond/tipsarr/backend/internal/notify"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Rombond/tipsarr/backend/internal/auth"
	"github.com/Rombond/tipsarr/backend/internal/avatars"
	"github.com/Rombond/tipsarr/backend/internal/boxoffice"
	"github.com/Rombond/tipsarr/backend/internal/library"
	"github.com/Rombond/tipsarr/backend/internal/media"
	"github.com/Rombond/tipsarr/backend/internal/netguard"
	"github.com/Rombond/tipsarr/backend/internal/perms"
	"github.com/Rombond/tipsarr/backend/internal/push"
	"github.com/Rombond/tipsarr/backend/internal/requests"
	"github.com/Rombond/tipsarr/backend/internal/trailers"
	"github.com/danielgtaylor/huma/v2"
)

type settingsBody struct {
	JellyfinURL              string `json:"jellyfinUrl"`
	JellyfinPublicURL        string `json:"jellyfinPublicUrl" doc:"Address browsers use to open Jellyfin (Play buttons); empty = same as jellyfinUrl"`
	TMDBConfigured           bool   `json:"tmdbConfigured" doc:"Secrets are write-only; this only says whether a key is saved"`
	JellyfinAPIKeyConfigured bool   `json:"jellyfinApiKeyConfigured" doc:"Needed for library and history sync"`
	BoxOfficeRegions         string `json:"boxofficeRegions" doc:"Comma-separated box-office region codes, e.g. US,GB,FR"`
	WebhookPath              string `json:"webhookPath" doc:"Path for the Jellyfin webhook plugin to call, with the secret in the X-Tipsarr-Token header (read it with POST /admin/secrets/jellyfin_webhook/reveal)"`
	WebhookTokenConfigured   bool   `json:"webhookTokenConfigured" doc:"The Jellyfin webhook secret exists; it is never returned here"`
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
	AppPublicURL             string `json:"appPublicUrl" doc:"Address people open Tipsarr at, e.g. https://tipsarr.example.org; webhook links ({{ url }}) start with it. Empty = no links"`
	DefaultLanguage          string `json:"defaultLanguage" doc:"App-wide default language, e.g. fr-FR; empty = the browser decides. A person's own language wins"`
	PushEnabled              bool   `json:"pushEnabled" doc:"Mobile push notifications through the relay (off by default)"`
	PushRelayURL             string `json:"pushRelayUrl" doc:"Address of the push relay, e.g. https://push.example.org"`
	PushText                 bool   `json:"pushText" doc:"Alerts name the media (needs a relay that allows free text for this app); off: generic alerts"`
	PushRelayKeyConfigured   bool   `json:"pushRelayKeyConfigured" doc:"The relay key is write-only; this only says whether one is saved"`
	PushKeyPublic            bool   `json:"pushKeyPublic" doc:"The key came from the relay's open registration: generic alerts only (pushText has no effect) and a daily quota"`
	PushDailyQuota           int    `json:"pushDailyQuota" doc:"Pushes per day the relay allows a public key; 0 = unlimited or unknown"`
	PushPausedUntil          int64  `json:"pushPausedUntil" doc:"Unix time when sending resumes after the relay's daily quota was reached; 0 = not paused"`
	PushDefaultRelayURL      string `json:"pushDefaultRelayUrl" doc:"The relay run by the app publisher, offered for registration"`
	ServarrAutoImport        bool   `json:"servarrAutoImport" doc:"Mirror what Radarr/Sonarr monitor but have not downloaded as approved requests (reads only)"`
	TeasersFilesEnabled      bool   `json:"teasersFilesEnabled" doc:"Teasers feed with trailer files from the optional trailer-resolver sidecar (off by default)"`
	TeasersFilesURL          string `json:"teasersFilesUrl" doc:"Address of the trailer resolver on the Docker network, e.g. http://trailer-resolver:8080"`
	TeasersFilesKeySet       bool   `json:"teasersFilesKeyConfigured" doc:"The resolver key is write-only; this only says whether one is saved"`
	TeasersFilesMaxHeight    int    `json:"teasersFilesMaxHeight" doc:"Largest video height the resolver downloads: 360, 480, 720 or 1080"`
	APIKeyConfigured         bool   `json:"apiKeyConfigured" doc:"The read-only dashboard key (X-Api-Key) is on; the key itself is read with POST /admin/secrets/api_key/reveal"`
}

type settingsOutput struct{ Body settingsBody }

type updateSettingsInput struct {
	Body struct {
		TMDBKey               *string `json:"tmdbApiKey,omitempty" doc:"Set or replace the TMDB key (empty string clears it)"`
		BoxOfficeRegions      *string `json:"boxofficeRegions,omitempty" doc:"Comma-separated region codes; empty resets to US"`
		JellyfinPublicURL     *string `json:"jellyfinPublicUrl,omitempty" doc:"Empty string clears it"`
		JellyfinAPIKey        *string `json:"jellyfinApiKey,omitempty" doc:"Jellyfin API key (Dashboard > API Keys); empty string clears it"`
		ServarrAutoImport     *bool   `json:"servarrAutoImport,omitempty"`
		PushEnabled           *bool   `json:"pushEnabled,omitempty" doc:"Needs the relay address and key; turning it on with either missing is refused"`
		PushText              *bool   `json:"pushText,omitempty"`
		PushRelayURL          *string `json:"pushRelayUrl,omitempty" doc:"Empty string clears it (and turns push off)"`
		PushRelayKey          *string `json:"pushRelayKey,omitempty" doc:"Write-only; empty string clears it (and turns push off)"`
		DefaultLanguage       *string `json:"defaultLanguage,omitempty" doc:"e.g. fr-FR; empty clears it"`
		AppPublicURL          *string `json:"appPublicUrl,omitempty" doc:"http(s)://host[/path]; empty clears it"`
		LDAPURL               *string `json:"ldapUrl,omitempty" doc:"Empty string turns LDAP pictures off; the connection is tested when saving"`
		LDAPBindDN            *string `json:"ldapBindDn,omitempty"`
		LDAPPassword          *string `json:"ldapBindPassword,omitempty" doc:"Write-only"`
		LDAPBaseDN            *string `json:"ldapBaseDn,omitempty"`
		OIDCIssuer            *string `json:"oidcIssuer,omitempty" doc:"Empty string turns single sign-on off. The provider is contacted when saving"`
		OIDCClientID          *string `json:"oidcClientId,omitempty"`
		OIDCClientSecret      *string `json:"oidcClientSecret,omitempty" doc:"Write-only; empty string clears it"`
		OIDCAdminGroup        *string `json:"oidcAdminGroup,omitempty"`
		OIDCGroupsClaim       *string `json:"oidcGroupsClaim,omitempty"`
		TeasersFilesEnabled   *bool   `json:"teasersFilesEnabled,omitempty" doc:"Needs the resolver address and key; turning it on with either missing is refused"`
		TeasersFilesURL       *string `json:"teasersFilesUrl,omitempty" doc:"Empty string clears it (and turns trailer files off)"`
		TeasersFilesKey       *string `json:"teasersFilesKey,omitempty" doc:"Write-only; empty string clears it (and turns trailer files off)"`
		TeasersFilesMaxHeight *int    `json:"teasersFilesMaxHeight,omitempty" doc:"360, 480, 720 or 1080"`
		APIKeyEnabled         *bool   `json:"apiKeyEnabled,omitempty" doc:"Turn the read-only dashboard key on (a key is created if there is none) or off (the key is deleted)"`
		RegenerateAPIKey      *bool   `json:"regenerateApiKey,omitempty" doc:"True: replace the dashboard key with a new one (the old one stops working)"`
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
		pushText, _ := d.Store.GetSetting(ctx, push.SettingText)
		pushPublic, _ := d.Store.GetSetting(ctx, push.SettingKeyPublic)
		pushQuota, _ := d.Store.GetSetting(ctx, push.SettingDailyQuota)
		quota, _ := strconv.Atoi(pushQuota)
		var paused int64
		if d.Push != nil {
			paused = d.Push.PausedUntil(ctx)
		}
		tfOn, _ := d.Store.GetSetting(ctx, trailers.SettingEnabled)
		tfURL, _ := d.Store.GetSetting(ctx, trailers.SettingURL)
		tfKey, _ := d.Store.GetSetting(ctx, trailers.SettingKey)
		apiKey, _ := d.Store.GetSetting(ctx, settingAPIKey)
		tfHeight := trailers.DefaultMaxHeight
		if v, _ := d.Store.GetSetting(ctx, trailers.SettingMaxHeight); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				tfHeight = n
			}
		}
		return settingsBody{
			APIKeyConfigured:    apiKey != "",
			TeasersFilesEnabled: tfOn == "true", TeasersFilesURL: tfURL, TeasersFilesKeySet: tfKey != "", TeasersFilesMaxHeight: tfHeight,
			PushEnabled: pushOn == "true", PushText: pushText == "true", PushRelayURL: pushURL, PushRelayKeyConfigured: pushKey != "",
			PushKeyPublic: pushPublic == "true" && pushKey != "", PushDailyQuota: quota, PushPausedUntil: paused, PushDefaultRelayURL: push.DefaultRelayURL,
			JellyfinURL: url, JellyfinPublicURL: publicURL, TMDBConfigured: key != "", DryRun: d.DryRun,
			JellyfinAPIKeyConfigured: jfKey != "", BoxOfficeRegions: regions, WebhookPath: "/api/v1/hooks/jellyfin", WebhookTokenConfigured: secret != "", ServarrAutoImport: autoImport != "false", DefaultLanguage: ldapGet(ctx, d, media.SettingDefaultLanguage), AppPublicURL: ldapGet(ctx, d, notify.SettingPublicURL),
			LDAPURL: ldapGet(ctx, d, avatars.SettingLDAPURL), LDAPBindDN: ldapGet(ctx, d, avatars.SettingLDAPBindDN), LDAPBaseDN: ldapGet(ctx, d, avatars.SettingLDAPBaseDN), LDAPPasswordSet: ldapGet(ctx, d, avatars.SettingLDAPPassword) != "",
			OIDCIssuer: oc.Issuer, OIDCClientID: oc.ClientID, OIDCClientSecretSet: oc.ClientSecret != "", OIDCAdminGroup: oc.AdminGroup, OIDCGroupsClaim: oc.GroupsClaim,
		}, nil
	}

	huma.Register(api, huma.Operation{
		OperationID: "getSettings", Method: http.MethodGet, Path: "/admin/settings",
		Summary: "Read settings (admin)", Tags: []string{"admin"}, Security: sec,
		Errors: []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, _ *struct{}) (*settingsOutput, error) {
		if _, err := requireCan(ctx, perms.ManageSettings); err != nil {
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
		if _, err := requireCan(ctx, perms.ManageSettings); err != nil {
			return nil, err
		}
		if d.Demo {
			if err := demoSettings(in); err != nil {
				return nil, err
			}
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
					slog.Info("OIDC discovery failed", "issuer", val, "err", err)
					return nil, fail(422, "oidc_unreachable", "cannot read the provider's configuration ("+string(netguard.Classify(err))+")")
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
					slog.Info("LDAP test failed", "err", err)
					return nil, fail(422, "ldap_unreachable", "cannot sign in to LDAP: "+avatars.LDAPFailure(err))
				}
			}
			for k, v := range next {
				if err := d.Store.SetSetting(ctx, k, v); err != nil {
					return nil, err
				}
			}
		}
		if in.Body.AppPublicURL != nil {
			v := strings.TrimRight(strings.TrimSpace(*in.Body.AppPublicURL), "/")
			if v != "" {
				if pu, err := neturl.Parse(v); err != nil || (pu.Scheme != "http" && pu.Scheme != "https") || pu.Host == "" {
					return nil, fail(422, "public_url_invalid", "appPublicUrl must be http(s)://host[/path]")
				}
			}
			if err := d.Store.SetSetting(ctx, notify.SettingPublicURL, v); err != nil {
				return nil, err
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
		if err := updateTeaserFiles(ctx, d, in.Body.TeasersFilesEnabled, in.Body.TeasersFilesURL, in.Body.TeasersFilesKey, in.Body.TeasersFilesMaxHeight); err != nil {
			return nil, err
		}
		if err := updateAPIKey(ctx, d, in.Body.APIKeyEnabled, in.Body.RegenerateAPIKey); err != nil {
			return nil, err
		}
		if in.Body.PushText != nil {
			v := "false"
			if *in.Body.PushText {
				v = "true"
			}
			if err := d.Store.SetSetting(ctx, push.SettingText, v); err != nil {
				return nil, err
			}
		}
		b, err := read(ctx)
		return &settingsOutput{Body: b}, err
	})

	if d.Push != nil {
		d.Push.Version = strings.TrimPrefix(Version, "v")
		d.Push.ServerURL = func(ctx context.Context) string { return ldapGet(ctx, d, notify.SettingPublicURL) }
	}
	huma.Register(api, huma.Operation{
		OperationID: "registerPush", Method: http.MethodPost, Path: "/admin/push/register",
		Summary:     "Turn mobile push on with the relay's open registration (admin)",
		Description: "Asks the relay for a key of its own (POST /v1/register with this server's install id), saves it with the address and turns push on. Registering again replaces the key.",
		Tags:        []string{"admin"}, Security: sec,
		Errors: []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusConflict, http.StatusUnprocessableEntity, http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable},
	}, func(ctx context.Context, in *struct {
		Body struct {
			RelayURL string `json:"relayUrl,omitempty" doc:"Relay address; empty = the publisher's relay (pushDefaultRelayUrl)"`
		}
	}) (*settingsOutput, error) {
		if _, err := requireCan(ctx, perms.ManageSettings); err != nil {
			return nil, err
		}
		if d.Push == nil {
			return nil, fail(503, "push_unavailable", "push is not available on this server")
		}
		u := strings.TrimSpace(in.Body.RelayURL)
		if u == "" {
			u = push.DefaultRelayURL
		}
		if err := push.ValidURL(strings.TrimRight(u, "/")); err != nil {
			return nil, fail(422, "invalid_push_url", err.Error())
		}
		if _, err := d.Push.Register(ctx, u); err != nil {
			return nil, pushRegisterErr(err)
		}
		b, err := read(ctx)
		return &settingsOutput{Body: b}, err
	})
}

// pushRegisterErr turns a relay refusal into an error code the apps translate.
func pushRegisterErr(err error) error {
	var re *push.RegisterError
	if !errors.As(err, &re) {
		return err
	}
	switch {
	case re.Status == 0 || re.Code == "unreachable":
		return fail(502, "push_relay_unreachable", err.Error())
	case re.Code == "registration_not_allowed":
		return fail(409, "push_register_not_allowed", err.Error())
	case re.Code == "registration_closed":
		return fail(503, "push_register_closed", err.Error())
	case re.Status == http.StatusTooManyRequests:
		return fail(429, "push_register_busy", fmt.Sprintf("%s, retry in %d s", err.Error(), re.RetryAfter))
	}
	return fail(502, "push_register_refused", err.Error())
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
		// a key pasted by hand is a manual key: no quota known, free text as the relay allows
		for _, k := range []string{push.SettingKeyPublic, push.SettingDailyQuota, push.SettingPausedUntil} {
			if err := set(k, ""); err != nil {
				return err
			}
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

// demoSettings applies the demo rules to a settings update: the TMDB key, the public address and the Jellyfin key
// are locked (silently kept), and turning on anything that would contact another service is refused.
func demoSettings(in *updateSettingsInput) error {
	b := &in.Body
	b.TMDBKey, b.AppPublicURL, b.JellyfinAPIKey = nil, nil, nil
	on := func(p *bool) bool { return p != nil && *p }
	set := func(p *string) bool { return p != nil && strings.TrimSpace(*p) != "" }
	if on(b.PushEnabled) || on(b.TeasersFilesEnabled) || set(b.LDAPURL) || set(b.OIDCIssuer) {
		return fail(http.StatusForbidden, demo.ErrorCode, demo.RefusalDetail)
	}
	return nil
}
