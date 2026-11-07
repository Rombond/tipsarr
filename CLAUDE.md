# Tipsarr

Self-hosted media discovery and requests for Jellyfin: Go backend, Svelte web app, native iOS and Android apps (all ship 1.0).

- `backend/` Go (chi + huma + bun), see `backend/CLAUDE.md`. `frontend/` SvelteKit SPA, see `frontend/CLAUDE.md`. `ios/` SwiftUI, see `ios/CLAUDE.md`. `android/` Kotlin/Compose, see `android/CLAUDE.md`.
- `api/openapi.yaml` is the generated spec the apps use. `design/` holds tokens, string build scripts, icons, Penpot scripts. `docs/` is the user documentation (update it with any user-visible change). `fastlane/metadata/` and `store/` hold store listings, review notes and `store/RELEASE-CHECKLIST.md`.
- `make dev | test | generate | build`. `make test` is what CI runs (plus `gofmt`, `go vet`). `TestHealthAndOpenAPI` fails locally when `backend/internal/web/dist` holds a built frontend (the SPA answers 200 instead of 404); it passes in CI.
- License MIT. Do not copy code from GPL projects (Boxarr); Seerr and SuggestArr are MIT and may be adapted with their notice kept.

## Product rules
- Never auto-request and never auto-add anything to Radarr/Sonarr. Dry-run is on by default and every write to Radarr/Sonarr goes through `servarr.Client.send`.
- Jellyfin is the only identity (password or OIDC single sign-on, both end in a Jellyfin account).
- Notifications: outgoing webhooks (`backend/internal/notify`: Liquid templates, presets) and mobile push through the relay (`backend/internal/push`; relay is the separate repo Rombond/push-relay, a public instance with open registration holds the APNs/FCM credentials).
- Public demo: `TIPSARR_DEMO_MODE` (`backend/internal/demo`, fake Jellyfin `internal/fakejf`), gateway `backend/cmd/demo-gateway`, deployment `deployments/demo/` (see its README and `backend/cmd/demo-gateway/CLAUDE.md`).

## Generated code and strings
- Every API change: run `make generate` (CI fails if the TS client, `api/openapi.yaml`, `design/generated/` or the app copies drifted). It also rebuilds tokens and strings.
- Strings: source is `locales/<lang>/*.json` (English is the source). `node design/build-strings.mjs` (run by `make generate`) produces web, server, iOS and Android strings; the iOS catalogs are copied by `make ios-strings` only. Never edit generated string files.
- Web `t()` treats any message containing ` | ` as a plural (first form for one, second otherwise): never use ` | ` as literal text in a string.
- French copy: tutoiement; non-breaking space before `:` `;` `?` `!`; feminine in parentheses (`connecté(e)`).

## Release
`master` is protected: merge through a PR, then tag `vX.Y.Z` on master; the tag triggers `.github/workflows/release.yml` (image). Create the GitHub Release with notes. Never push or tag without an explicit ask.
