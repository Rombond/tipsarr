# Demo gateway

One Tipsarr sandbox per visitor; architecture and settings in `deployments/demo/README.md`.

- A session is created only for a browser page load (`GET` with `Accept: text/html`, outside `/api/`) or a successful app sign-in (`POST /api/v1/auth/token` or `/auth/login`, whose returned token is then mapped to the session for `Authorization: Bearer`). Pre-login calls (`/status`, `/health`, `/auth/methods`, `/setup/status`) are answered by a warm sandbox without allocating one. Other API calls without a session get 401 `demo_expired`.
- Demo accounts: `admin` / `demo` and `user` / `demo` (seed in `seed.go`; bump the seed version when the seed changes so the template rebuilds).
- New sessions are capped per client IP (`DEMO_SESSIONS_PER_IP`, default 5 per hour). Test locally (`go run ./cmd/demo-gateway`, `gateway_test.go`); never curl the public demo, it burns the IP quota.
- The image runs as 65532:65532: a mounted data dir must be owned by 65532.
