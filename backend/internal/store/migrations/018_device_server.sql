-- The address the app uses to reach this server, sent with every push so the app opens the right account.

ALTER TABLE devices ADD COLUMN server VARCHAR(255) NOT NULL DEFAULT '';
