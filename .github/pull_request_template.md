## What and why

<!-- What does this change, and why? Link the issue: "Fixes #123". -->

## How it was tested

<!-- Commands run, screens checked (web, iOS, Android), databases (SQLite / PostgreSQL / MySQL). -->

## Checklist

- [ ] `make test` passes, `gofmt -l backend` prints nothing
- [ ] `make generate` run after API changes, generated files committed
- [ ] New user-facing text is in `locales/en/` (and `fr/` if possible), not hard-coded
- [ ] `docs/` updated if behaviour or configuration changed
- [ ] Nothing is requested or sent to Radarr/Sonarr automatically; writes go through the dry-run-aware client
- [ ] No code copied from GPL projects
