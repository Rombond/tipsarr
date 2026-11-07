# Contributing to Tipsarr

Thanks for helping. Bug reports, ideas, translations, documentation and code are all welcome.

- **Bugs and ideas**: open an [issue](https://github.com/Rombond/tipsarr/issues/new/choose) with the forms. Try to
  reproduce on the [demo](https://tipsarr-demo.brebond.com) first; it often tells whether the problem is in Tipsarr or in
  the setup.
- **Security problems**: never in a public issue, see [SECURITY.md](SECURITY.md).
- **Translations**: see [CONTRIBUTING-TRANSLATIONS.md](CONTRIBUTING-TRANSLATIONS.md).
- **Larger changes**: open an issue first to agree on the approach.

By contributing you agree that your work is released under the [MIT license](LICENSE) and you follow the
[code of conduct](CODE_OF_CONDUCT.md).

## Product rules

Pull requests that break these are declined:

- Never request or add anything to Radarr/Sonarr automatically. Every request is a person's action.
- Dry-run is on by default; every write to Radarr/Sonarr goes through the one dry-run-aware client
  (`servarr.Client.send`).
- Jellyfin is the only identity (password or OpenID Connect, both end in a Jellyfin account).
- Do not copy code from GPL projects (Boxarr). Seerr and SuggestArr are MIT: adapted code keeps their notice.

## Development setup

Requirements: Go 1.26+, Node 22+ (CI uses Node 25). Xcode for iOS, Android Studio for Android.

```bash
make dev        # backend on :8080 + Vite dev server on :5173 (proxies /api)
```

A full local environment with fake Jellyfin, TMDB, Radarr, Sonarr and box office (no real services or keys needed),
sample accounts and data:

```bash
dev/run.sh            # web app on http://localhost:8088, accounts alice/bob/carol/dave/erin, password "test"
dev/run.sh --reset    # start from fresh data
dev/stop.sh
```

See [dev/README.md](dev/README.md) for what each account tests and the fake services' behaviour.

## Layout

- `backend/`: Go ([chi](https://github.com/go-chi/chi), [huma](https://huma.rocks), [bun](https://bun.uptrace.dev)).
  Migrations are portable SQL in `backend/internal/store/migrations` (SQLite, PostgreSQL, MySQL).
- `frontend/`: SvelteKit SPA (Svelte 5, Tailwind 4, shadcn-svelte), embedded in the Go binary.
- `ios/`, `android/`: native apps. `api/openapi.yaml`: the generated API spec they use.
- `locales/`: every text, shared by the web app, the server and the apps.
- `design/`: tokens, icons and the scripts that generate the app strings and tokens.
- `trailer-resolver/`: optional sidecar for Teasers. `deployments/`: Compose files, including the demo.

## Tests and generated files

```bash
make test       # go test ./... + svelte-check + i18n check (what CI runs)
make lint       # go vet ./...
gofmt -l backend   # must print nothing
make generate   # after any API change
```

`make generate` re-exports the OpenAPI spec, regenerates the frontend types, the app copies of the spec, tokens and
strings. CI fails if they drifted, so commit its output with the API change.

PostgreSQL and MySQL tests run when `TIPSARR_TEST_PG_URL` / `TIPSARR_TEST_MYSQL_URL` are set (CI uses service
containers).

## Pull requests

1. Fork, create a branch from `master`.
2. Keep a PR to one topic. Add or update tests, update `docs/` when behaviour changes.
3. New strings: add them to `locales/en/` (and `fr/` if you can); never hard-code user-facing text.
4. `make test` and `make generate` must leave a clean tree.
5. Fill in the pull request template.

## Commit messages

Short imperative subject, scope first, lower case; a body explains why when it is not obvious. Examples from the
history:

```
feat(push): open registration with the push relay
fix(issues): refuse a problem report on a title not in the library
i18n(fr): tutoiement, unified glossary, French typography
refactor(devenv): extract the fake Jellyfin into internal/fakejf
```

Common scopes: `feat`, `fix`, `refactor`, `docs`, `build`, `i18n`, with the area in parentheses (`web`, `push`,
`webhooks`, `demo`…).
