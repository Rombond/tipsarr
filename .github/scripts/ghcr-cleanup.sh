#!/usr/bin/env bash
# Deletes container versions nothing uses any more: untagged, older than MIN_AGE_DAYS, and not referenced by any
# tagged image. A multi-arch image (or one with a provenance attestation) is an index whose per-platform and
# attestation manifests are untagged versions of their own: they are kept as long as a tag points at the index.
# Usage: OWNER=rombond PACKAGE=tipsarr [DRY_RUN=1] [MIN_AGE_DAYS=2] ghcr-cleanup.sh   (needs gh, jq, curl)
set -euo pipefail
: "${OWNER:?}" "${PACKAGE:?}"
DRY_RUN=${DRY_RUN:-0}
MIN_AGE_DAYS=${MIN_AGE_DAYS:-2}
owner=${OWNER,,}

kind=$(gh api "users/$OWNER" --jq .type)
[ "$kind" = Organization ] && base="orgs/$OWNER" || base="users/$OWNER"
versions=$(gh api --paginate "$base/packages/container/$PACKAGE/versions" --jq '.[] | {id, name, updated_at, tags: .metadata.container.tags}' | jq -s .)

# registry token (public package: anonymous pull is enough)
token=$(curl -fsS "https://ghcr.io/token?scope=repository:$owner/$PACKAGE:pull" | jq -r .token)
accept='application/vnd.oci.image.index.v1+json,application/vnd.docker.distribution.manifest.list.v2+json,application/vnd.oci.image.manifest.v1+json,application/vnd.docker.distribution.manifest.v2+json'

keep=$(mktemp)
for digest in $(jq -r '.[] | select(.tags | length > 0) | .name' <<<"$versions"); do
  echo "$digest" >>"$keep"
  curl -fsS -H "Authorization: Bearer $token" -H "Accept: $accept" \
    "https://ghcr.io/v2/$owner/$PACKAGE/manifests/$digest" | jq -r '.manifests[]?.digest' >>"$keep"
done

cutoff=$(date -u -d "-$MIN_AGE_DAYS days" +%Y-%m-%dT%H:%M:%SZ)
deleted=0 kept=0
while IFS=$'\t' read -r id digest updated; do
  if grep -qx "$digest" "$keep" || [[ "$updated" > "$cutoff" ]]; then kept=$((kept + 1)); continue; fi
  if [ "$DRY_RUN" = 1 ]; then echo "would delete $PACKAGE $digest ($updated)"
  else gh api -X DELETE "$base/packages/container/$PACKAGE/versions/$id" >/dev/null && echo "deleted $PACKAGE $digest ($updated)"; fi
  deleted=$((deleted + 1))
done < <(jq -r '.[] | select(.tags | length == 0) | [.id, .name, .updated_at] | @tsv' <<<"$versions")
echo "$PACKAGE: $(jq length <<<"$versions") versions, $(grep -c . "$keep") referenced by tags, $deleted to delete, $kept untagged kept"
