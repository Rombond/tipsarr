# Tipsarr

Self-hosted media request + discovery app: one Go backend, one Svelte frontend. Rewrite in progress on branch `rewrite`; design docs live in the Obsidian vault (`Serveur/Dev/Tipsarr/`: Architecture, API, Data Model, Decisions, Feature Plan).

- `backend/` Go (chi + huma + bun). See `backend/CLAUDE.md`.
- `frontend/` SvelteKit SPA. See `frontend/CLAUDE.md`.
- `make dev | test | generate | build`.
- License MIT. **Never copy code from Boxarr (GPL-3)**; it is a behaviour reference only. Seerr and SuggestArr (MIT) may be adapted with their notice kept. Local reference clones live in `~/Documents/Dev/arrStack/` (outside this repo).
- Product rules: never auto-request, never auto-add; Jellyfin is the only identity; webhook-only notifications.
