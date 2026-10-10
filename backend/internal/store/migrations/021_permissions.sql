-- Permissions (Seerr style): one integer mask per person, and per-person request quotas that override the defaults.
-- 4980768 = request + request movies + request shows + create issues, what a person could do before this change.

ALTER TABLE users ADD COLUMN permissions BIGINT NOT NULL DEFAULT 4980768;
ALTER TABLE users ADD COLUMN movie_quota_limit INT;
ALTER TABLE users ADD COLUMN movie_quota_days INT;
ALTER TABLE users ADD COLUMN tv_quota_limit INT;
ALTER TABLE users ADD COLUMN tv_quota_days INT;

-- The old "users may choose the root folder" setting becomes the "advanced requests" permission (8192).
UPDATE users SET permissions = permissions | 8192
    WHERE EXISTS (SELECT 1 FROM settings WHERE skey = 'requests.user_folder' AND svalue = 'true');
INSERT INTO settings (skey, svalue)
    SELECT 'permissions.default', '4988960' WHERE EXISTS (SELECT 1 FROM settings WHERE skey = 'requests.user_folder' AND svalue = 'true');
DELETE FROM settings WHERE skey = 'requests.user_folder';
