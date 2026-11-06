-- 4K requests (Seerr's bits). Everyone keeps 4K: request 4K (1024) + movies (2048) + shows (4096) for every person,
-- and whoever auto-approves keeps auto-approving in 4K: 128 -> 32768, 256 -> 65536, 512 -> 131072.
-- The stored default (settings 'permissions.default', a text value) is updated in Go: see afterMigration in store.go.

UPDATE users SET permissions = permissions | 7168;
UPDATE users SET permissions = permissions | 32768 WHERE (permissions & 128) <> 0;
UPDATE users SET permissions = permissions | 65536 WHERE (permissions & 256) <> 0;
UPDATE users SET permissions = permissions | 131072 WHERE (permissions & 512) <> 0;

-- Whether the request's quality profile allows 4K (2160p), decided when it is made and when it is approved.
ALTER TABLE requests ADD COLUMN uhd INT NOT NULL DEFAULT 0;
