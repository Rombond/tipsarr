-- Where a person's admin role came from, so a later sign-in can take back only what a sign-in gave:
-- '' (not admin, or an admin from before this column: kept until the next sign-in decides), 'jellyfin' (a
-- Jellyfin administrator), 'oidc' (the single sign-on admin group), 'manual' (set by an admin in Tipsarr).
ALTER TABLE users ADD COLUMN admin_source VARCHAR(16) NOT NULL DEFAULT '';

-- Failed sign-in counters (per username and per client address), kept across restarts.
CREATE TABLE login_throttle (
    throttle_key VARCHAR(191) NOT NULL PRIMARY KEY,
    failures     INTEGER      NOT NULL,
    locked_until BIGINT       NOT NULL,
    updated_at   BIGINT       NOT NULL
);
