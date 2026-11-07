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

Migrations only go forward. To go back to an older version, restore the backup made before upgrading; an older Tipsarr
does not check the schema version and may misbehave on a database migrated by a newer one.

## Pinning a version

`:latest` follows the newest release. Each release is also tagged `X.Y.Z` and `X.Y` (for example
`ghcr.io/rombond/tipsarr:1.0.0` or `:1.0`), to choose when to upgrade.

## Mobile apps

Update the server first, then the apps. Apps keep working with an older server but hide what it does not support.
