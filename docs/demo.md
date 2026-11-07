# Demo

**https://tipsarr-demo.brebond.com**

Each visitor gets a Tipsarr of their own, started from a seeded template, with admin rights. You can change every
setting, approve requests, create users and webhooks: only you see it. The sandbox is deleted after 30 minutes without
activity, or 60 minutes after it started. The login page offers "Sign in as admin" and "Sign in as user".

What is real and what is not:

- Movie and show data, posters and box-office charts are real (TMDB, Box Office Mojo), through a shared cache.
- Jellyfin, Radarr and Sonarr are fakes with a sample library, four accounts and watch history.
- Dry-run is forced: nothing is downloaded. Webhooks can be written and previewed but are never sent; push, LDAP, single
  sign-on and the webhook test answer `demo_disabled`. The TMDB and Jellyfin keys cannot be changed.
- Each sandbox may only connect to the demo gateway; it cannot reach the internet.

## Running your own

The demo is the `demo-gateway` program (`backend/cmd/demo-gateway`), which starts one Tipsarr process per visitor
behind one address. Instructions, architecture and all `DEMO_*` settings:
[`deployments/demo/README.md`](../deployments/demo/README.md).

```bash
docker build -f Dockerfile.demo -t tipsarr-demo .
TMDB_API_KEY=... docker compose -f deployments/demo/docker-compose.yml up -d
```

Put a TLS reverse proxy in front of port 8080 and keep port 9090 (health and metrics) private. A sandbox uses about
160 MiB; size `DEMO_MAX_SESSIONS` to your memory.
