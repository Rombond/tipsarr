# Mobile apps

Native apps for iOS/iPadOS (SwiftUI) and Android (Kotlin, Jetpack Compose), with the same features as the web interface:
discovery, search, requests (including 4K), request progress, issues, watchlist, profiles and stats, admin approval,
and the Teasers tab when enabled on the server.

- **iOS and iPadOS** (App Store): in review, link coming with the approval.
- **Android**: signed APK attached to each [GitHub release](https://github.com/Rombond/tipsarr/releases/latest)
  (`tipsarr-<version>.apk`). Add this repository to [Obtainium](https://github.com/ImranR98/Obtainium) to get updates.
  Not on Google Play for now (internal testing only).

## Connecting

Enter your server's address (for example `https://tipsarr.example.org`) and sign in with your Jellyfin username and
password. Several servers or accounts can be added. The server must be reachable from the phone: on the home network,
over a VPN, or through a reverse proxy with HTTPS.

To try the apps without a server, use the [demo](demo.md) address `https://tipsarr-demo.brebond.com` and sign in
as `admin` / `demo` (admin rights) or `user` / `demo` (plain user). Signing in starts your own sandbox; when it ends
(30 minutes idle, 60 minutes at most) the app returns to its sign-in screen. Push notifications are off in the demo.

## Language and region

The apps show titles and descriptions in the phone's language. The iOS app also takes the phone's region (release
dates, box office) until you pick one in Settings > Region: after you change the phone's language and region, the
next launch uses the new ones. A region you picked (in the app, on the web or on Android) is kept.

## Notifications

See [push notifications](push-notifications.md). The admin turns push on for the server; each person then allows
notifications on their phone and chooses the categories in the app's settings.

## Compatibility

The apps are built from the same API spec as the server (`api/openapi.yaml`) and tolerate fields added by newer
servers. Keep the server up to date; an app newer than the server may hide features the server does not have.

## Building them yourself

See [`ios/README.md`](../ios/README.md) and [`android/README.md`](../android/README.md). Android needs a
`google-services.json` from your own Firebase project; push to a self-built app needs your own relay (see above).
