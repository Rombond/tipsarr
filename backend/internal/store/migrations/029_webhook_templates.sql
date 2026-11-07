-- Webhooks v2: a Liquid template for the body (empty = the raw JSON envelope), the preset it came from,
-- extra headers (JSON list of {name, value}), authentication and "one delivery per recipient".
ALTER TABLE webhooks ADD COLUMN preset VARCHAR(32) NOT NULL DEFAULT '';
ALTER TABLE webhooks ADD COLUMN template TEXT;
ALTER TABLE webhooks ADD COLUMN headers TEXT;
ALTER TABLE webhooks ADD COLUMN auth_type VARCHAR(16) NOT NULL DEFAULT '';
ALTER TABLE webhooks ADD COLUMN auth_username VARCHAR(191) NOT NULL DEFAULT '';
ALTER TABLE webhooks ADD COLUMN auth_secret VARCHAR(512) NOT NULL DEFAULT '';
ALTER TABLE webhooks ADD COLUMN per_recipient INT NOT NULL DEFAULT 0;
