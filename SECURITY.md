# Security policy

## Supported versions

| Version | Supported |
|---|---|
| latest 1.x release | yes |
| older releases | no, please upgrade |

Fixes are released as a new version of the server image (`ghcr.io/rombond/tipsarr`) and, when needed, of the apps.

## Reporting a vulnerability

Please **do not open a public issue**. Report it privately through GitHub:
[Security → Report a vulnerability](https://github.com/Rombond/tipsarr/security/advisories/new).

Include the affected version, how to reproduce, and the impact you see. You will get an answer as soon as possible;
this is a hobby project maintained in spare time, so please allow a reasonable delay before any disclosure. Credit is
given in the advisory unless you prefer otherwise.

Problems with the push relay go to [Rombond/push-relay](https://github.com/Rombond/push-relay/security/advisories/new).

## Scope

In scope: the Tipsarr server, web interface, iOS and Android apps, the trailer-resolver and demo-gateway programs, and
the public push relay.

Out of scope: the public demo's sandboxes themselves (each visitor is an admin of a throwaway copy by design), unless you
can reach another visitor's sandbox, the host or the internet from a sandbox; denial of service by volume; issues in
Jellyfin, Radarr, Sonarr or your reverse proxy.

## Hardening tips

- Keep Tipsarr behind HTTPS; set `TIPSARR_COOKIE_SECURE=true` when it is only served over HTTPS.
- Use `TIPSARR_SETUP_TOKEN=true` if a fresh install is reachable from the internet before setup.
- Treat the Jellyfin webhook URL and the read-only API key as secrets.
- Run the container with the [recommended hardening](docs/installation.md#hardening-recommended) (`cap_drop: [ALL]`,
  `no-new-privileges`, read-only root filesystem) and keep the config folder at mode `700`.
- Secrets stored in the database are encrypted with the key in `<config>/secret.key` (or `TIPSARR_SECRET_KEY`): back
  it up with the database, but store it apart from database dumps you share or ship off-site.

## Verifying the image

Release images are built by GitHub Actions from a version tag, carry an SBOM and SLSA provenance attestation, and are
signed with keyless [cosign](https://docs.sigstore.dev/) (no long-lived key: the signature is tied to the release
workflow of this repository). Verify a digest before deploying it:

```bash
cosign verify ghcr.io/rombond/tipsarr@sha256:<digest> \
  --certificate-identity-regexp '^https://github.com/Rombond/tipsarr/\.github/workflows/release\.yml@refs/tags/v' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

Inspect the SBOM and provenance with
`docker buildx imagetools inspect ghcr.io/rombond/tipsarr:<version> --format '{{ json .SBOM }}'` (or `.Provenance`).
GitHub Actions in the workflows are pinned by commit SHA and the base images by digest; Dependabot proposes the updates.
