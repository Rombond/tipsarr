# Privacy

Tipsarr is self-hosted software. There is no Tipsarr account, no cloud service, no analytics and no tracking. This page
describes what each part handles. Last updated: 2026-11-07.

## The Tipsarr server

The server is run by an administrator (you, or whoever invited you), who controls all its data. It stores in its own
database: the Jellyfin user id, name and role of each person who signs in; session hashes; requests, issues, comments,
watchlists and "not interested" lists; a copy of the watch history read from Jellyfin (for suggestions and stats);
uploaded profile pictures; settings and API keys. Passwords are never stored: they are checked by Jellyfin.

The server contacts, depending on its configuration:

- **TMDB** (`api.themoviedb.org`, `image.tmdb.org`) for metadata and images, with the administrator's API key. Requests
  carry titles and ids being looked up, not user names. Images are proxied and cached, so browsers and apps never
  contact TMDB directly.
- **Box Office Mojo** for box-office charts.
- **Jellyfin, Radarr, Sonarr**, LDAP and the OpenID Connect provider configured by the administrator.
- **Webhook destinations** the administrator configures (these receive what their templates contain, which may include
  titles and user names).
- **The push relay**, only if the administrator turns push on (see below).

The public demo (`tipsarr-demo.brebond.com`) runs throwaway copies with sample data; everything in a sandbox is deleted
when the session ends.

## The mobile apps

- **What they store on the device**: the server addresses and session tokens (iOS Keychain; Android private storage
  encrypted with the Android Keystore), recent searches, and display preferences. On iOS, if you turn on Face ID
  sign-in, your password is kept in the Keychain behind Face ID to sign you back in.
- **Who they talk to**: only your Tipsarr server(s), and Apple's or Google's push service to get a push token. Opening
  a trailer or an external link (TMDB, IMDb, YouTube…) opens it in the corresponding app or browser.
- **No analytics, no advertising, no crash reporting SDK.** The Android app includes Firebase Cloud Messaging only to
  receive push notifications (no Firebase Analytics); the iOS app has no third-party SDK.
- **Push token**: sent to your Tipsarr server with the platform, language, the notification categories you chose and
  the app version, so it can address notifications to you. It is tied to your sign-in session and deleted from the
  server when you sign out or the session is revoked.

## The push relay (`push.brebond.com`)

Run by the app publisher, only used when your server's administrator turns push on. Source:
[Rombond/push-relay](https://github.com/Rombond/push-relay).

- **Per notification it receives**: the device push token, platform, event name (e.g. `request.approved`), an id
  (request or issue number), the language and the host name of the Tipsarr server. It builds a generic text from its own
  templates and forwards it to Apple (APNs) or Google (FCM). No title, user name, comment or picture.
- **It does not store device tokens.** Logs contain the key id, app, platform, event and a short hash of the token.
- **At registration** of a server: a random install id, optionally the server's public address and Tipsarr version, and
  a keyed hash (HMAC) of the client IP address. These are kept for abuse handling; the IP hash is deleted after 30 days,
  and a key unused for 90 days is deleted with its data.
- Usage counters per key (for the daily quota).
- The relay's reverse proxy keeps standard access logs (IP address, time, path), rotated and deleted after at most
  90 days.

Apple and Google receive the notification text and the device token, as for any push notification.

## Contact

Questions: open an [issue](https://github.com/Rombond/tipsarr/issues). Sensitive matters: a
[private advisory](https://github.com/Rombond/tipsarr/security/advisories/new).
