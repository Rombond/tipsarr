# Tipsarr

Self-hosted media discovery and requests for Jellyfin: one Go backend, one Svelte frontend.

- `backend/` Go (chi + huma + bun). See `backend/CLAUDE.md`.
- `frontend/` SvelteKit SPA (Svelte 5, Tailwind 4, shadcn-svelte). See `frontend/CLAUDE.md`.
- `make dev | test | generate | build`. `make test` is what CI runs (plus `gofmt`, `go vet`).
- License MIT. Do not copy code from GPL projects (Boxarr); Seerr and SuggestArr are MIT and may be adapted with their notice kept.
- Product rules: never auto-request and never auto-add anything to Radarr/Sonarr; Jellyfin is the only identity (password or OIDC single sign-on, both end in a Jellyfin account); notifications are outgoing webhooks only; dry-run is on by default and every write to Radarr/Sonarr goes through `servarr.Client.send`.
- Every API change: run `make generate` (CI fails if the generated client drifted).
