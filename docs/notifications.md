# Notifications (outgoing webhooks)

Tipsarr sends notifications only as outgoing HTTP POSTs (Admin > Webhooks). Each webhook has a URL, the events it
listens to, and optionally a Liquid template, authentication, extra headers, an HMAC secret and "send once per
recipient". Templating is inspired by Seerr's Liquid webhooks (seerr-team/seerr#2039, MIT).

## Events

| Event | When | Concerned people (per-recipient) |
|---|---|---|
| `request.created` | someone requests a title | admins and people who manage requests, minus the requester |
| `request.approved` | an admin approves | the requester, minus whoever approved |
| `request.auto_approved` | the requester's rights approved it at once (sent instead of `request.approved`) | the requester and the admins |
| `request.declined` | an admin declines (the reason is appended to `message`) | the requester |
| `request.failed` | sending to Radarr/Sonarr failed, or the item vanished there | the requester |
| `media.available` | the title can be watched | the requester |
| `issue.created` | a problem is reported | admins and people who manage issues, minus the reporter |
| `issue.commented` | a comment on an issue | reporter, participants and issue managers, minus the commenter |
| `issue.resolved` | an issue is resolved | reporter and participants, minus who resolved it |
| `issue.reopened` | a resolved issue is reopened | reporter and participants, minus who reopened it |
| `test` | the Test button (ignores subscriptions; sample data: *Dune (2021)*) | the admin pressing it |

Compatibility: a webhook subscribed to `request.approved` also receives `request.auto_approved`.
No events ticked = all events. The event name is also sent in the `X-Tipsarr-Event` header.

## Default body (no template)

An empty template keeps the historical envelope:

```json
{"event": "request.created", "timestamp": "2026-11-07T12:00:00Z", "dryRun": true, "data": {"request": { "...": "..." }}}
```

`data` is `{request}` for request events, `{issue, message}` / `{issue, comment, by}` / `{issue}` for issue events.

## Templates

