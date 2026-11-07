# Privacy answers (App Store and Google Play)

Derived from the app code at 1.0 (iOS `ios/Tipsarr`, Android `android/app`) and [PRIVACY.md](../PRIVACY.md).

## What the code does

| Fact | iOS | Android |
|---|---|---|
| Third-party SDKs | none (only Apple's swift-openapi packages) | Firebase Cloud Messaging only (`firebase-messaging`, no Analytics, no Crashlytics) |
| Network destinations | the user's own Tipsarr server(s); APNs for the push token | the user's own Tipsarr server(s); Firebase (Installations + FCM) for the push token |
| Session tokens, server list | Keychain | private SharedPreferences encrypted with AES-GCM, key in Android Keystore (`SecureStore`) |
| Password | only if Face ID sign-in is on: Keychain, `.userPresence`, this device only | never stored |
| Other local data | recent searches, display preferences (UserDefaults / `@AppStorage`) | same (private SharedPreferences) |
| Sent to the user's server | Jellyfin username/password at sign-in, device name, app version, language, push token + chosen categories, requests, comments, issues, profile picture (if uploaded) | same |
| Push relay (`push.brebond.com`, run by the publisher) | receives the push token from the **server**, never from the app; does not store tokens (only a short hash in logs) | same |
| Analytics, ads, tracking, crash reporting | none | none |
| Backups | `allowBackup` n/a; credential items are `ThisDeviceOnly` | `android:allowBackup="false"` |

Key point: the developer does not operate the user's Tipsarr server. Data the app sends there is held by the server's
administrator (often the user), not by the developer. The only developer-operated service is the push relay, which
handles the push token in real time and does not keep it.

## App Store: App Privacy

- **Data collection: "No, we do not collect data from this app."**
  Apple defines "collect" as transmitting data off the device so that the developer or its partners can access it
  longer than needed to serve the request in real time. The app talks only to a server the user (or their admin)
  runs; the relay processes the token in real time and does not store it.
- **Tracking: No.** `NSPrivacyTracking` = false.
- Privacy policy URL: https://github.com/Rombond/tipsarr/blob/master/PRIVACY.md
- Privacy manifest: `ios/Tipsarr/Tipsarr/PrivacyInfo.xcprivacy` (added for 1.0). Required-reason APIs found in the
  code: **UserDefaults** only (reason `CA92.1`, app's own data). No file-timestamp, system-boot-time, disk-space or
  active-keyboard API is used. No collected data types, no tracking domains.

> [!WARNING]
> If Apple disagrees ("you send the push token to a server"), the fallback answer is: *Identifiers → Device ID*,
> purpose *App Functionality*, **not linked** to the user's identity, **not used for tracking**.

## Google Play: Data safety

- **Does your app collect or share any of the required user data types? Yes** (because of Firebase Messaging).
- **Is all collected data encrypted in transit? Yes** (HTTPS to Firebase; to the Tipsarr server, HTTPS unless the
  self-hoster uses plain HTTP on a LAN, see the open question below).
- **Do you provide a way for users to request that their data be deleted? Yes**: signing out deletes the push
  registration on the server; everything else lives on the user's own server, deleted by its admin. The relay keeps
  no token.

| Data type | Collected | Shared | Ephemeral | Required | Purpose |
|---|---|---|---|---|---|
| Device or other IDs (FCM token / Firebase installation ID) | Yes | No (Google acts as a service provider) | No | No (push is optional) | App functionality |

Everything else: **not collected**. Rationale: Google counts as "collection" data sent off the device to the
developer or a third party; the user's own Tipsarr server is neither (the developer has no access to it). Account
credentials, requests, comments and the profile picture go only to that server.

> [!NOTE]
> Conservative alternative if a reviewer pushes back: also declare *Personal info → Name / User IDs* and
> *Photos* (profile picture), *App activity → Other user-generated content* (requests, comments), all collected, not
> shared, purpose App functionality, optional except name/user id. It is more defensive and does not harm the listing
> much.

## Open questions

- `android:usesCleartextTraffic="true"` lets the app reach `http://` servers on a LAN. The Data safety "encrypted in
  transit" answer then depends on the user's setup. Answering "Yes" is the common practice for self-hosted clients
  (all traffic the developer controls is HTTPS); confirm you are comfortable with it.
- PRIVACY.md still has two TODOs (server deletes the device on sign-out; access-log retention of the relay). Resolve
  them before the stores read the policy.
