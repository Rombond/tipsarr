# SuggestArr API Reference

This document describes the SuggestArr backend REST API (Flask, `api_service/`). It is generated from the blueprint route handlers and their docstrings; no OpenAPI/Swagger spec exists in this repo yet.

## Conventions

- **Base URL prefix:** all routes are mounted under `/api` (each blueprint below lists its specific prefix).
- **Auth:** unless a route is explicitly listed as public, it requires a `Authorization: Bearer <jwt>` header. The JWT is obtained via `POST /api/auth/login` (or `/api/auth/refresh`). Routes annotated `@require_role('admin')` additionally require the authenticated user to have the `admin` role.
- **Content type:** requests with a body use `application/json`; all responses are JSON.
- **Common envelope:** most (not all) endpoints return `{"status": "success" | "error", ...}`; some legacy/simpler endpoints return bare objects/arrays — see each endpoint for the exact shape.
- **Rate limiting:** several sensitive endpoints (login, force-run, dry-run, etc.) are rate-limited per IP; limits are noted per endpoint where applicable.

## Blueprints

| Blueprint | Prefix | Purpose |
|---|---|---|
| [Auth](#auth-apiauth) | `/api/auth` | Login, session/JWT refresh, account setup & self-service profile |
| [Users](#users-apiusers) | `/api/users` | Admin user management + per-user media-account linking |
| [Admin](#admin-apiadmin) | `/api/admin` | Config backup/restore (export/import) |
| [Jobs](#jobs-apijobs) | `/api/jobs` | Discover/recommendation job CRUD, execution, suggestions workflow, metadata lookups |
| [Automation](#automation-apiautomation) | `/api/automation` | Legacy global automation trigger, workflow request review, request stats |
| [Jellyfin](#jellyfin-apijellyfin) | `/api/jellyfin` | Jellyfin server connectivity (libraries, users, connection test) |
| [Plex](#plex-apiplex) | `/api/plex` | Plex server connectivity + OAuth login flow |
| [Seer](#seer-apiseer) | `/api/seer` | Overseerr/Jellyseerr connectivity, login, Radarr/Sonarr server info |
| [Config](#config-apiconfig) | `/api/config` | Application configuration (env vars, sections, setup status, log level, backups, per-user OpenAI keys) |
| [TMDB](#tmdb-apitmdb) | `/api/tmdb` | TMDB proxy endpoints (popular, genres, languages, watch providers) + cache management |
| [OMDb](#omdb-apiomdb) | `/api/omdb` | OMDb API key validation |
| [Trakt](#trakt-apitrakt) | `/api/trakt` | Trakt account linking (device-code OAuth) for media users, per-user or admin-managed |
| [AI Search](#ai-search-apiai-search) | `/api/ai-search` | LLM-powered semantic search, feedback, request-from-search |
| [Cleanup](#cleanup-apicleanup) | `/api/cleanup` | Automated library cleanup settings, manual run, audit log |
| [Logs](#logs-api) | `/api` (route `/logs`) | Tail application log file |
| [Health](#health-apihealth) | `/api/health` | Liveness/readiness probes |
| [Integrations](#integrations-apiintegrations) | `/api/integrations` | Self-service linking of Jellyfin/Emby/Plex accounts via username+password |

---

## Auth (`/api/auth`)

#### `GET /api/auth/status`
Returns setup state and current-auth status so the SPA can decide which screen to show.

**Auth:** None (public). Rate-limit exempt.
**Request body / query params:** None
**Response:**
```json
{
  "auth_setup_complete": true,
  "app_setup_complete": true,
  "allow_registration": false,
  "authenticated": true,
  "username": "alice",
  "bypass": true
}
```
`username`/`bypass` only present when `authenticated` is true; `bypass` only when using local_bypass/disabled auth modes.
**Errors:** None (fails open with 200 and default-false values on internal error)

#### `POST /api/auth/setup`
Creates the very first admin account. Self-guards via existing user count.

**Auth:** None (public). Rate limit: 5/hour per IP.
**Request body:**
| Name | Type | Required | Description |
|---|---|---|---|
| username | str | yes | ≤64 chars |
| password | str | yes | ≥ `MIN_PASSWORD_LENGTH` chars |

**Response:** `201` `{ "message": "Admin account created" }`
**Errors:** `400` validation error; `403` setup already completed

#### `POST /api/auth/login`
Authenticates with username/password; issues JWT access token and sets httpOnly refresh cookie.

**Auth:** None (public). Rate limit: 10/minute per IP.
**Request body:** `username` (str, required), `password` (str, required)
**Response:** `200` `{ "access_token": "<jwt>", "role": "<role>", "username": "<username>" }` + `Set-Cookie: suggestarr_refresh` (httpOnly, SameSite=Strict, path=`/api/auth/refresh`)
**Errors:** `400` missing fields; `401` invalid credentials; `403` account disabled

#### `POST /api/auth/refresh`
Issues a new short-lived JWT using the httpOnly refresh cookie.

**Auth:** Relies on refresh cookie (public route). Rate limit: 30/minute per IP.
**Request body / query params:** None (reads `suggestarr_refresh` cookie)
**Response:** `200` `{ "access_token": "<jwt>" }`
**Errors:** `401` missing/invalid/expired refresh token, or user not found/disabled

#### `POST /api/auth/logout`
Revokes the current refresh token and clears the cookie.

**Auth:** JWT required
**Request body / query params:** None
**Response:** `200` `{ "message": "Logged out" }` (also clears cookie)

#### `POST /api/auth/register`
Self-registration endpoint, gated by `ALLOW_REGISTRATION` config flag.

**Auth:** Public, gated by config flag. Rate limit: 5/hour per IP.
**Request body:** `username` (str, required, ≤64 chars), `password` (str, required, ≥`MIN_PASSWORD_LENGTH`)
**Response:** `201` `{ "message": "Account created" }`
**Errors:** `400` validation error; `403` registration disabled; `409` username taken

#### `GET /api/auth/me`
Returns the authenticated user's identity.

**Auth:** JWT required
**Response:** `200` `{ "id": "<user_id>", "username": "<name>", "role": "<role>" }`

#### `PATCH /api/auth/me`
Updates the authenticated user's own username and/or password.

**Auth:** JWT required
**Request body:** `username` (str, no), `current_password` (str, required if `new_password` given), `new_password` (str, no, ≥`MIN_PASSWORD_LENGTH`). At least one field required.
**Response:** `200` `{ "message": "Profile updated" }` (or with a re-issued `access_token` when username changed)
**Errors:** `400` validation/no changes; `401` wrong current password; `409` username taken

---

## Users (`/api/users`)

#### `GET /api/users`
List all SuggestArr accounts (password hashes excluded).

**Auth:** `@require_role('admin')`
**Response:** `200` list of `{ id, username, role, is_active, created_at, last_login }`

#### `POST /api/users`
Admin-creates a new SuggestArr account.

**Auth:** `@require_role('admin')`
**Request body:** `username` (str, required, ≤64 chars), `password` (str, required), `role` (str, no, `admin`|`user`, default `user`)
**Response:** `201` `{ "id": <int>, "username": <str>, "role": <str> }`
**Errors:** `400` validation error; `409` username taken

#### `PATCH /api/users/<int:user_id>`
Updates a user's role, active status, and/or permission fields. Admins cannot modify their own account; last active admin cannot be demoted/deactivated.

**Auth:** `@require_role('admin')`
**Request body (all optional, ≥1 required):** `role` (`admin`|`user`), `is_active` (bool), `can_manage_ai` (bool), `seer_user_id` (int|null), `allowed_tabs`/`visible_tabs` (comma string or array)
**Response:** `200` `{ "message": "User updated" }`
**Errors:** `400` validation; `403` self-modify or last-admin guard; `404` not found

#### `POST /api/users/<int:user_id>/permissions`
Updates a user's permission flags and visible tab list.

**Auth:** `@require_role('admin')`
**Request body:** `can_manage_ai` (bool, no), `allowed_tabs` (array/string, no), `visible_tabs` (legacy alias, no)
**Response:** `200` `{ "success": true, "message": "Permissions updated", "visible_tabs": [...], "can_manage_ai": false }`
**Errors:** `400` invalid payload; `404` not found

#### `DELETE /api/users/<int:user_id>`
Permanently deletes a SuggestArr account. Admins cannot delete their own account; last active admin cannot be deleted.

**Auth:** `@require_role('admin')`
**Response:** `200` `{ "message": "User deleted" }`
**Errors:** `403` guard rails; `404` not found

#### `POST /api/users/<int:user_id>/link/<provider>`
Admin-forces a media server username map for any user profile. `provider` ∈ `jellyfin`, `plex`, `emby`.

**Auth:** `@require_role('admin')`
**Request body:** `external_user_id` (str, required), `external_username` (str, required)
**Response:** `200` `{ "message": "<Provider> account linked for user" }`
**Errors:** `400` invalid provider/missing fields; `404` user not found

#### `GET /api/users/me/links`
Returns the current user's linked external media accounts (access tokens excluded).

**Auth:** Authenticated
**Response:** `200` list of `{ id, provider, external_username, created_at }`

#### `POST /api/users/me/link/jellyfin`
Persists the current user's selected Jellyfin account link.

**Auth:** Authenticated
**Request body:** `external_user_id` (str, required), `external_username` (str, required)
**Response:** `200` `{ "message": "Jellyfin account linked", "external_username": <str> }`
**Errors:** `400` missing fields

#### `POST /api/users/me/link/emby`
Same as Jellyfin link, for Emby.

**Auth:** Authenticated
**Request body:** `external_user_id` (str, required), `external_username` (str, required)
**Response:** `200` `{ "message": "Emby account linked", "external_username": <str> }`
**Errors:** `400` missing fields

#### `GET /api/users/me/link/<provider>/users`
Returns the user list from the configured Jellyfin/Emby server. `provider` ∈ `jellyfin`, `emby`.

**Auth:** Authenticated
**Response:** `200` list of `{ "id": str, "name": str }`
**Errors:** `400` unsupported provider; `503` media server not configured; `502` unreachable server

#### `DELETE /api/users/me/link/<provider>`
Removes the current user's link to an external media account. `provider` ∈ `jellyfin`, `plex`, `emby`.

**Auth:** Authenticated
**Response:** `200` `{ "message": "Account unlinked" }`
**Errors:** `400` invalid provider; `404` no linked account

#### `GET /api/users/me/link/plex/oauth-start`
Begins the Plex OAuth device flow by generating a one-time PIN via plex.tv.

**Auth:** Authenticated
**Response:** `200` `{ "pin_id": <int>, "auth_url": <str> }`
**Errors:** `502` unreachable Plex.tv

#### `POST /api/users/me/link/plex/oauth-poll`
Polls plex.tv to check whether the user has authorized the PIN; stores the link when authorized.

**Auth:** Authenticated
**Request body:** `pin_id` (int, required)
**Response:** `200` `{ "status": "pending" }` or `{ "status": "linked", "external_username": <str> }`
**Errors:** `400` missing `pin_id`; `502` unreachable Plex.tv or could not retrieve user info

---

## Admin (`/api/admin`)

#### `GET /api/admin/export-config`
Exports the current DB-backed configuration as a portable backup.

**Auth:** Public for redacted export; admin role required (checked manually) when `include_secrets=true`
**Query params:** `include_secrets` (str `"true"`/`"false"`, default `false`)
**Response:** `200` `{ "schema_version": 2, "integrations": {...}, "settings": {...} }` (secrets redacted unless admin + `include_secrets=true`)
**Errors:** `403` non-admin requesting secrets

#### `POST /api/admin/import-config`
Imports a configuration backup (v2 schema, or legacy v1 flat env-var dict).

**Auth:** `@require_role('admin')`
**Request body:** `schema_version` (int, no — omit for legacy v1), `integrations` (dict, required for v2: service name → config dict), `settings` (dict, ignored on import)
**Response:** `200` `{ "status": "success", "services_written": [<str>, ...], "count": <int> }`
**Errors:** `400` malformed payload or unsupported schema version

---

## Jobs (`/api/jobs`)

#### `GET /api/jobs`
Get all discover jobs. Admins see all jobs; normal users only their own.

**Auth:** Authenticated
**Response:** `200` `{ "status": "success", "jobs": [ {..., "next_run": "..."} ], "request_approval_default": false }`
**Errors:** `500` internal error

#### `GET /api/jobs/suggestions`
List suggestions (media items pending/approved/etc.) for the current user, paginated and searchable.

**Auth:** Authenticated
**Query params:** `status` (default `awaiting_approval`; one of `awaiting_approval`,`queued`,`submitting`,`submitted`,`rejected`,`failed`), `page` (default 1), `per_page` (default 24, max 100), `search` (≤100 chars)
**Response:** `200` `{ "status": "success", "items": [...], "total": 0, "page": 1, "pages": 1 }`
**Errors:** `400` invalid status/pagination

#### `POST /api/jobs/suggestions/approve`
Approve one or more suggestions.

**Auth:** Authenticated; rate limit 20/min
**Request body:** `ids` (array of int, 1–100, required)
**Response:** `200` `{ "status": "success", "updated": <count> }`
**Errors:** `400` invalid `ids`

#### `POST /api/jobs/suggestions/blacklist`
Blacklist (and reject) one or more suggestions.

**Auth:** Authenticated; rate limit 20/min
**Request body:** `ids` (array of int, 1–100, required)
**Response:** `200` `{ "status": "success", "updated": <count> }`

#### `POST /api/jobs/suggestions/reject`
Reject one or more suggestions.

**Auth:** Authenticated; rate limit 20/min
**Request body:** `ids` (array of int, 1–100, required)
**Response:** `200` `{ "status": "success", "updated": <count> }`

#### `POST /api/jobs/suggestions/retry`
Retry one or more previously failed suggestions.

**Auth:** Authenticated; rate limit 20/min
**Request body:** `ids` (array of int, 1–100, required)
**Response:** `200` `{ "status": "success", "updated": <count> }`

#### `GET /api/jobs/suggestions/blacklist`
List all blacklisted media items.

**Auth:** `@require_role('admin')`
**Response:** `200` `{ "status": "success", "items": [...] }`

#### `DELETE /api/jobs/suggestions/blacklist/<media_type>/<tmdb_id>`
Remove an item from the blacklist ("restore"). `media_type` ∈ `movie`,`tv`.

**Auth:** `@require_role('admin')`
**Response:** `200` `{ "status": "success", "removed": <bool/count> }`
**Errors:** `400` invalid media_type

#### `GET /api/jobs/<int:job_id>`
Get a single discover job by ID. Admins can view any job; normal users only their own.

**Auth:** Authenticated
**Response:** `200` `{ "status": "success", "job": {..., "next_run": "..."} }`
**Errors:** `404` not found/not owned

#### `POST /api/jobs`
Create a new job (`discover`, `recommendation`, or `trakt_recommendations`). `owner_id` is always set server-side.

**Auth:** Authenticated
**Request body:**
| Name | Type | Required | Description |
|---|---|---|---|
| name | string | yes | Job name |
| job_type | string | no (default `discover`) | `discover`, `recommendation`, `trakt_recommendations` |
| media_type | string | yes | `movie`/`tv` for discover; `movie`/`tv`/`both` otherwise |
| filters | object | yes | Filter dictionary (see [Recommendation job filters](#recommendation-job-filters) below) |
| schedule_type | string | yes | `preset` or `cron` |
| schedule_value | any | yes | Schedule value |
| max_results | int | no (default 20) | Max total requests per run |
| user_ids | list | required (exactly 1) for `trakt_recommendations` | Linked media user IDs |
| enabled | bool | no (default true) | Whether to schedule immediately |
| delivery_mode | string | no (default `inherit`) | `inherit`, `automatic`, `manual` |
| approval_pause_mode | string | no (default `inherit`) | `inherit`, `always`, `never` |
| request_profiles | object | no | Per-media (`movie`/`tv`) dict: `serverId`(int), `profileId`(int), `rootFolder`(str), `is4k`(bool), `languageProfileId`(int) |
| seer_identity_mode | string | no (default `technical_user`) | `matching_user`, `technical_user`, `admin_user` (admin_user requires admin owner) |
| unwatched_suggestion_days | int | no (default 7) | Must be positive |

**Response:** `201` `{ "status": "success", "job_id": <id> }`
**Errors:** `400` missing/invalid fields; `403` `admin_user` identity requires admin owner

##### Recommendation job filters
Common keys read by `recommendation_automation.py` (all optional, fall back to global config):
`max_similar_movie`, `max_similar_tv` (per-seed similar-content cap — **note:** the dry-run/Preview endpoint does not currently enforce this cap, see project notes), `max_content` (seeds to check per user), `search_size`, `vote_average_gte`, `vote_count_gte`, `request_first_season_only`, `exclude_downloaded`, `exclude_requested`, `min_seasons`, `honor_seer_discovery`/`honor_jellyseer_discovery`, `use_trakt_as_seed`, `use_trakt_as_exclusion`, `with_original_language`/`language`, `without_genres`, `watch_region`, `with_watch_providers`, `min_runtime`, `include_tvod`, `rating_source`, `imdb_rating_gte`, `imdb_min_votes`, `use_llm`, `only_first_movie_in_collection`, year-range keys (`release_year_gte`/`year_from`/`primary_release_date_gte`/`first_air_date_gte` and the `_lte`/`_to` equivalents).

#### `PUT /api/jobs/<int:job_id>`
Update an existing job. Admins can update any job; normal users only their own. Accepts any subset of the create-job fields (same validation applied when present).

**Auth:** Authenticated
**Response:** `200` `{ "status": "success", "message": "Job updated" }`
**Errors:** `404` not found; `403` insufficient permissions; `400` invalid fields

#### `DELETE /api/jobs/<int:job_id>`
Delete a job (execution history cascades). Admins can delete any job; normal users only their own.

**Auth:** Authenticated
**Response:** `200` `{ "status": "success", "message": "Job deleted" }`
**Errors:** `404` not found; `403` insufficient permissions

#### `POST /api/jobs/<int:job_id>/toggle`
Toggle a job's enabled status (and (un)schedule it accordingly).

**Auth:** Authenticated
**Response:** `200` `{ "status": "success", "enabled": <bool>, "message": "Job enabled|disabled" }`
**Errors:** `404` not found; `403` insufficient permissions

#### `POST /api/jobs/<int:job_id>/run`
Execute a job immediately, synchronously within the request. Admins can run any job; normal users only their own.

**Auth:** Authenticated; rate limit 5/min
**Response:**
- `200` `{ "status": "success", "message": "Job executed successfully", "results_count": n, "requested_count": n }`
- `200` `{ "status": "paused", "message": "...", "results_count": 0, "requested_count": 0 }` (pending Seer approvals)
- `500` `{ "status": "error", "message": "..." }`

**Errors:** `404` not found; `403` insufficient permissions

#### `POST /api/jobs/<int:job_id>/dry-run`
Simulate job execution (runs the full TMDb discovery/filter pipeline) without enqueueing to Seer or writing history. **Note:** unlike a real run, the dry-run branch of `request_similar_media()` does not currently cap results by `max_similar_movie`/`max_similar_tv` — it marks every filter-passing candidate as `would_request` up to the job's Total Request Limit (`max_results`), so Preview counts can look higher than an actual run would produce.

**Auth:** Authenticated; rate limit 10/min
**Response:** `200` `{ "status": "success", "dry_run": true, "items_count": n, "items": [...] }`
**Errors:** `404` not found; `403` insufficient permissions

#### `POST /api/jobs/run-all`
Execute all enabled jobs immediately in a background thread; returns immediately.

**Auth:** `@require_role('admin')`; rate limit 5/min
**Response:** `202` `{ "status": "success", "message": "Running N job(s) in the background.", "jobs_count": N }`
**Errors:** `409` already running; `404` no enabled jobs

#### `GET /api/jobs/<int:job_id>/history`
Get execution history for a specific job (non-admin restricted to own jobs; returns 404 rather than 403 to avoid leaking existence).

**Auth:** Authenticated
**Query params:** `limit` (int, default 50)
**Response:** `200` `{ "status": "success", "job_name": "...", "history": [...] }`
**Errors:** `404` not found/not owned

#### `GET /api/jobs/history`
Get recent execution history across all jobs (filtered to owned jobs for non-admin).

**Auth:** Authenticated
**Query params:** `limit` (int, default 100)
**Response:** `200` `{ "status": "success", "history": [...] }`

#### `GET /api/jobs/genres/<media_type>`
Get available TMDb genres for a media type. `media_type` ∈ `movie`,`tv`,`both`.

**Auth:** Authenticated
**Response:** `200` `{ "status": "success", "genres": [...] }`
**Errors:** `400` invalid media_type

#### `GET /api/jobs/languages`
Get available languages from TMDb.

**Auth:** Authenticated
**Response:** `200` `{ "status": "success", "languages": [...] }`

#### `GET /api/jobs/watch-regions`
Get available watch-provider regions from TMDb.

**Auth:** Authenticated
**Response:** `200` `{ "status": "success", "regions": [ {"iso_3166_1": "...", "english_name": "..."} ] }`

#### `GET /api/jobs/watch-providers`
Get available streaming providers for a region from TMDb.

**Auth:** Authenticated
**Query params:** `region` (string, required, ISO 3166-1 code, e.g. `IT`)
**Response:** `200` `{ "status": "success", "providers": [ {"provider_id": "...", "provider_name": "..."} ] }`
**Errors:** `400` missing region

#### `GET /api/jobs/defaults`
Return default filter values for new jobs, derived from global content-filter config.

**Auth:** Authenticated
**Response:**
```json
{
  "status": "success",
  "defaults": {
    "vote_average_gte": 6.0,
    "vote_count_gte": null,
    "request_first_season_only": false
  }
}
```
> **Note:** this endpoint currently does not return `max_similar_movie`, `max_similar_tv`, or `search_size` defaults, so brand-new job forms in the UI don't pre-populate those sliders from global config (see project notes / `issue.md`).

#### `GET /api/jobs/queue-status`
Return the current Seer delivery queue status.

**Auth:** Authenticated
**Response:** `200` `{ "status": "success", "queued": 0, "submitting": 0, "submitted": 0, "failed": 0, "total_pending": 0 }`

#### `GET /api/jobs/llm-status`
Check if LLM is configured and available for AI-enhanced recommendations.

**Auth:** Authenticated
**Response:** `200` `{ "status": "success", "configured": <bool>, "advanced_algorithm_enabled": <bool> }`

#### `POST /api/jobs/sync-ai-setting`
Sync `use_llm` flag across all recommendation jobs based on `ENABLE_ADVANCED_ALGORITHM` config.

**Auth:** `@require_role('admin')`
**Response:** `200` `{ "status": "success", "use_llm": <bool>, "updated_jobs": <count> }`

#### `POST /api/jobs/llm-test`
Test the LLM connection with a real API call.

**Auth:** `@require_role('admin')`; rate limit 5/min
**Request body:** `OPENAI_API_KEY` (required unless `OPENAI_BASE_URL` given, e.g. Ollama), `OPENAI_BASE_URL` (no), `LLM_MODEL` (no, default `gpt-4o-mini`)
**Response:** `200` `{ "status": "success", "message": "Connection successful!" }`
**Errors:** `400` missing key or connection test failed

#### `POST /api/jobs/import-config`
One-time import of YAML config settings into the database (only if no jobs exist yet and `CRON_TIMES` configured).

**Auth:** `@require_role('admin')`
**Response:** `200` `{ "status": "success", "job_id": <id>, "message": "Config imported to database" }` or a skipped/failed message

---

## Automation (`/api/automation`)

> **Note:** `POST /api/automation/force_run` uses the legacy `ContentAutomation` engine, which reads similarity/content limits (`max_similar_movie`, `max_similar_tv`, `max_content`, `search_size`, etc.) **only from global env/config**, not from any per-job `filters`. It is unrelated to the per-job scheduler (`RecommendationAutomation`) used by `/api/jobs/<id>/run` and `/api/jobs/<id>/dry-run`.

#### `GET /api/automation/requests/workflow`
List "workflow" suggestions (paginated, filterable) for the current user, with visibility restricted per `REQUEST_VISIBILITY` config.

**Auth:** Authenticated
**Query params:** `status` (default `awaiting_approval`; `all`,`awaiting_approval`,`queued`,`submitting`,`submitted`,`rejected`,`failed`,`blacklisted`), `page` (default 1), `per_page` (default 24, max 100), `search` (≤100 chars), `media_type` (default `all`; `all`,`movie`,`tv`), `user_id` (admin, or own linked profile if `REQUEST_VISIBILITY=own`)
**Response:** `200` `{ "status": "success", "items": [...], "total": 0, "page": 1, "pages": 1 }`
**Errors:** `400` invalid params

#### `POST /api/automation/requests/workflow/approve`
Approve one or more workflow suggestions.

**Auth:** Authenticated; rate limit 20/min
**Request body:** `ids` (array of int, 1–100, required)
**Response:** `200` `{ "status": "success", "updated": <count> }`

#### `POST /api/automation/requests/workflow/reject`
Reject one or more workflow suggestions.

**Auth:** Authenticated; rate limit 20/min
**Request body:** `ids` (array of int, 1–100, required)
**Response:** `200` `{ "status": "success", "updated": <count> }`

#### `POST /api/automation/requests/workflow/blacklist`
Blacklist (and reject) one or more workflow suggestions.

**Auth:** Authenticated; rate limit 20/min
**Request body:** `ids` (array of int, 1–100, required)
**Response:** `200` `{ "status": "success", "updated": <count> }`

#### `POST /api/automation/requests/workflow/retry`
Retry one or more failed workflow suggestions.

**Auth:** Authenticated; rate limit 20/min
**Request body:** `ids` (array of int, 1–100, required)
**Response:** `200` `{ "status": "success", "updated": <count> }`

#### `POST /api/automation/requests/workflow/request-again`
Re-request previously rejected (optionally blacklisted) suggestions.

**Auth:** Authenticated; rate limit 20/min
**Request body:** `ids` (array of int, 1–100, required), `remove_blacklist` (bool, no, default false)
**Response:** `200` `{ "status": "success", "updated": <count> }`

#### `POST /api/automation/force_run`
Execute the full legacy content automation process (`ContentAutomation`) in a background thread; returns immediately.

**Auth:** `@require_role('admin')`; rate limit 5/min
**Response:** `202` `{ "status": "success", "message": "Task started in the background!" }`
**Errors:** `409` already running

#### `GET /api/automation/requests`
Get all automation requests grouped by source, paginated and sorted.

**Auth:** Authenticated
**Query params:** `page` (default 1), `per_page` (default 8), `sort_by` (default `date-desc`; `date-desc`,`date-asc`,`title-asc`,`title-desc`,`rating-desc`,`rating-asc`)
**Response:** `200` grouped-by-source dict
**Errors:** `500` `{ "error": "An internal error occurred" }`

#### `GET /api/automation/requests/ai-search`
Get requests that originated from AI Search, paginated and sorted.

**Auth:** Authenticated
**Query params:** `page` (default 1), `per_page` (default 12), `sort_by` (default `date-desc`; `date-desc`,`date-asc`,`title-asc`,`title-desc`)
**Response:** `200` dict
**Errors:** `500` `{ "error": "An internal error occurred" }`

#### `GET /api/automation/requests/stats`
Get statistics for automation requests.

**Auth:** Authenticated
**Response:** `200` stats dict
**Errors:** `500` `{ "error": "An internal error occurred" }`

---

## Jellyfin (`/api/jellyfin`)

#### `GET/POST /api/jellyfin/libraries`
Fetch Jellyfin libraries using globally configured credentials, or credentials supplied in the request body.

**Auth:** None
**Request body (optional):** `JELLYFIN_API_URL`, `JELLYFIN_TOKEN` — fall back to stored config if omitted
**Response:** `200` `{ "message": "Libraries fetched successfully", "items": [...] }`
**Errors:** `400` not configured / invalid URL (SSRF guard); `404` none found

#### `GET /api/jellyfin/test`
Test the Jellyfin server connection using globally configured API key and URL.

**Auth:** None
**Response:** `200` `{ "message": "Jellyfin connection successful!", "status": "success", "data": { "libraries_count": 3, "server_url": "..." } }`
**Errors:** `400` not configured / invalid URL / connection or token invalid

#### `GET/POST /api/jellyfin/users`
Fetch Jellyfin users using globally configured credentials, or credentials supplied in the request body.

**Auth:** None
**Request body (optional):** `JELLYFIN_API_URL`, `JELLYFIN_TOKEN`
**Response:** `200` `{ "message": "Users fetched successfully", "users": [...] }`
**Errors:** `400` not configured / invalid URL; `404` none found

---

## Plex (`/api/plex`)

#### `POST /api/plex/libraries`
Fetch Plex libraries using the provided API key and server URL.

**Auth:** None
**Request body:** `PLEX_API_URL` (required), `PLEX_TOKEN` (required)
**Response:** `200` `{ "message": "Libraries fetched successfully", "items": [...] }`
**Errors:** `400` missing/invalid URL; `404` none found

#### `POST /api/plex/auth`
Start Plex OAuth flow: obtains a PIN and auth URL from Plex.tv.

**Auth:** None
**Response:** `200` `{ "pin_id": "1234", "auth_url": "https://app.plex.tv/auth#?..." }`

#### `POST /api/plex/callback`
Check whether a Plex PIN-based login has completed and retrieve the resulting auth token.

**Auth:** None
**Request body:** `pin_id` (required)
**Response:** `200` `{ "auth_token": "abc123" }`
**Errors:** `401` `{ "error": "Authentication failed" }` if not yet available

#### `POST /api/plex/api/v1/auth/plex`
Log in with an already-obtained Plex auth token.

**Auth:** None
**Request body:** `authToken` (required)
**Response:** `200` `{ "message": "Login success", "auth_token": "abc123" }`
**Errors:** `401` `{ "error": "Invalid token" }`

#### `GET /api/plex/check-auth/<int:pin_id>`
Check if a Plex login (by PIN id) has completed.

**Auth:** None
**Response:** `200` `{ "auth_token": "abc123" }` or `{ "auth_token": null }` (always 200)

#### `POST /api/plex/servers`
Find all Plex servers available to the authenticated Plex user.

**Auth:** None (requires Plex `auth_token` in body)
**Request body:** `auth_token` (required)
**Response:** `200` `{ "message": "Plex servers fetched successfully", "servers": [...] }`
**Errors:** `400` missing token; `404` none found

#### `POST /api/plex/test`
Test Plex server connection using the provided token and URL.

**Auth:** None
**Request body:** `token` (required), `api_url` (required)
**Response:** `200` `{ "message": "Plex connection successful!", "status": "success", "data": { "libraries_count": 5, "server_url": "..." } }`
**Errors:** `400` missing/invalid, or connection/token invalid

#### `POST /api/plex/users`
Fetch Plex users using the provided API token (and optional server URL).

**Auth:** None
**Request body:** `PLEX_TOKEN` (required), `PLEX_API_URL` (no)
**Response:** `200` `{ "message": "Users fetched successfully", "users": [...] }`
**Errors:** `400` missing/invalid; `404` none found

---

## Seer (`/api/seer`)

#### `GET/POST /api/seer/get_users`
Fetch Seer (Overseerr/Jellyseerr) users using globally configured credentials, or credentials supplied in the request body.

**Auth:** None
**Request body (optional):** `SEER_API_URL`, `SEER_TOKEN`, `SEER_SESSION_TOKEN`
**Response:** `200` `{ "message": "Users fetched successfully", "users": [...] }`
**Errors:** `400` not configured/invalid URL; `404` failed to fetch

#### `GET/POST /api/seer/test`
Test Seer API connection.

**Auth:** None
**Request body (optional):** `SEER_API_URL`, `SEER_TOKEN`, `SEER_SESSION_TOKEN`
**Response:** `200` `{ "message": "Seer connection successful!", "status": "success", "data": { "users_count": 4, "server_url": "...", "http_status": 200 } }`
**Errors:** `400` not configured/invalid URL/test failed/auth failed

#### `POST /api/seer/login`
Log in to Seer with username/password to obtain a session token.

**Auth:** None
**Request body:** `SEER_API_URL` (required), `SEER_TOKEN` (no), `SEER_USER_NAME` (required), `SEER_PASSWORD` (required)
**Response:** `200` `{ "message": "Login successful", "type": "success", "session_token": "xyz" }`
**Errors:** `400` missing fields/invalid URL; `401` login failed

#### `GET/POST /api/seer/radarr-servers`
Fetch available Radarr servers configured in Seer (quality profiles, root folders, tags).

**Auth:** None
**Request body (optional):** `SEER_API_URL`, `SEER_TOKEN`, `SEER_SESSION_TOKEN`
**Response:** `200` `{ "servers": [...] }`
**Errors:** `400` not configured/invalid URL

#### `GET/POST /api/seer/sonarr-servers`
Fetch available Sonarr servers configured in Seer (quality profiles, root folders, tags).

**Auth:** None
**Request body (optional):** `SEER_API_URL`, `SEER_TOKEN`, `SEER_SESSION_TOKEN`
**Response:** `200` `{ "servers": [...] }`
**Errors:** `400` not configured/invalid URL

---

## Config (`/api/config`)

#### `GET /api/config/fetch`
Load current configuration in JSON format (real secret values, not redacted).

**Auth:** `@require_role('admin')`
**Response:** `200` flat config dict + `integrations` (raw DB integrations table)

#### `POST /api/config/save`
Save environment variables and sync the DB integrations table. Secrets may be passed as `"***"` to keep the existing value.

**Auth:** `@require_role('admin')`
**Request body:** arbitrary flat config JSON object
**Response:** `200` `{ "message": "Configuration saved successfully!", "status": "success" }`

#### `POST /api/config/reset`
Reset (clear) environment variables.

**Auth:** `@require_role('admin')`
**Response:** `200` `{ "message": "Configuration cleared successfully!", "status": "success" }`

#### `POST /api/config/test-db-connection`
Test database connection with supplied credentials.

**Auth:** `@require_role('admin')`
**Request body:** `DB_TYPE`, `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` (all required)
**Response:** `200` result object from `DatabaseManager.test_connection`
**Errors:** `400` missing keys; `500` connection failed

#### `GET /api/config/sections`
Get available configuration sections.

**Auth:** `@require_role('admin')`
**Response:** `200` `{ "sections": [...], "status": "success" }`

#### `GET /api/config/section/<section_name>`
Get a specific configuration section (real keys, DB-backed values injected).

**Auth:** `@require_role('admin')`
**Response:** `200` `{ "section": "<name>", "data": {...}, "status": "success" }`
**Errors:** `400` invalid section

#### `POST /api/config/section/<section_name>`
Save a specific configuration section and sync DB integrations. Secret fields may be `"***"`.

**Auth:** `@require_role('admin')`
**Request body:** section data (required, non-empty)
**Response:** `200` `{ "message": "Configuration section <name> saved successfully!", "section": "<name>", "status": "success" }`
**Errors:** `400` no data/invalid section

#### `GET /api/config/status`
Get setup completion status.

**Auth:** None (rate-limit exempt)
**Response:** `200` `{ "setup_completed": false, "is_complete": false, "selected_service": "plex", "has_tmdb_key": true, "trakt_app_configured": false, "status": "success" }`

#### `POST /api/config/complete-setup`
Mark setup as completed.

**Auth:** `@require_role('admin')`
**Response:** `200` `{ "message": "Setup marked as completed successfully!", "status": "success" }`
**Errors:** `400` essential configuration missing

#### `GET /api/config/log-level`
Get current log level.

**Auth:** `@require_role('admin')`
**Response:** `200` `{ "log_level": "INFO", "available_levels": ["DEBUG","INFO","WARNING","ERROR"], "status": "success" }`

#### `POST /api/config/log-level`
Set log level.

**Auth:** `@require_role('admin')`
**Request body:** `level` (required; `DEBUG`,`INFO`,`WARNING`,`ERROR`)
**Response:** `200` `{ "message": "Log level set to <LEVEL> successfully!", "log_level": "<LEVEL>", "previous_level": "<OLD>", "status": "success" }`
**Errors:** `400` missing/invalid level

#### `GET /api/config/pool-stats`
Get database connection pool statistics.

**Auth:** `@require_role('admin')`
**Response:** `200` `{ "message": "...", "current_pool": {...}, "all_pools": {...}, "status": "success" }`

#### `POST /api/config/force_run`
Force run the automation script immediately (background execution).

**Auth:** `@require_role('admin')`
**Response:** `200` `{ "message": "Automation script forced successfully!", "status": "success" }`

#### `POST /api/config/test-db`
Test database connection using the current stored configuration.

**Auth:** `@require_role('admin')`
**Response:** `200` result object from `DatabaseManager.test_connection`

#### `GET /api/config/docker-info`
Get Docker container/image tag information.

**Auth:** None
**Response:** `200` `{ "tag": "latest", "build_date": null, "source": "environment|metadata_file|docker_socket|fallback", "status": "success" }`

#### `GET /api/config/docker-digest/<tag>`
Proxy to Docker Hub API to fetch the image digest for a given tag.

**Auth:** None
**Response:** `200` `{ "tag": "<tag>", "digest": "sha256:...", "status": "success" }`
**Errors:** `404` no digest found; `500` Docker Hub request failed

#### `GET /api/config/export`
Export the full application configuration as a portable JSON snapshot.

**Auth:** `@require_role('admin')`
**Query params:** `include_secrets` (default `'false'`)
**Response:** `200` `{ "version": ..., "integrations": {...}, "settings": {...}, "media_users": [...] }`

#### `GET /api/config/openai/user`
Get the current user's personal OpenAI configuration (masked key).

**Auth:** Authenticated user with `can_manage_ai` permission
**Response:** `200` `{ "status": "success", "openai_api_key": "sk-abcd...", "openai_base_url": "..." }`
**Errors:** `403` lacks permission; `404` no user-specific config

#### `POST /api/config/openai/user`
Save the current user's personal OpenAI configuration.

**Auth:** Authenticated user with `can_manage_ai` permission
**Request body:** `openai_api_key` or `openai_base_url` (at least one required)
**Response:** `200` `{ "status": "success", "message": "OpenAI configuration saved" }`
**Errors:** `400` neither field provided; `403` lacks permission

#### `DELETE /api/config/openai/user`
Delete the current user's personal OpenAI configuration (falls back to global config).

**Auth:** Authenticated user with `can_manage_ai` permission
**Response:** `200` `{ "status": "success", "message": "OpenAI configuration deleted. Using global config." }`
**Errors:** `403` lacks permission

#### `POST /api/config/import`
Import a configuration snapshot produced by the export endpoint.

**Auth:** `@require_role('admin')`
**Request body:** matching export format (`version`, `integrations`, `settings`)
**Response:** `200` `{ "message": "Configuration imported successfully", "status": "success" }`
**Errors:** `400` invalid payload

---

## TMDB (`/api/tmdb`)

#### `POST /api/tmdb/cache/clear`
Clear the in-memory TMDB response cache.

**Auth:** `@require_role('admin')`
**Response:** `200` `{ "status": "success", "message": "Cache cleared (<N> entries removed)" }`

#### `POST /api/tmdb/test`
Test a TMDB API key supplied in the request body (not yet saved).

**Auth:** None
**Request body:** `api_key` (required)
**Response:** `200` `{ "status": "success", "message": "TMDB API connection successful!", "data": { "total_results": 0, "total_pages": 0 } }`
**Errors:** `400` missing/invalid key or TMDB error

#### `GET /api/tmdb/popular`
Proxy `GET /movie/popular` from TMDB (cached, uses server-stored API key).

**Auth:** None
**Query params:** `page` (default 1), `include_adult` (default `'false'`)
**Response:** `200` `{ "status": "success", "results": [...], "page": 1, "total_pages": 10 }`
**Errors:** `400` TMDB not configured; `502` upstream error

#### `GET /api/tmdb/genres/movie`
Proxy `GET /genre/movie/list` from TMDB (cached).

**Auth:** None
**Response:** `200` `{ "status": "success", "genres": [...] }`
**Errors:** `400` TMDB not configured; `502` upstream error

#### `GET /api/tmdb/genres/tv`
Proxy `GET /genre/tv/list` from TMDB (cached).

**Auth:** None
**Response:** `200` `{ "status": "success", "genres": [...] }`
**Errors:** `400` TMDB not configured; `502` upstream error

#### `GET /api/tmdb/languages`
Proxy `GET /configuration/languages` from TMDB (cached).

**Auth:** None
**Response:** `200` `{ "status": "success", "languages": [...] }`
**Errors:** `400` TMDB not configured; `502` upstream error

#### `GET /api/tmdb/providers/regions`
Proxy `GET /watch/providers/regions` from TMDB (cached).

**Auth:** None
**Response:** `200` `{ "status": "success", "results": [...] }`
**Errors:** `400` TMDB not configured; `502` upstream error

#### `GET /api/tmdb/providers/movie`
Proxy `GET /watch/providers/movie` from TMDB filtered by region (cached per region).

**Auth:** None
**Query params:** `watch_region` (required, ISO 3166-1 code, e.g. `US`)
**Response:** `200` `{ "status": "success", "results": [...] }`
**Errors:** `400` missing region/TMDB not configured; `502` upstream error

---

## OMDb (`/api/omdb`)

#### `POST /api/omdb/test`
Test an OMDb API key against a stable test title (Interstellar, `tt0816692`).

**Auth:** None
**Request body:** `api_key` (required)
**Response:** `200` `{ "status": "success", "message": "OMDb API key validated successfully!" }`
**Errors:** `400` missing/invalid key or OMDb error

---

## Trakt (`/api/trakt`)

#### `GET /api/trakt/media-users`
Admin: list media-server users with token-safe Trakt link status.

**Auth:** `@require_role('admin')`
**Response:** `200` `{ "media_users": [ { "provider": "plex", "external_user_id": "1", "external_username": "alice", "trakt": { "connected": false } } ] }`

#### `GET /api/trakt/me`
Authenticated user: return own media-profile Trakt status.

**Auth:** Authenticated
**Response:** `200` `{ "media_user": {...} }`
**Errors:** `404` media server account not linked

#### `POST /api/trakt/me/device/code`
Start Trakt device-code OAuth flow for own media profile.

**Auth:** Authenticated
**Response:** `200` Trakt device activation payload (`user_code`, `verification_url`, `device_code`, `expires_in`, `interval`)
**Errors:** `404` profile not linked; `400` credentials not configured/request failed

#### `POST /api/trakt/me/device/token`
Poll Trakt OAuth completion and persist own link.

**Auth:** Authenticated
**Request body:** `device_code` (required)
**Response:** `200` `{ "connected": true, "status": "connected", "trakt_user_id": "123", "trakt_username": "...", "expires_at": "..." }`; `202` `{ "connected": false, "status": "pending" }` while awaiting
**Errors:** `404` profile not linked; `400` missing/expired/denied

#### `POST /api/trakt/media-users/<provider>/<external_user_id>/device/code`
Admin: start Trakt device-code OAuth for a target media user.

**Auth:** `@require_role('admin')`
**Request body (optional):** `client_id`/`TRAKT_CLIENT_ID`, `client_secret`/`TRAKT_CLIENT_SECRET`
**Response:** `200` Trakt device activation payload
**Errors:** `404` media user not found; `400` credentials not configured/failed

#### `POST /api/trakt/media-users/<provider>/<external_user_id>/device/token`
Admin: poll Trakt OAuth completion and persist tokens for a target media user.

**Auth:** `@require_role('admin')`
**Request body:** `device_code` (required), `client_id`/`client_secret` (optional)
**Response:** `200` connected payload; `202` pending
**Errors:** `404` media user not found; `400` missing/expired/denied

#### `DELETE /api/trakt/media-users/<provider>/<external_user_id>`
Admin: unlink the Trakt account associated with a media user.

**Auth:** `@require_role('admin')`
**Response:** `200` `{ "connected": false, "status": "deleted" }`
**Errors:** `404` media user not found

#### `DELETE /api/trakt/me`
Authenticated user: unlink own Trakt account.

**Auth:** Authenticated
**Response:** `200` `{ "connected": false, "status": "deleted" }`
**Errors:** `404` media profile not linked

#### `GET /api/trakt/media-users/<provider>/<external_user_id>/recent`
Admin: preview recent items fetched from a linked Trakt account.

**Auth:** `@require_role('admin')`
**Query params:** `limit` (default 10, clamped 1–50)
**Response:** `200` `{ "status": "success", "provider": "...", "external_user_id": "...", "trakt_username": "...", "items": [...] }`
**Errors:** `404` not found/not linked/tokens unavailable; `400` not configured/failed

#### `GET /api/trakt/me/recent`
Authenticated user: preview recent items fetched from own Trakt account.

**Auth:** Authenticated
**Query params:** `limit` (default 10, clamped 1–50)
**Response:** `200` `{ "status": "success", "provider": "...", "external_user_id": "...", "trakt_username": "...", "items": [...] }`
**Errors:** `404` not linked/tokens unavailable; `400` not configured/failed

#### `PUT /api/trakt/sources/<provider>/<external_user_id>`
Admin: toggle `use_as_seed` / `use_as_exclusion` for a media user's `watched_history` Trakt source.

**Auth:** `@require_role('admin')`
**Request body:** `use_as_seed` (bool, no), `use_as_exclusion` (bool, no) — defaults to existing values if omitted
**Response:** `200` `{ "use_as_seed": true, "use_as_exclusion": true }`
**Errors:** `404` media user not found or no `watched_history` source

---

## AI Search (`/api/ai-search`)

#### `POST /api/ai-search/query`
Execute an AI-powered semantic search for movies/TV.

**Auth:** Authenticated; rate limit 10/min
**Request body:**
| Name | Type | Required | Description |
|---|---|---|---|
| query | str | yes | Natural language description of desired content |
| media_type | str | no | `movie`/`tv`/`both` (default `movie`) |
| user_ids | list | no | Media-server user IDs to scope history |
| max_results | int | no | Default 12 |
| use_history | bool | no | Default true |
| exclude_watched | bool | no | Default true |
| exclude_seen | bool | no | Default false |

**Response:** `200` `{ "status": "success", "results": [...], "ai_reasoning": {...}, "total": 0 }`
**Errors:** `400` missing query/LLM not configured

#### `POST /api/ai-search/request`
Request a media item via Seer, tagged with `tmdb_source_id = 'ai_search'`.

**Auth:** Authenticated; rate limit 20/min
**Request body:** `tmdb_id` (int, required), `media_type` (required, `movie`/`tv`, default `movie`), `rationale` (no), `search_query` (no), `metadata` (dict, no)
**Response:** `200` `{ "status": "success", "message": "Request submitted successfully." }`
**Errors:** `400` missing/invalid fields; `409` already requested/available

#### `GET /api/ai-search/status`
Report whether the LLM is configured and AI search is available.

**Auth:** Authenticated
**Response:** `200` `{ "available": true }` or `{ "available": false, "message": "..." }`

#### `GET /api/ai-search/feedback`
Return all stored like/dislike feedback for AI search results.

**Auth:** Authenticated
**Response:** `200` `{ "status": "success", "feedback": [...] }`

#### `POST /api/ai-search/feedback`
Record a like or dislike for an AI search result.

**Auth:** Authenticated; rate limit 60/min
**Request body:** `tmdb_id` (required), `media_type` (required, `movie`/`tv`), `feedback` (required, `like`/`dislike`), `title` (no), `year` (no)
**Response:** `200` `{ "status": "success" }`
**Errors:** `400` missing/invalid fields

#### `DELETE /api/ai-search/feedback`
Remove feedback for a TMDB item.

**Auth:** Authenticated; rate limit 60/min
**Request body:** `tmdb_id` (required), `media_type` (required)
**Response:** `200` `{ "status": "success" }`

#### `DELETE /api/ai-search/seen`
Clear the already-recommended history (optionally scoped to a media type).

**Auth:** Authenticated; rate limit 30/min
**Request body:** `media_type` (no; omit to clear all)
**Response:** `200` `{ "status": "success", "deleted": 0 }`
**Errors:** `400` invalid media_type

---

## Cleanup (`/api/cleanup`)

#### `GET /api/cleanup/settings`
Fetch current cleanup automation settings.

**Auth:** Authenticated
**Response:** `200` `{ "status": "success", "settings": {...} }`

#### `POST /api/cleanup/settings`
Update cleanup automation settings.

**Auth:** `@require_role('admin')`; rate limit 30/min
**Request body:** `enabled` (bool, no), `dry_run` (bool, no), `grace_days` (int, no, 1–365)
**Response:** `200` `{ "status": "success", "settings": {...} }`
**Errors:** `400` invalid `grace_days`/non-boolean fields

#### `POST /api/cleanup/run`
Trigger an immediate manual cleanup run.

**Auth:** `@require_role('admin')`
**Request body:** `dry_run` (bool, no — overrides configured setting for this run)
**Response:** `200` `{ "status": "success", "result": {...} }`
**Errors:** `400` non-boolean `dry_run`; `409` `{ "code": "already_running" }`

#### `GET /api/cleanup/log`
Retrieve the cleanup audit log.

**Auth:** Authenticated
**Query params:** `limit` (default 100, clamped 1–500)
**Response:** `200` `{ "status": "success", "log": [...] }`

---

## Logs (`/api`)

> This blueprint is registered with `url_prefix='/api'`, so its `/logs` route resolves to `/api/logs`.

#### `GET /api/logs`
Retrieve recent application log lines from `app.log`.

**Auth:** Authenticated
**Query params:** `limit` (default 500, max 2000), `offset` (default 0, max 100000 — number of most-recent lines to skip)
**Response:** `200` JSON array of raw log lines, e.g. `["2026-07-25 10:00:00 INFO ...", "..."]`
**Errors:** None explicit — read failures return an empty array `[]` with 200

---

## Health (`/api/health`)

#### `GET /api/health/live`
Liveness probe — confirms the process is running (no dependency checks).

**Auth:** None
**Response:** `200` `{ "status": "ok" }`

#### `GET /api/health/ready`
Readiness probe — checks DB, TMDB, Seer, and LLM dependencies.

**Auth:** None
**Response:** `200` `{ "status": "ok", "db": "ok", "tmdb": "ok", "seer": "not_configured", "llm": "ok" }` — each of `db`/`tmdb`/`seer`/`llm` is `"ok"`, `"error"`, or `"not_configured"`
**Errors:** `503` if `db` or `tmdb` (critical services) report `"error"`

#### `GET /api/health`
Alias for `/api/health/ready` (same handler).

**Auth:** None
**Response:** same shape as `/api/health/ready`

---

## Integrations (`/api/integrations`)

#### `POST /api/integrations/jellyfin/link`
Link the authenticated user's own Jellyfin account using credentials.

**Auth:** Authenticated
**Request body:** `username` (required), `password` (required)
**Response:** `200` `{ "message": "Jellyfin account linked", "provider": "jellyfin", "external_user_id": "...", "external_username": "..." }`
**Errors:** `400` missing fields; `401` invalid credentials; `502` unreachable/invalid response; `503` not configured

#### `POST /api/integrations/emby/link`
Link the authenticated user's own Emby account (same flow as Jellyfin, against `JELLYFIN_API_URL`).

**Auth:** Authenticated
**Request body:** `username` (required), `password` (required)
**Response:** `200` `{ "message": "Emby account linked", "provider": "emby", "external_user_id": "...", "external_username": "..." }`
**Errors:** `400` missing fields; `401` invalid credentials; `502` unreachable/invalid response; `503` not configured

#### `POST /api/integrations/plex/link`
Link the authenticated user's own Plex account via `plex.tv/users/sign_in.json`.

**Auth:** Authenticated
**Request body:** `username` (required, Plex username or email), `password` (required)
**Response:** `200` `{ "message": "Plex account linked", "provider": "plex", "external_user_id": "...", "external_username": "..." }`
**Errors:** `400` missing fields; `401` invalid credentials; `502` unreachable/invalid response
