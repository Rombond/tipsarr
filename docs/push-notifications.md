# Push notifications

The iOS and Android apps can show notifications when a request is approved, declined, fails or becomes available, when
someone requests a title (for people who approve requests) and when an issue is reported, commented or resolved.

## Why a relay

Apple (APNs) and Google (FCM) deliver a notification to an app only when it is signed with the credentials of that
app's publisher. A self-hosted Tipsarr server does not have them, and they cannot be handed out. So the publisher runs a
small relay, [push-relay](https://github.com/Rombond/push-relay) (MIT), at `https://push.brebond.com`:

```
Tipsarr server ──HTTPS, server key──► push-relay ──► APNs (iOS) / FCM (Android) ──► phone
       ▲                                                                              │
       └───────────────── the app opens and fetches the details from your server ◄────┘
```

## Turning it on

Settings → Push notifications → **Turn on mobile notifications**. The server registers once with the relay and gets a
key of its own (it identifies itself with a random install id it generates and keeps). Then each person allows
notifications in the app and chooses which kinds they want (their requests, requests to approve, issues).

Nothing is sent while `TIPSARR_DRY_RUN` is on: the log only says which push would have gone out.

You can also paste a key given by a relay operator instead of registering.

## What goes through the relay

For each notification Tipsarr sends: the device's push token, the platform, the event name (e.g. `request.approved`),
an id (request or issue number), the language (`en`, `fr`) and the host name the app uses for your server (so a phone
with several accounts opens the right one). At registration it sends the install id, your server's public Tipsarr
address if you set one for links, and the Tipsarr version.

The relay builds the text from its own templates, for example "Your request was approved". **No title, user name,
comment or poster** goes through the relay, Apple or Google. A key from the public relay can never send free text.

The relay does not store device tokens and logs only a short hash of them. It keeps the server keys (hashed), usage
counters, the install id, the optional server address and version, and a keyed hash of the registering IP address
(dropped after 30 days by default). See [PRIVACY.md](../PRIVACY.md).

## Quota

A key from the public relay has a **daily quota** of notifications (shown in Settings). When it is reached, Tipsarr
pauses push until the next UTC midnight and Settings shows until when; webhooks keep working. A key unused for 90 days
is deleted by the relay; Tipsarr registers again on its own when the relay refuses its key.
<!-- TODO: confirm the public relay's actual quota and unused-key settings (relay defaults: 200 per day, 90 days, 30 days). -->

## Running your own relay

You can run push-relay yourself (see its README), but **the apps from the App Store and Google Play only work with the
official relay**: a notification for those apps must be signed with the publisher's Apple key and Firebase project.
Your own relay is useful only with your own build of the apps (your bundle id, Apple push key and Firebase project), or
for other apps. In Tipsarr, set the relay address and key under Settings → Push notifications.

With your own relay and a manually issued key, you can turn on "Name the title in alerts" if the relay allows free text
for the app: the title then goes through the relay and Apple/Google.

## Alternatives

Do not want any third party involved? Leave push off and use [webhooks](notifications.md) to a self-hosted ntfy or
Gotify, or to Pushover, Telegram, Discord and others.
