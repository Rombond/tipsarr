# Upgrading

```bash
docker compose pull && docker compose up -d
```

Database migrations run automatically at start. Read the [release notes](https://github.com/Rombond/tipsarr/releases)
before a major version.

## Back up first

- **SQLite** (default): stop the container and copy the `/config` folder (it holds `tipsarr.db` and the caches), or copy
  just `tipsarr.db` while stopped.
- **PostgreSQL / MySQL**: `pg_dump` / `mysqldump` the database.

Back up `secret.key` (in the config folder) together with the database: the stored secrets are encrypted with it,
and a database restored without it needs every API key and password entered again. See
[Secrets encryption key](installation.md#secrets-encryption-key).

Migrations only go forward. To go back to an older version, restore the backup made before upgrading; an older Tipsarr
does not check the schema version and may misbehave on a database migrated by a newer one.

## Pinning a version

`:latest` follows the newest release. Each release is also tagged `X.Y.Z` and `X.Y` (for example
`ghcr.io/rombond/tipsarr:1.0.0` or `:1.0`), to choose when to upgrade.

## Mobile apps

Update the server first, then the apps. Apps keep working with an older server but hide what it does not support.

## Version notes

### 0.18: TV seasons follow Sonarr's numbering

TV seasons now come from Sonarr (TVDB numbering), not TMDB (see the [FAQ](faq.md)). Existing requests are not
renumbered. TV requests already sent to Sonarr or imported are marked as TVDB-numbered. The others (pending or
failed) are marked as TMDB-numbered and checked at approval: if the two numberings match, nothing changes; a request
for every TMDB season becomes every Sonarr season; otherwise approval fails with a message asking to request every
season or pick the season in Sonarr. Nothing wrong is ever sent to Sonarr.

To list the requests that will go through this check (read-only):

```sql
SELECT id, title, status FROM requests
WHERE media_type = 'tv' AND season_numbering = 'tmdb' AND status IN ('pending', 'failed');
```

Requests already in Sonarr under a wrong season are not changed: fix them in Sonarr.
