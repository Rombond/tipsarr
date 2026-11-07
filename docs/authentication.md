# Authentication

Jellyfin is the only identity. Every Tipsarr account is a Jellyfin account; there is no separate user database or
password. Sessions are random tokens (an `HttpOnly`, `SameSite=Lax` cookie in the browser, a bearer token in the apps);
only their hash is stored. Failed password logins are rate limited.

## Jellyfin username and password

The default. Tipsarr checks the password against Jellyfin and does not keep it. A user disabled in Jellyfin is refused.

## OpenID Connect (single sign-on)

Settings → Single sign-on. Register a client at your provider, then enter its issuer address, client id and secret.

- **Redirect URI**: `https://<tipsarr>/api/v1/auth/oidc/callback` (shown in Settings).
- **Scopes**: `openid profile email groups`. PKCE (S256) is used.
- The provider's `preferred_username` is matched, ignoring case, to a **Jellyfin username**. Those names must agree (they
  do when Jellyfin uses the LDAP plugin against the same directory). A Jellyfin API key must be saved.
- Members of the optional **admin group** (from the `groups` claim) become admins; Jellyfin admins always are.
- Tipsarr itself must reach the provider's address (from the container), not only your browser.
- When single sign-on is on, the password form is hidden. If sign-on refuses someone (no matching or disabled Jellyfin
  account, provider error, expired attempt) the login page says why and shows the password form again.
  `/login?password=1` always shows it, in case the provider is down.

The mobile apps sign in with the Jellyfin username and password; they do not offer single sign-on yet.
<!-- TODO: confirm the apps have no OIDC sign-in (none found in ios/ or android/ code). -->

### Authelia example

```yaml
identity_providers:
  oidc:
    clients:
      - client_id: tipsarr
        client_name: Tipsarr
        client_secret: '$pbkdf2-sha512$...'   # hash of the secret you enter in Tipsarr
        public: false
        authorization_policy: one_factor
        require_pkce: true
        pkce_challenge_method: S256
        redirect_uris: ['https://tipsarr.example.org/api/v1/auth/oidc/callback']
        scopes: [openid, profile, email, groups]
```

Authentik and Keycloak work the same way: a confidential OAuth2/OpenID client with that redirect URI and those scopes.

## LDAP

Tipsarr does not bind to LDAP to sign people in. Point Jellyfin at your directory with Jellyfin's LDAP plugin: LDAP users
then sign in to Tipsarr with their LDAP password (checked by Jellyfin), or through OpenID Connect if your provider uses
the same directory. Tipsarr only reads LDAP for profile pictures (see [configuration](configuration.md#profile-pictures-from-ldap)).

## First run on an exposed address

If the port is reachable from the internet before you finish the setup, start with `TIPSARR_SETUP_TOKEN=true`: the
first-run page then asks for a one-time token printed in the container log.