The body and the URL are [Liquid](https://shopify.github.io/liquid/) templates, rendered for every delivery.
Values are printed as-is: **wrap free text in the `json` filter**, which outputs a quoted, escaped JSON value
(`{{ message | json }}` gives `"Line one\nwith \"quotes\""`; a missing value gives `null`). Standard filters work
(`default`, `append`, `upcase`, `date`...). The variable `nl` holds a newline, for `append: nl`.

The URL may contain Liquid too (`https://ntfy.sh/{{ media.type }}`); after rendering it must still be `http(s)://`
and must not point at a configured Radarr/Sonarr instance (refused, like plain URLs).

A syntax error is refused when saving. Rendering errors show in **Preview** (renders without sending, any sample
event) and in **Test** (answers 422 `template_error` with the message).

### Variables

| Variable | Content |
|---|---|
| `event` | event name |
| `timestamp` | RFC 3339, UTC |
| `dryRun` | true when nothing is sent to Radarr/Sonarr |
| `subject` | short localized title, e.g. "Request approved: Dune (2021)" |
| `message` | localized sentence (plus the decline reason) |
| `image` | poster URL (TMDB, w500), empty when unknown |
| `url` | link to the title (`/media/<type>/<tmdbId>`) or the issue (`/issues/<id>`) under the **Tipsarr address for links** set on the Webhooks page; empty when that address is not set |
| `media.tmdbId`, `media.imdbId`, `media.tvdbId`, `media.type`, `media.title`, `media.year`, `media.status`, `media.is4k` | the title (imdbId/tvdbId are empty for now) |
| `request.*` | the request as in the API (`id`, `status`, `stage`, `seasons`, `requestedBy`, `declineReason`...) |
| `requestedBy.id`, `.name`, `.discordId`, `.telegramChatId` | the requester |
| `issue.*` | the issue as in the API (`id`, `kind`, `status`, `season`, `episode`, `createdBy`...) |
| `comment.message`, `comment.name`, `comment.userId` | the comment (issue.created: the report text) |
| `notifyUser.id`, `.name`, `.discordId`, `.telegramChatId`, `.language` | the recipient, only with "send once per recipient" (empty otherwise) |
| `data` | the full `data` object of the default envelope |

Sections that do not apply are empty objects (`{{ issue.title }}` prints nothing on a request event). Use
`default` for fallbacks: `{{ notifyUser.telegramChatId | default: "123456" | json }}`.

`subject` and `message` use the recipient's language when sent per recipient, otherwise the server default
language (Settings > General), falling back to English.

## Authentication and headers

- **None**, **Bearer** (`Authorization: Bearer <token>`) or **Basic** (user + password). Secrets are write-only.
- **Extra headers**: any list of name/value (e.g. `X-Gotify-Key`). They override the defaults
  (`Content-Type: application/json`, `User-Agent: Tipsarr`).
- **Signing secret** (unchanged): `X-Tipsarr-Signature: sha256=HMAC-SHA256(secret, body)` over the rendered body.

## Send once per recipient

With this option each event is delivered once per concerned person (table above) with `notifyUser` filled.
People set their **Discord user id** and **Telegram chat id** in Profile > Settings (admins can set them through
`PATCH /admin/users/{id}`). People without an id still get their delivery; templates decide what to do
(`{% if notifyUser.discordId %}...{% endif %}` or `default`).

## Presets

Presets fill the template, headers and authentication; replace the UPPER_CASE placeholders. They are served by
`GET /api/v1/admin/webhooks/presets` (source: `backend/internal/notify/presets.go`).

### Raw JSON
Empty template: the envelope above.

### Discord
URL `https://discord.com/api/webhooks/ID/TOKEN`. Embed with the poster; mentions the recipient per recipient.

```liquid
{%- capture mention -%}{% if notifyUser.discordId %}<@{{ notifyUser.discordId }}>{% endif %}{%- endcapture -%}
{
  "username": "Tipsarr",
  "content": {{ mention | json }},
  "embeds": [{
    "title": {{ subject | json }},
    "description": {{ message | json }},
    {% if url != "" %}"url": {{ url | json }},{% endif %}
    {% if image != "" %}"thumbnail": {"url": {{ image | json }}},{% endif %}
    "color": 6514417,
    "footer": {"text": {{ event | json }}},
    "timestamp": {{ timestamp | json }}
  }]
}
```

### ntfy
URL `https://ntfy.sh` (server root, JSON publishing), optional Bearer access token.

```liquid
{
  "topic": "tipsarr",
  "title": {{ subject | json }},
  "message": {{ message | json }},
  {% if url != "" %}"click": {{ url | json }},{% endif %}
  {% if image != "" %}"attach": {{ image | json }},{% endif %}
  "tags": ["clapper"]
}
```

### Gotify
URL `https://gotify.example.org/message`, header `X-Gotify-Key: APP_TOKEN`.

```liquid
{
  "title": {{ subject | json }},
  "message": {{ message | json }},
  "priority": 5,
  "extras": {
    "client::display": {"contentType": "text/markdown"}{% if url != "" %},
    "client::notification": {"click": {"url": {{ url | json }}}{% if image != "" %}, "bigImageUrl": {{ image | json }}{% endif %}}{% endif %}
  }
}
```

### Telegram
URL `https://api.telegram.org/botBOT_TOKEN/sendMessage`. Per recipient, each person's chat id is used.

```liquid
{
  "chat_id": {{ notifyUser.telegramChatId | default: "CHAT_ID" | json }},
  "text": {{ subject | append: nl | append: message | append: nl | append: url | json }},
  "disable_web_page_preview": false
}
```

### Slack
URL `https://hooks.slack.com/services/T000/B000/XXXX` (incoming webhook).

```liquid
{
  "text": {{ subject | json }},
  "blocks": [{
    "type": "section",
    "text": {"type": "mrkdwn", "text": {{ "*" | append: subject | append: "*" | append: nl | append: message | json }}}{% if image != "" %},
    "accessory": {"type": "image", "image_url": {{ image | json }}, "alt_text": {{ media.title | default: subject | json }}}{% endif %}
  }]
}
```

### Pushover
URL `https://api.pushover.net/1/messages.json`.

```liquid
{
  "token": "APP_TOKEN",
  "user": "USER_KEY",
  "title": {{ subject | json }},
  "message": {{ message | json }}{% if url != "" %},
  "url": {{ url | json }},
  "url_title": "Tipsarr"{% endif %}
}
```

### Apprise
URL `http://apprise:8000/notify/KEY` (Apprise API).

```liquid
{
  "title": {{ subject | json }},
  "body": {{ message | append: nl | append: url | json }},
  "type": "info",
  "format": "text"
}
```

### Home Assistant
URL `http://homeassistant:8123/api/webhook/WEBHOOK_ID` (webhook trigger; everything lands in `trigger.json`).

```liquid
{
  "event": {{ event | json }},
  "subject": {{ subject | json }},
  "message": {{ message | json }},
  "image": {{ image | json }},
  "url": {{ url | json }},
  "media": {{ media | json }},
  "requestedBy": {{ requestedBy | json }},
  "notifyUser": {{ notifyUser | json }}
}
```

## API

- `GET/POST /api/v1/admin/webhooks`, `PUT/DELETE /api/v1/admin/webhooks/{id}`
- `GET /api/v1/admin/webhooks/presets` (presets + subscribable events)
- `POST /api/v1/admin/webhooks/preview?event=...&id=...` (renders an unsaved form; masks secrets)
- `POST /api/v1/admin/webhooks/{id}/test` (204 delivered, 422 template error, 502 receiver failure or Radarr/Sonarr URL)
- `PUT /api/v1/admin/settings` `appPublicUrl` (base of `url`)
- `PATCH /api/v1/me` / `PATCH /api/v1/admin/users/{id}`: `discordId`, `telegramChatId`
