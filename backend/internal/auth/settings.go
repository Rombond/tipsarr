package auth

// Setting keys owned by setup / auth.
const (
	SettingJellyfinURL    = "jellyfin.url"
	SettingJellyfinAPIKey = "jellyfin.api_key"
	SettingTMDBKey        = "tmdb.api_key"

	// Single sign-on through an OpenID Connect provider (Authelia...).
	SettingOIDCIssuer       = "oidc.issuer"
	SettingOIDCClientID     = "oidc.client_id"
	SettingOIDCClientSecret = "oidc.client_secret"
	SettingOIDCAdminGroup   = "oidc.admin_group"
	SettingOIDCGroupsClaim  = "oidc.groups_claim"
	// SettingOIDCUsernameClaim is the ID token / userinfo claim matched to the Jellyfin username.
	SettingOIDCUsernameClaim = "oidc.username_claim"
	DefaultUsernameClaim     = "preferred_username"
)
