# API

The web interface and the mobile apps use the same HTTP API under `/api/v1`. It is described by an OpenAPI 3 document:

- served by every server at **`/api/v1/openapi.json`**;
- in the repository as [`api/openapi.yaml`](../api/openapi.yaml) (regenerated with `make generate`).

There is no bundled documentation UI; load the spec in any OpenAPI viewer.

## Authentication

| Method | Use |
|---|---|
| Session cookie `tipsarr_session` | the web interface |
| `Authorization: Bearer <token>` | the mobile apps (token from `POST /api/v1/auth/login`) |
| `X-Api-Key: <key>` | read-only dashboard key |

## Read-only API key

Settings → API key for dashboards → turn it on. The key reads everything an admin can see and can never change
anything. Send it as the `X-Api-Key` header, never in the URL (URLs end up in proxy logs). The settings never
return the key itself: the eye button reads it (`POST /api/v1/admin/secrets/api_key/reveal`, admin role only), and
**New key** (`POST /api/v1/admin/secrets/api_key/regenerate`) replaces it and shows the new one. Example:

```bash
curl -H "X-Api-Key: $KEY" https://tipsarr.example.org/api/v1/requests/counts
```

It works with Homepage, Glance and similar dashboards. "Create a new key" invalidates the old one at once.

## Live events

`GET /api/v1/events` is a Server-Sent Events stream (request progress, sync status). Reverse proxies must not buffer it.

## Webhooks

Outgoing webhooks are described in [notifications.md](notifications.md).

## Stability

The API is versioned by its path (`/v1`). Fields may be added in any release; clients should ignore unknown fields.
