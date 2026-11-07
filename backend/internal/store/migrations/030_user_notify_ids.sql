-- Per-person chat ids that webhook templates can use to mention or message someone directly.
ALTER TABLE users ADD COLUMN discord_id VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN telegram_chat_id VARCHAR(64) NOT NULL DEFAULT '';
