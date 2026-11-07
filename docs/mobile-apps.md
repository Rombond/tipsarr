# Mobile apps

Native apps for iOS/iPadOS (SwiftUI) and Android (Kotlin, Jetpack Compose), with the same features as the web interface:
discovery, search, requests (including 4K), request progress, issues, watchlist, profiles and stats, admin approval,
and the Teasers tab when enabled on the server.

<!-- TODO: store links once published. -->
- App Store: *coming soon*
- Google Play: *coming soon*

## Connecting

Enter your server's address (for example `https://tipsarr.example.org`) and sign in with your Jellyfin username and
password. Several servers or accounts can be added. The server must be reachable from the phone: on the home network,
over a VPN, or through a reverse proxy with HTTPS.

To try the apps without a server, use the [demo](demo.md) address `https://tipsarr-demo.brebond.com`.
<!-- TODO: confirm the apps accept the demo gateway (one-click sign-in exists on the web login page only). -->

## Notifications

See [push notifications](push-notifications.md). The admin turns push on for the server; each person then allows
notifications on their phone and chooses the categories in the app's settings.

## Compatibility

The apps are built from the same API spec as the server (`api/openapi.yaml`) and tolerate fields added by newer
servers. Keep the server up to date; an app newer than the server may hide features the server does not have.

## Building them yourself

See [`ios/README.md`](../ios/README.md) and [`android/README.md`](../android/README.md). Android needs a
`google-services.json` from your own Firebase project; push to a self-built app needs your own relay (see above).
