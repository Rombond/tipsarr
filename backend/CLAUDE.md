# Backend (Go)

Module `github.com/Rombond/tipsarr/backend`. Run: `go run ./cmd/tipsarr` (env vars are listed in the README). `go run ./cmd/tipsarr openapi` prints the OpenAPI spec.

Layout rules:
- `internal/api` handlers are thin (huma operations, under `/api/v1`); logic goes in service packages.
- Only `internal/clients/*` call external services; only `internal/store` touches the DB.
- Migrations: `internal/store/migrations/NNN_name.sql`, **portable SQL** (runs on SQLite, PostgreSQL, MySQL): no AUTOINCREMENT/SERIAL, timestamps are unix-second BIGINT, ids are app-generated VARCHAR, `VARCHAR(191)` for indexed/PK text.
- DB URL: `sqlite:<path>` (default, `<config>/tipsarr.db`), `postgres://…`, `mysql://user:pass@host:3306/db`.
- Auth = a Jellyfin account, by password or by OIDC single sign-on (matched to a Jellyfin user by username) → own session cookie `tipsarr_session` (sha256 of token stored).
- Tests: `go test ./...`. External engines: set `TIPSARR_TEST_PG_URL` / `TIPSARR_TEST_MYSQL_URL` (see store_test.go).
- After changing API types run `make generate` at repo root (updates the frontend TS client).
