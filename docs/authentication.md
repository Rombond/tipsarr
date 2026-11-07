# Authentication

Jellyfin is the only identity. Every Tipsarr account is a Jellyfin account; there is no separate user database or
password. Sessions are random tokens (an `HttpOnly`, `SameSite=Lax` cookie in the browser, a bearer token in the apps);
only their hash is stored.

## Jellyfin username and password

The default. Tipsarr checks the password against Jellyfin and does not keep it. A user disabled in Jellyfin is refused.

### Failed sign-ins

Failed password sign-ins are counted per username and per client address, in the database (a restart does not reset
them). After 8 failures for a username (30 for one address), each further failure locks it: 1 minute, then twice as
long each time, at most 1 hour. Every failure also waits about a second. A successful sign-in clears the username's
counter (not the address's); counters are forgotten after a day without failures. Lockouts are logged
(`sign-in locked after repeated failures`). The 1 hour cap means a stranger guessing an admin's password can delay
them, never lock them out for good. Jellyfin applies its own lockout too (its user setting "maximum failed login
attempts").

## Sessions

- A session lasts `TIPSARR_SESSION_TTL` (default `10d`) after its last use: using it pushes the expiry back.
- About once an hour, Tipsarr asks Jellyfin (with the saved API key) whether each signed-in account still exists, is
  enabled and is still an administrator. A removed or disabled account loses all its sessions at once.
- If Jellyfin cannot be reached, sessions keep working for `TIPSARR_SESSION_RECHECK_GRACE` (default `24h`), then are
  refused until Jellyfin answers again (they are not deleted). The count restarts when Tipsarr restarts.
- **Password changes**: Jellyfin does not tell Tipsarr when a password changes, so existing Tipsarr sessions survive
  it. After a password change (or a suspected leak), use **Profile → Devices → Sign out everywhere else**, or disable
  and re-enable the account in Jellyfin (that ends every Tipsarr session at the next check).

## Admin role

- A Jellyfin administrator becomes a Tipsarr admin when signing in. When Jellyfin takes that away, the Tipsarr role
  goes too: at the next session check (about an hour) or the next sign-in.
- A role given by the single sign-on **admin group** is checked again only at the next single sign-on sign-in (the
  group is not visible between sign-ins, and a password sign-in does not know it).
- A role given by hand in Users is never taken back by a sign-in.
- Admins created before role tracking existed have no recorded origin: the first sign-in after the upgrade records it. If Jellyfin
  (or the group) no longer makes such an account admin, the role is kept as if given by hand and a warning is logged
  (`kept the admin role of an account from before role tracking`): review those accounts in Users.
- Sign-in settings (single sign-on, LDAP), the Jellyfin API key, the public address and the dashboard API key can only be
  changed by an **admin**, not by someone who was only given the "manage settings" permission.

## OpenID Connect (single sign-on)

Settings → Single sign-on. Register a client at your provider, then enter its issuer address, client id and secret.

- **Redirect URI**: `https://<tipsarr>/api/v1/auth/oidc/callback` (shown in Settings).
- **Scopes**: `openid profile email groups`. PKCE (S256) is used.
- The provider's `preferred_username` (or the claim set in **Username claim**) must equal a **Jellyfin username exactly,
  case included**. There is no fallback to `name` or `nickname` (display names people can often change themselves).
  Those names must agree (they do when Jellyfin uses the LDAP plugin against the same directory). A Jellyfin API key
  must be saved. If you choose `email` as the username claim, an address the provider marks `email_verified: false`
  is refused. Pick a claim that people cannot edit at the provider.
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

A fresh install asks for a one-time token printed in the container log (`first run: ... setup token`) before it
accepts the first-run setup, so a stranger who reaches the port first cannot claim it. This is the default unless
`TIPSARR_JELLYFIN_URL` is set (that closes the setup page anyway). `TIPSARR_SETUP_TOKEN=false` turns it off.

## Client address

The address used for sign-in limits and logs is the connection's, unless the connection comes from a trusted proxy
(`TIPSARR_TRUSTED_PROXIES`, by default the private and loopback ranges): then the `X-Forwarded-For` header is read from
the right and the first untrusted address is used (`X-Real-IP` when there is no `X-Forwarded-For`). `True-Client-IP`
is ignored. If Tipsarr's port is reachable directly from a private network you do not control, set
`TIPSARR_TRUSTED_PROXIES` to your proxy's address only.
