# Demo gateway

One Tipsarr sandbox per visitor; architecture and settings in `deployments/demo/README.md`.

- A session is created only for a browser page load (`GET` with `Accept: text/html`, outside `/api/`); API calls without a session get 401 `demo_expired`.
- New sessions are capped per client IP (`DEMO_SESSIONS_PER_IP`, default 5 per hour). Test locally (`go run ./cmd/demo-gateway`, `gateway_test.go`); never curl the public demo, it burns the IP quota.
- The image runs as 65532:65532: a mounted data dir must be owned by 65532.
